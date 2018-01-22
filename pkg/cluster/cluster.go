package cluster

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/cache"
	"github.com/kinvolk/kube-spawn/pkg/machinectl"
	"github.com/kinvolk/kube-spawn/pkg/multiprint"
	"github.com/kinvolk/kube-spawn/pkg/nspawntool"
	"github.com/kinvolk/kube-spawn/pkg/utils"
	"github.com/kinvolk/kube-spawn/pkg/utils/fs"
)

type ClusterSettings struct {
	CNIPluginDir          string
	ContainerRuntime      string
	HyperkubeImage        string
	KubernetesSourceDir   string
	KubernetesVersion     string
	RuntimeEndpoint       string
	RktBinaryPath         string
	RktStage1ImagePath    string
	RktletBinaryPath      string
	UseLegacyCgroupDriver bool
}

type Cluster struct {
	dir  string
	name string
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

const (
	validNameRegexpStr = "^[a-zA-Z0-9-]{1,50}$"
	weaveNet           = "https://github.com/weaveworks/weave/releases/download/v2.0.5/weave-daemonset-k8s-1.7.yaml"

	// Avoid token passing between master and worker nodes by using
	// a hard-coded token
	kubeadmToken = "aaaaaa.bbbbbbbbbbbbbbbb"
)

var validNameRegexp = regexp.MustCompile(validNameRegexpStr)

func ValidName(name string) bool {
	return validNameRegexp.MatchString(name)
}

func New(dir, name string) (*Cluster, error) {
	if !ValidName(name) {
		return nil, fmt.Errorf("got invalid cluster name %q (expected %q)", name, validNameRegexpStr)
	}
	return &Cluster{
		dir:  dir,
		name: name,
	}, nil
}

func validateClusterSettings(clusterSettings *ClusterSettings) error {
	if clusterSettings.KubernetesVersion == "" && (clusterSettings.HyperkubeImage == "" || clusterSettings.KubernetesSourceDir == "") {
		return fmt.Errorf("either kubernetes version or hyperkube image and kubernetes source dir must be given")
	}
	if clusterSettings.ContainerRuntime != "docker" && clusterSettings.ContainerRuntime != "rkt" {
		return fmt.Errorf("unsupported container runtime given: %s", clusterSettings.ContainerRuntime)
	}
	return nil
}

// Create creates a new kube-spawn cluster environment. It does..
//
// * Validate the cluster settings
// * Download the required files into the cache
// * Generate the shared, readonly rootfs directory structure (later
//   mounted via overlayfs)
// * Copy the required files from the source directories and cache to
//   the target locations
func (c *Cluster) Create(clusterSettings *ClusterSettings, clusterCache *cache.Cache) error {
	if err := validateClusterSettings(clusterSettings); err != nil {
		return err
	}
	if clusterCache == nil {
		return fmt.Errorf("no cache given but required")
	}

	cacheDirKubernetes := path.Join(clusterCache.Dir(), "kubernetes")

	if clusterSettings.KubernetesSourceDir == "" {
		if err := bootstrap.DownloadKubernetesBinaries(clusterSettings.KubernetesVersion, cacheDirKubernetes); err != nil {
			log.Fatalf("Failed to download required Kubernetes binaries: %v", err)
		}
	}

	if err := bootstrap.DownloadSocatBin(clusterCache.Dir()); err != nil {
		log.Fatalf("Failed to download `socat` into cache dir: %v", err)
	}

	if err := os.MkdirAll(c.BaseRootfsPath(), 0755); err != nil {
		log.Fatalf("Failed to create directory %q: %v", c.BaseRootfsPath(), err)
	}

	log.Print("Generating configuration files from templates ...")

	clusterSettings.UseLegacyCgroupDriver = clusterSettings.ContainerRuntime == "docker"
	if clusterSettings.ContainerRuntime == "rkt" {
		clusterSettings.RuntimeEndpoint = "unix:///var/run/rktlet.sock"
	}

	rootfsDir := c.BaseRootfsPath()
	if err := fs.CreateFileFromBytes(path.Join(rootfsDir, "/etc/docker/daemon.json"), []byte(DockerDaemonConfig)); err != nil {
		return err
	}
	if err := fs.CreateFileFromBytes(path.Join(rootfsDir, "/etc/systemd/system/docker.service.d/20-kube-spawn-dropin.conf"), []byte(DockerSystemdDropin)); err != nil {
		return err
	}
	if err := fs.CreateFileFromBytes(path.Join(rootfsDir, "/etc/resolv.conf"), []byte("nameserver 8.8.8.8")); err != nil {
		return err
	}
	if clusterSettings.ContainerRuntime == "rkt" {
		if err := fs.CreateFileFromBytes(path.Join(rootfsDir, "/etc/systemd/system/rktlet.service"), []byte(RktletSystemdUnit)); err != nil {
			return err
		}
	}
	if err := fs.CreateFileFromBytes(path.Join(rootfsDir, "/etc/systemd/network/50-weave.network"), []byte(WeaveSystemdNetworkdConfig)); err != nil {
		return err
	}

	buf, err := ExecuteTemplate(KubespawnBootstrapScriptTmpl, clusterSettings)
	if err != nil {
		return err
	}
	if err := fs.CreateFileFromReader(path.Join(rootfsDir, "/opt/kube-spawn/bootstrap.sh"), &buf); err != nil {
		return err
	}

	buf, err = ExecuteTemplate(KubeletSystemdDropinTmpl, clusterSettings)
	if err != nil {
		return err
	}
	if err := fs.CreateFileFromReader(path.Join(rootfsDir, "/etc/systemd/system/kubelet.service.d/20-kube-spawn-dropin.conf"), &buf); err != nil {
		return err
	}

	buf, err = ExecuteTemplate(KubeadmConfigTmpl, clusterSettings)
	if err != nil {
		return err
	}
	if err := fs.CreateFileFromReader(path.Join(rootfsDir, "/etc/kubeadm/kubeadm.yml"), &buf); err != nil {
		return err
	}

	log.Print("Copying files for cluster ...")

	var (
		kubeletPath        string
		kubeadmPath        string
		kubectlPath        string
		kubeletServicePath string
		kubeadmDropinPath  string
	)
	if clusterSettings.KubernetesSourceDir != "" {
		// If Docker was used to build Kubernetes (`build/run.sh make`),
		// the binaries would be in `"_output/dockerized/bin/linux/amd64`,
		// look their first. If we don't find them there, try with
		// `_output/bin`
		kubernetesSourceBinaryDir := path.Join(clusterSettings.KubernetesSourceDir, "_output/dockerized/bin/linux/amd64")

		kubeadmPath = path.Join(kubernetesSourceBinaryDir, "kubeadm")
		if exists, err := fs.PathExists(kubeadmPath); err != nil {
			log.Fatalf("Failed to stat %q: %v", kubeadmPath, err)
		} else if !exists {
			kubernetesSourceBinaryDir = path.Join(clusterSettings.KubernetesSourceDir, "_output/bin")

			kubeadmPath = path.Join(kubernetesSourceBinaryDir, "kubeadm")
			if exists, err := fs.PathExists(kubeadmPath); err != nil {
				log.Fatalf("Failed to stat %q: %v", kubernetesSourceBinaryDir, err)
			} else if !exists {
				log.Fatalf("Cannot find expected `_output` directory in %q", clusterSettings.KubernetesSourceDir)
			}

		}

		kubeletPath = path.Join(kubernetesSourceBinaryDir, "kubelet")
		kubectlPath = path.Join(kubernetesSourceBinaryDir, "kubectl")

		kubernetesSourceBuildDir := path.Join(clusterSettings.KubernetesSourceDir, "build")

		kubeletServicePath = path.Join(kubernetesSourceBuildDir, "debs/kubelet.service")
		kubeadmDropinPath = path.Join(kubernetesSourceBuildDir, "rpms/10-kubeadm.conf")
	} else {
		kubeletPath = path.Join(cacheDirKubernetes, clusterSettings.KubernetesVersion, "kubelet")
		kubeadmPath = path.Join(cacheDirKubernetes, clusterSettings.KubernetesVersion, "kubeadm")
		kubectlPath = path.Join(cacheDirKubernetes, clusterSettings.KubernetesVersion, "kubectl")

		kubeletServicePath = path.Join(cacheDirKubernetes, clusterSettings.KubernetesVersion, "kubelet.service")
		kubeadmDropinPath = path.Join(cacheDirKubernetes, clusterSettings.KubernetesVersion, "10-kubeadm.conf")
	}

	type copyItem struct {
		dst string
		src string
	}
	var copyItems []copyItem

	// copyItem destinations must be relative to the cluster rootfs path
	// We will prepend it later

	copyItems = append(copyItems, copyItem{dst: "/usr/bin/kubelet", src: kubeletPath})
	copyItems = append(copyItems, copyItem{dst: "/usr/bin/kubeadm", src: kubeadmPath})
	copyItems = append(copyItems, copyItem{dst: "/usr/bin/kubectl", src: kubectlPath})
	copyItems = append(copyItems, copyItem{dst: "/etc/systemd/system/kubelet.service", src: kubeletServicePath})
	copyItems = append(copyItems, copyItem{dst: "/etc/systemd/system/kubelet.service.d/10-kubeadm.conf", src: kubeadmDropinPath})

	socatPath := path.Join(clusterCache.Dir(), "socat")
	copyItems = append(copyItems, copyItem{dst: "/usr/bin/socat", src: socatPath})

	copyItems = append(copyItems, copyItem{dst: "/opt/cni/bin/bridge", src: path.Join(clusterSettings.CNIPluginDir, "bridge")})
	copyItems = append(copyItems, copyItem{dst: "/opt/cni/bin/loopback", src: path.Join(clusterSettings.CNIPluginDir, "loopback")})

	if clusterSettings.ContainerRuntime == "rkt" {
		copyItems = append(copyItems, copyItem{dst: "/usr/bin/rkt", src: clusterSettings.RktBinaryPath})
		copyItems = append(copyItems, copyItem{dst: "/usr/bin/stage1-coreos.aci", src: clusterSettings.RktStage1ImagePath})
		copyItems = append(copyItems, copyItem{dst: "/usr/bin/rktlet", src: clusterSettings.RktletBinaryPath})
	}

	// TODO: the docker-runc wrapper ensures `--no-new-keyring` is
	// set, otherwise Docker will attempt to use keyring syscalls
	// which are not allowed in systemd-nspawn containers. It can
	// be removed once we require systemd v235 or later. We then
	// will be able to whitelist the required syscalls; see:
	// https://github.com/systemd/systemd/pull/6798
	kubeSpawnRuncPath := "kube-spawn-runc"
	if !utils.IsExecBinary(kubeSpawnRuncPath) {
		if lp, err := exec.LookPath(kubeSpawnRuncPath); err != nil {
			log.Fatal(errors.Wrap(err, "kube-spawn-runc binary not found but required"))
		} else {
			kubeSpawnRuncPath = lp
		}
	}
	copyItems = append(copyItems, copyItem{dst: "/usr/bin/kube-spawn-runc", src: kubeSpawnRuncPath})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var failed bool
	errorChan := make(chan error)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errorChan:
				failed = true
				log.Printf("%v", err)
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(len(copyItems))
	for _, item := range copyItems {
		go func(dst, src string) {
			defer wg.Done()
			dst = path.Join(c.BaseRootfsPath(), dst)
			if copyErr := fs.CopyFile(src, dst); copyErr != nil {
				errorChan <- errors.Wrapf(copyErr, "Failed to copy file %q -> %q", src, dst)
			}
		}(item.dst, item.src)
	}

	wg.Wait()

	if failed {
		return fmt.Errorf("copying necessary files didn't succeed")
	}
	return nil
}

func (c *Cluster) Start(numberNodes int, cniPluginDir string) error {
	if numberNodes < 1 {
		return fmt.Errorf("cannot start less than 1 node")
	}
	if err := bootstrap.PrepareCoreosImage(); err != nil {
		return err
	}

	if err := bootstrap.EnsureRequirements(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var failed bool
	errorChan := make(chan error)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errorChan:
				failed = true
				log.Printf("%v", err)
			}
		}
	}()

	log.Printf("Starting %d nodes in cluster %s ...", numberNodes, c.name)

	// Note: currently only a single master node is supported and
	// the code written with that limitation. Supporting multiple
	// master nodes shouldn't be too much work though (figure out
	// multi master setup with kubeadm + use loadbalancer + use
	// loadbalancer IP from worker nodes)

	var wg sync.WaitGroup
	wg.Add(numberNodes)
	for i := 0; i < numberNodes; i++ {
		go func(nodeNumber int) {
			defer wg.Done()

			var machineNameSuffix string
			if nodeNumber == 0 {
				machineNameSuffix = fmt.Sprintf("master-%s", randString(6))
			} else {
				machineNameSuffix = fmt.Sprintf("worker-%s", randString(6))
			}
			machineName := fmt.Sprintf("kube-spawn-%s-%s", c.name, machineNameSuffix)

			log.Printf("Waiting for machine %s to start up ...", machineName)

			if err := nspawntool.Run("coreos", c.BaseRootfsPath(), path.Join(c.MachineRootfsPath(), machineName), machineName, cniPluginDir); err != nil {
				errorChan <- errors.Wrapf(err, "Failed to start machine %s", machineName)
				return
			}

			log.Printf("Started %s", machineName)
			log.Printf("Bootstrapping %s ...", machineName)

			if err := machinectl.Exec(machineName, "/opt/kube-spawn/bootstrap.sh"); err != nil {
				errorChan <- errors.Wrapf(err, "Failed to bootstrap machine %s", machineName)
			}
		}(i)
	}
	wg.Wait()

	if failed {
		return fmt.Errorf("starting the cluster didn't succeed")
	}

	log.Printf("Cluster %q started", c.name)

	masterMachines, err := c.MasterMachines()
	if err != nil {
		return fmt.Errorf("failed to get list of master machines: %v", err)
	}
	if len(masterMachines) == 0 {
		return fmt.Errorf("no master machines found")
	}

	masterMachine := masterMachines[0]

	log.Println("Note: `kubeadm init` can take several minutes")

	multiPrinter := multiprint.New(ctx)
	multiPrinter.RunPrintLoop()

	if err := kubeadmInit(masterMachine.Name, multiPrinter.NewWriter(fmt.Sprintf("%s ", masterMachine.Name))); err != nil {
		return errors.Wrapf(err, "failed to kubeadm init %q", masterMachine.Name)
	}
	if err := applyNetworkPlugin(masterMachine.Name, multiPrinter.NewWriter(fmt.Sprintf("%s ", masterMachine.Name))); err != nil {
		return err
	}

	adminKubeconfigSource := path.Join(c.MachineRootfsPath(), masterMachine.Name, "etc/kubernetes/admin.conf")
	if err := fs.CopyFile(adminKubeconfigSource, c.AdminKubeconfigPath()); err != nil {
		return err
	}

	workerMachines, err := c.WorkerMachines()
	if err != nil {
		return err
	}

	masterIP := masterMachine.IP

	wg.Add(len(workerMachines))
	for _, worker := range workerMachines {
		go func(nodeName string) {
			defer wg.Done()
			if err := kubeadmJoin(masterIP, nodeName, multiPrinter.NewWriter(fmt.Sprintf("%s ", nodeName))); err != nil {
				errorChan <- errors.Wrapf(err, "Failed to kubeadm join %q", worker.Name)
			}
		}(worker.Name)
	}
	wg.Wait()

	if failed {
		return fmt.Errorf("provisioning the worker nodes with kubeadm didn't succeed")
	}
	return nil
}

func (c *Cluster) AdminKubeconfigPath() string {
	return path.Join(c.dir, "admin.kubeconfig")
}

func (c *Cluster) AdminKubeconfig() (string, error) {
	kubeconfigBytes, err := ioutil.ReadFile(c.AdminKubeconfigPath())
	if err != nil {
		return "", err
	}
	return string(kubeconfigBytes), nil
}

func (c *Cluster) BaseRootfsPath() string {
	return path.Join(c.dir, "rootfs-base-readonly")
}

func (c *Cluster) MachineRootfsPath() string {
	return path.Join(c.dir, "rootfs-machines")
}

func (c *Cluster) MasterMachines() ([]machinectl.Machine, error) {
	return machinectl.ListByRegexp(fmt.Sprintf("^kube-spawn-%s-master-[a-z0-9]+$", c.name))
}

func (c *Cluster) WorkerMachines() ([]machinectl.Machine, error) {
	return machinectl.ListByRegexp(fmt.Sprintf("^kube-spawn-%s-worker-[a-z0-9]+$", c.name))
}

func (c *Cluster) Machines() ([]machinectl.Machine, error) {
	return machinectl.ListByRegexp(fmt.Sprintf("^kube-spawn-%s.*$", c.name))
}

func (c *Cluster) ListImages() ([]machinectl.Image, error) {
	return machinectl.ListImagesByRegexp(fmt.Sprintf("^kube-spawn-%s.*$", c.name))
}

func (c *Cluster) Stop() error {
	if err := c.StopMachines(); err != nil {
		return err
	}
	if err := c.RemoveImages(); err != nil {
		return err
	}
	// TODO(schu): remove network bits
	return nil
}

func (c *Cluster) Destroy() error {
	if err := c.Stop(); err != nil {
		return err
	}
	if err := os.RemoveAll(c.dir); err != nil {
		return fmt.Errorf("failed to remove cluster dir %q: %v", c.dir, err)
	}
	return nil
}

func (c *Cluster) RemoveImages() error {
	images, err := c.ListImages()
	if err != nil {
		return err
	}
	for _, image := range images {
		if err := machinectl.Remove(image.Name); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cluster) StopMachines() error {
	machines, err := c.Machines()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var failed bool
	errorChan := make(chan error)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errorChan:
				failed = true
				log.Printf("%v", err)
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(len(machines))
	for _, machine := range machines {
		go func(machineName string) {
			defer wg.Done()
			if err := machinectl.Poweroff(machineName); err != nil {
				errorChan <- fmt.Errorf("Failed to poweroff machine %q: %v", machine.Name, err)
			}
		}(machine.Name)
	}
	wg.Wait()
	close(errorChan)

	if failed {
		return fmt.Errorf("failed to poweroff all cluster machines")
	}
	return nil
}

func kubeadmInit(machineName string, outWriter io.Writer) error {
	initCmd := []string{
		"/usr/bin/kubeadm",
		"init",
		"--ignore-preflight-errors=all",
		"--config=/etc/kubeadm/kubeadm.yml",
	}
	if _, err := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, initCmd...); err != nil {
		return errors.Wrap(err, "kubeadm init failed")
	}
	if _, err := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, "/usr/bin/kubeadm", "token", "create", kubeadmToken, "--ttl=0"); err != nil {
		return errors.Wrap(err, "failed registering token")
	}
	return nil
}

func kubeadmJoin(masterIP, machineName string, outWriter io.Writer) error {
	joinCmd := []string{
		"/usr/bin/kubeadm",
		"join",
		"--ignore-preflight-errors=all",
		"--token", kubeadmToken,
		"--discovery-token-unsafe-skip-ca-verification",
		fmt.Sprintf("%s:6443", masterIP),
	}
	_, err := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, joinCmd...)
	return err
}

func applyNetworkPlugin(machineName string, outWriter io.Writer) error {
	_, err := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, "/usr/bin/kubectl", "apply", "-f", weaveNet)
	return err
}
