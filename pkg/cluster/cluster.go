package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/cache"
	"github.com/kinvolk/kube-spawn/pkg/machinectl"
	"github.com/kinvolk/kube-spawn/pkg/multiprint"
	"github.com/kinvolk/kube-spawn/pkg/nspawntool"
	"github.com/kinvolk/kube-spawn/pkg/utils/fs"
)

type ClusterSettings struct {
	CNIPluginDir          string
	CNIPlugin             string
	ContainerRuntime      string
	ClusterCIDR           string
	PodNetworkCIDR        string
	HyperkubeImage        string
	KubeadmApiVersion     string
	KubeadmResetOptions   string
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

	// This is needed to avoid ellipsis at the end of machine name, when running
	// `machinectl list` or `machinectl list-images`. Without setting COLUMNS to a
	// high value, machinectl prints out a shortened machine name with an ellipsis
	// at the end, so kube-spawn fails to start or stop the cluster.
	_ = os.Setenv("COLUMNS", "200")
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
	flannelNet         = "https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml"
	calicoRBAC         = "https://docs.projectcalico.org/v3.1/getting-started/kubernetes/installation/hosted/rbac-kdd.yaml"

	canalRBAC = "https://docs.projectcalico.org/v2.6/getting-started/kubernetes/installation/hosted/canal/rbac.yaml"
	canalNet  = "https://docs.projectcalico.org/v2.6/getting-started/kubernetes/installation/hosted/canal/canal.yaml"

	// Avoid token passing between master and worker nodes by using
	// a hard-coded token
	kubeadmToken = "aaaaaa.bbbbbbbbbbbbbbbb"
)

var cniFiles = map[string][]string{
	"base":    {"bridge", "dhcp", "host-device", "host-local", "ipvlan", "loopback", "macvlan", "portmap", "ptp", "sample", "tuning", "vlan"},
	"flannel": {"flannel"},
	"calico":  {"calico", "calico-ipam"},
	"canal":   {"flannel", "calico", "calico-ipam"},
	"weave":   {},
}

var validNameRegexp = regexp.MustCompile(validNameRegexpStr)

func ValidName(name string) bool {
	return validNameRegexp.MatchString(name)
}

func New(dir, name string) (*Cluster, error) {
	if !ValidName(name) {
		return nil, errors.Errorf("got invalid cluster name %q (expected %q)", name, validNameRegexpStr)
	}
	return &Cluster{
		dir:  dir,
		name: name,
	}, nil
}

func validateClusterSettings(clusterSettings *ClusterSettings) error {
	if clusterSettings.KubernetesVersion == "" && (clusterSettings.HyperkubeImage == "" || clusterSettings.KubernetesSourceDir == "") {
		return errors.Errorf("either kubernetes version or hyperkube image and kubernetes source dir must be given")
	}
	if clusterSettings.ContainerRuntime != "docker" && clusterSettings.ContainerRuntime != "rkt" {
		return errors.Errorf("unsupported container runtime given: %s", clusterSettings.ContainerRuntime)
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
		return errors.Errorf("no cache given but required")
	}

	cacheDirKubernetes := path.Join(clusterCache.Dir(), "kubernetes")

	if clusterSettings.KubernetesSourceDir == "" {
		if err := bootstrap.DownloadKubernetesBinaries(clusterSettings.KubernetesVersion, cacheDirKubernetes); err != nil {
			return errors.Wrap(err, "failed to download required Kubernetes binaries")
		}
	}

	if err := bootstrap.DownloadSocatBin(clusterCache.Dir()); err != nil {
		return errors.Wrap(err, "failed to download `socat` into cache dir")
	}

	if err := os.MkdirAll(c.BaseRootfsPath(), 0755); err != nil {
		return errors.Wrapf(err, "failed to create directory %q", c.BaseRootfsPath())
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
			return errors.Wrapf(err, "Failed to stat %q: %v", kubeadmPath)
		} else if !exists {
			kubernetesSourceBinaryDir = path.Join(clusterSettings.KubernetesSourceDir, "_output/bin")

			kubeadmPath = path.Join(kubernetesSourceBinaryDir, "kubeadm")
			if exists, err := fs.PathExists(kubeadmPath); err != nil {
				return errors.Wrapf(err, "Failed to stat %q: %v", kubernetesSourceBinaryDir)
			} else if !exists {
				return errors.Errorf("Cannot find expected `_output` directory in %q", clusterSettings.KubernetesSourceDir)
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
	for _, file := range cniFiles["base"] {
		var dst string = path.Join("opt/cni/bin", file)
		var src string = path.Join(clusterSettings.CNIPluginDir, file)
		copyItems = append(copyItems, copyItem{dst: dst, src: src})
	}
	for _, file := range cniFiles[clusterSettings.CNIPlugin] {
		var dst string = path.Join("opt/cni/bin", file)
		var src string = path.Join(clusterSettings.CNIPluginDir, file)
		copyItems = append(copyItems, copyItem{dst: dst, src: src})
	}

	if clusterSettings.ContainerRuntime == "rkt" {
		copyItems = append(copyItems, copyItem{dst: "/usr/bin/rkt", src: clusterSettings.RktBinaryPath})
		copyItems = append(copyItems, copyItem{dst: "/usr/bin/stage1-coreos.aci", src: clusterSettings.RktStage1ImagePath})
		copyItems = append(copyItems, copyItem{dst: "/usr/bin/rktlet", src: clusterSettings.RktletBinaryPath})
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
			case err, ok := <-errorChan:
				if !ok {
					return
				}
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
		return errors.Errorf("copying necessary files didn't succeed")
	}
	return prepareBaseRootfs(c.BaseRootfsPath(), clusterSettings)
}

func prepareBaseRootfs(rootfsDir string, clusterSettings *ClusterSettings) error {
	log.Print("Generating configuration files from templates ...")

	clusterSettings.UseLegacyCgroupDriver = clusterSettings.ContainerRuntime == "docker"
	if clusterSettings.ContainerRuntime == "rkt" {
		clusterSettings.RuntimeEndpoint = "unix:///var/run/rktlet.sock"
	}

	if err := fs.CreateFileFromString(path.Join(rootfsDir, "/usr/bin/kube-spawn-runc"), KubeSpawnRuncWrapperScript); err != nil {
		return err
	}
	if err := fs.CreateFileFromString(path.Join(rootfsDir, "/etc/docker/daemon.json"), DockerDaemonConfig); err != nil {
		return err
	}
	if err := fs.CreateFileFromString(path.Join(rootfsDir, "/etc/systemd/system/docker.service.d/20-kube-spawn.conf"), DockerSystemdDropin); err != nil {
		return err
	}
	if err := fs.CreateFileFromString(path.Join(rootfsDir, "/etc/resolv.conf"), "nameserver 8.8.8.8"); err != nil {
		return err
	}

	if clusterSettings.ContainerRuntime == "rkt" {
		buf, err := ExecuteTemplate(RktletSystemdUnitTmpl, clusterSettings)
		if err != nil {
			return err
		}
		if err := fs.CreateFileFromReader(path.Join(rootfsDir, "/etc/systemd/system/rktlet.service"),
			&buf); err != nil {
			return err
		}
	}
	if err := fs.CreateFileFromString(path.Join(rootfsDir, "/etc/cni/calico.yaml"), CalicoNet); err != nil {
		return err
	}
	if err := fs.CreateFileFromString(path.Join(rootfsDir, "/etc/systemd/network/50-weave.network"), WeaveSystemdNetworkdConfig); err != nil {
		return err
	}

	// When kubeadm is 1.11.0 or newer, `kubeadm reset` stops at user prompt
	// "Y or n", which prevents subsequent steps from working at all. Thus we
	// need to deal with kubeadm options differently with k8s versions.
	kubeadmVersion, err := kubeadmGitVersion(path.Join(rootfsDir, "usr/bin/kubeadm"))
	if err != nil {
		return err
	}

	apiVersion, err := getKubeadmApiVersion(kubeadmVersion)
	if err != nil {
		return err
	}
	clusterSettings.KubeadmApiVersion = apiVersion

	opts, err := getKubeadmResetOptions(kubeadmVersion)
	if err != nil {
		return err
	}
	clusterSettings.KubeadmResetOptions = opts

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
	if err := fs.CreateFileFromReader(path.Join(rootfsDir, "/etc/systemd/system/kubelet.service.d/20-kube-spawn.conf"), &buf); err != nil {
		return err
	}

	buf, err = ExecuteTemplate(KubeadmConfigTmpl, clusterSettings)
	if err != nil {
		return err
	}
	if err := fs.CreateFileFromReader(path.Join(rootfsDir, "/etc/kubeadm/kubeadm.yml"), &buf); err != nil {
		return err
	}

	return nil
}

func (c *Cluster) Start(numberNodes int, cniPluginDir string, cniPlugin string) error {
	if numberNodes < 1 {
		return errors.Errorf("cannot start less than 1 node")
	}
	if err := bootstrap.PrepareBaseImage(); err != nil {
		return err
	}

	if err := bootstrap.EnsureRequirements(); err != nil {
		return err
	}

	poolSize, err := bootstrap.GetPoolSize(bootstrap.BaseImageName, numberNodes)
	if err != nil {
		return err
	}
	log.Printf("new poolSize to be : %d\n", poolSize)
	if err := bootstrap.EnlargeStoragePool(poolSize); err != nil {
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
			case err, ok := <-errorChan:
				if !ok {
					return
				}
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

			if err := nspawntool.Run(bootstrap.BaseImageName, c.BaseRootfsPath(), path.Join(c.MachineRootfsPath(), machineName), machineName, cniPluginDir); err != nil {
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
		return errors.Errorf("starting the cluster didn't succeed")
	}

	log.Printf("Cluster %q started", c.name)

	masterMachines, err := c.MasterMachines()
	if err != nil {
		return errors.Errorf("failed to get list of master machines: %v", err)
	}
	if len(masterMachines) == 0 {
		return errors.Errorf("no master machines found")
	}

	masterMachine := masterMachines[0]

	log.Println("Note: `kubeadm init` can take several minutes")

	multiPrinter := multiprint.New(ctx)
	multiPrinter.RunPrintLoop()

	// Determine the kubeadm version by parsing `kubeadm version`
	// We need to know in order to adjust used configuration flags
	kubeadmVersion, err := kubeadmGitVersion(path.Join(c.BaseRootfsPath(), "usr/bin/kubeadm"))
	if err != nil {
		return errors.Wrap(err, "failed to determine kubeadm version")
	}

	shortName := strings.TrimPrefix(masterMachine.Name, fmt.Sprintf("kube-spawn-%s-", c.name))
	cliWriter := multiPrinter.NewWriter(fmt.Sprintf("%s ", shortName))
	if err := kubeadmInit(kubeadmVersion, masterMachine.Name, cliWriter); err != nil {
		return errors.Wrapf(err, "failed to kubeadm init %q", masterMachine.Name)
	}
	if err := applyNetworkPlugin(masterMachine.Name, cniPlugin, cliWriter); err != nil {
		return errors.Wrapf(err, "Failed to apply network plugin %q", cniPlugin)
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
			shortName := strings.TrimPrefix(nodeName, fmt.Sprintf("kube-spawn-%s-", c.name))
			if err := kubeadmJoin(kubeadmVersion, masterIP, nodeName, multiPrinter.NewWriter(fmt.Sprintf("%s ", shortName))); err != nil {
				errorChan <- errors.Wrapf(err, "Failed to kubeadm join %q", worker.Name)
			}
		}(worker.Name)
	}
	wg.Wait()

	if failed {
		return errors.Errorf("provisioning the worker nodes with kubeadm didn't succeed")
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
	if err := c.StopMachines(30 * time.Second); err != nil {
		return err
	}
	if err := c.RemoveImages(30 * time.Second); err != nil {
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
		return errors.Errorf("failed to remove cluster dir %q: %v", c.dir, err)
	}
	return nil
}

func (c *Cluster) RemoveImages(timeout time.Duration) error {
	images, err := c.ListImages()
	if err != nil {
		return err
	}
	if len(images) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tickChan := time.Tick(1 * time.Second)

	var wg sync.WaitGroup
	wg.Add(len(images))
	for i, image := range images {
		go func(imageName string, idx int) {
			defer wg.Done()
			for range tickChan {
				if err := machinectl.Remove(imageName); err == nil {
					return
				}
				select {
				case <-ctx.Done():
					// timeout
					return
				default:
				}
			}
		}(image.Name, i)
	}
	wg.Wait()

	images, err = c.ListImages()
	if err != nil {
		return err
	}
	if len(images) > 0 {
		return errors.Errorf("failed to remove all images (use `machinectl remove ...` to remove them manually)")
	}
	return nil
}

func (c *Cluster) StopMachines(timeout time.Duration) error {
	machines, err := c.Machines()
	if err != nil {
		return err
	}
	if len(machines) == 0 {
		return nil
	}

	for _, machine := range machines {
		if err := machinectl.Poweroff(machine.Name); err != nil {
			return errors.Wrapf(err, "failed to poweroff machine %q", machine.Name)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tickChan := time.Tick(2 * time.Second)

waitPoweroff:
	for range tickChan {
		select {
		case <-ctx.Done():
			// timeout
			break waitPoweroff
		default:
		}
		for _, machine := range machines {
			if machinectl.IsRunning(machine.Name) {
				continue waitPoweroff
			}
		}
		// all machines stopped already
		return nil
	}

	// poweroff didn't succeed in time, terminate the machines
	for _, machine := range machines {
		if machinectl.IsRunning(machine.Name) {
			if err := machinectl.Terminate(machine.Name); err != nil {
				return errors.Wrapf(err, "failed to terminate machine %q", machine.Name)
			}
		}
	}
	return nil
}

func kubeadmInit(kubeadmVersionStr, machineName string, outWriter io.Writer) error {
	initCmd := []string{
		"/usr/bin/kubeadm",
		"init",
		"--config=/etc/kubeadm/kubeadm.yml",
	}
	kubeadmVersion, err := semver.NewVersion(kubeadmVersionStr)
	if err != nil {
		return err
	}
	isLargerEqual19, err := semver.NewConstraint(">= 1.9")
	if err != nil {
		return err
	}
	if !isLargerEqual19.Check(kubeadmVersion) {
		initCmd = append(initCmd, "--skip-preflight-checks")
	} else {
		initCmd = append(initCmd, "--ignore-preflight-errors=all")
	}
	if _, err := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, initCmd...); err != nil {
		return errors.Wrap(err, "kubeadm init failed")
	}
	if _, err := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, "/usr/bin/kubeadm", "token", "create", kubeadmToken, "--ttl=0"); err != nil {
		return errors.Wrap(err, "failed registering token")
	}
	return nil
}

func kubeadmJoin(kubeadmVersionStr, masterIP, machineName string, outWriter io.Writer) error {
	joinCmd := []string{
		"/usr/bin/kubeadm",
		"join",
		"--token", kubeadmToken,
	}
	kubeadmVersion, err := semver.NewVersion(kubeadmVersionStr)
	if err != nil {
		return err
	}
	isLargerEqual19, err := semver.NewConstraint(">= 1.9")
	if err != nil {
		return err
	}
	if !isLargerEqual19.Check(kubeadmVersion) {
		joinCmd = append(joinCmd, "--skip-preflight-checks")
	} else {
		joinCmd = append(joinCmd,
			"--ignore-preflight-errors=all",
			"--discovery-token-unsafe-skip-ca-verification")
	}
	joinCmd = append(joinCmd, fmt.Sprintf("%s:6443", masterIP))
	_, err = machinectl.RunCommand(outWriter, nil, "", "shell", machineName, joinCmd...)
	return err
}

func getKubeadmResetOptions(kubeadmVersionStr string) (string, error) {
	kubeadmResetOptions := ""
	kubeadmVersion, err := semver.NewVersion(kubeadmVersionStr)
	if err != nil {
		return "", err
	}
	isLargerEqual111, err := semver.NewConstraint(">= 1.11")
	if err != nil {
		return "", err
	}
	if isLargerEqual111.Check(kubeadmVersion) {
		kubeadmResetOptions = "--force"
	}
	return kubeadmResetOptions, nil
}

func getKubeadmApiVersion(kubeadmVersionStr string) (string, error) {
	kubeadmApiVersion := ""
	kubeadmVersion, err := semver.NewVersion(kubeadmVersionStr)
	if err != nil {
		return "", err
	}
	isLargerEqual111, err := semver.NewConstraint(">= 1.11")
	if err != nil {
		return "", err
	}
	if isLargerEqual111.Check(kubeadmVersion) {
		kubeadmApiVersion = "v1alpha2"
	} else {
		kubeadmApiVersion = "v1alpha1"
	}
	return kubeadmApiVersion, nil
}

func applyNetworkPlugin(machineName string, cniPlugin string, outWriter io.Writer) error {
	switch cniPlugin {
	case "weave":
		_, err := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, "/usr/bin/kubectl", "apply", "-f", weaveNet)
		return err
	case "flannel":
		_, err := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, "/usr/bin/kubectl", "apply", "-f", flannelNet)
		return err
	case "canal":
		_, err1 := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, "/usr/bin/kubectl", "apply", "-f", canalRBAC)
		_, err2 := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, "/usr/bin/kubectl", "apply", "-f", canalNet)
		if err1 != nil {
			return err1
		}
		return err2
	case "calico":
		if _, err := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, "/usr/bin/kubectl", "apply", "-f", calicoRBAC); err != nil {
			return err
		}
		_, err := machinectl.RunCommand(outWriter, nil, "", "shell", machineName, "/usr/bin/kubectl", "apply", "-f", "/etc/cni/calico.yaml")
		return err
	default:
		return errors.Errorf("Incorrect cni plugin %q", cniPlugin)
	}
	return nil
}

type kubeadmVersionType struct {
	ClientVersion struct {
		GitVersion string `json:"gitVersion"`
	} `json:"clientVersion"`
}

func kubeadmGitVersion(kubeadmPath string) (string, error) {
	out, err := exec.Command(kubeadmPath, "version", "-o", "json").Output()
	if err != nil {
		return "", err
	}
	var kubeadmVersion kubeadmVersionType
	if err := json.Unmarshal(out, &kubeadmVersion); err != nil {
		return "", err
	}
	return kubeadmVersion.ClientVersion.GitVersion, nil
}
