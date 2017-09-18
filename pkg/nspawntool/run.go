/*
Copyright 2017 Kinvolk GmbH

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nspawntool

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	cnitypes "github.com/containernetworking/cni/pkg/types"
	cniversion "github.com/containernetworking/cni/pkg/version"
	"github.com/pkg/errors"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/utils"
)

const bindro string = "--bind-ro="
const bindrw string = "--bind="

var (
	goPath  string
	cniPath string

	k8sBinds     []string
	rktBinds     []string
	defaultBinds []string
)

type bindOption struct {
	bindPrefix string // either bindro or bindrw
	srcMount   string // path to source file for bind mount
	dstMount   string // path to destination file for bind mount
}

// composeBindOption() combinds 3 entries of bindOption into a single string,
// which can be passed directly to systemd-nspawn. For example:
//   --bind=/src/file:/dst/file
func (bo *bindOption) composeBindOption() (string, error) {
	if _, err := os.Stat(bo.srcMount); os.IsNotExist(err) {
		return "", errors.Wrapf(err, "invalid file %s", bo.srcMount)
	}
	return fmt.Sprintf("%s%s:%s", bo.bindPrefix, bo.srcMount, bo.dstMount), nil
}

func parseBind(bindstring string) string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return strings.Replace(bindstring, "$PWD", pwd, 1)
}

func getDefaultBindOpts(kubeSpawnDir, cniPath string) []bindOption {
	return []bindOption{
		// kube-spawn bins
		{
			bindrw,
			filepath.Join(kubeSpawnDir, "scripts"),
			"/opt/kube-spawn",
		},
		{
			bindro,
			parseBind("$PWD/kube-spawn-runc"),
			"/opt/kube-spawn-runc",
		},
		// shared kubeSpawnDir
		{
			bindrw,
			path.Join(kubeSpawnDir, "default"),
			kubeSpawnDir,
		},
		// extra configs
		{
			bindro,
			filepath.Join(kubeSpawnDir, "etc/daemon.json"),
			"/etc/docker/daemon.json",
		},
		{
			bindro,
			filepath.Join(kubeSpawnDir, "etc/kubeadm.yml"),
			"/etc/kubeadm/kubeadm.yml",
		},
		{
			bindro,
			filepath.Join(kubeSpawnDir, "etc/kube_20-kubeadm-extra-args.conf"),
			"/etc/systemd/system/kubelet.service.d/20-kubeadm-extra-args.conf",
		},
		{
			bindro,
			filepath.Join(kubeSpawnDir, "etc/kube_tmpfiles_kubelet.conf"),
			"/usr/lib/tmpfiles.d/kubelet.conf",
		},
		{
			bindro,
			filepath.Join(kubeSpawnDir, "etc/weave_50-weave.network"),
			"/etc/systemd/network/50-weave.network",
		},
		// cni bins
		{
			bindrw,
			parseBind(cniPath),
			"/opt/cni/bin",
		},
	}
}

func getK8sBindOpts(k8srelease, kubeSpawnDir, goPath string) ([]bindOption, error) {
	if utils.IsK8sDev(k8srelease) {
		// self-compiled k8s development tree
		k8sOutputDir, err := utils.GetK8sBuildOutputDir(filepath.Join(goPath, "/src/k8s.io/kubernetes"))
		if err != nil {
			return nil, errors.Wrap(err, "error getting k8s output directory")
		}

		return []bindOption{
			// bins
			{
				bindro,
				path.Join(k8sOutputDir, "kubelet"),
				"/usr/bin/kubelet",
			},
			{
				bindro,
				path.Join(k8sOutputDir, "kubeadm"),
				"/usr/bin/kubeadm",
			},
			{
				bindro,
				path.Join(k8sOutputDir, "kubectl"),
				"/usr/bin/kubectl",
			},
			// service files
			{
				bindro,
				path.Join(goPath, "/src/k8s.io/kubernetes/build/debs/kubelet.service"),
				"/usr/lib/systemd/system/kubelet.service",
			},
			{
				bindro,
				path.Join(goPath, "/src/k8s.io/kubernetes/build/rpms/10-kubeadm.conf"),
				"/etc/systemd/system/kubelet.service.d/10-kubeadm.conf",
			},
		}, nil
	} else {
		// k8s releases pre-built and downloaded
		return []bindOption{
			// bins
			{
				bindro,
				path.Join(kubeSpawnDir, "/k8s/kubelet"),
				"/usr/bin/kubelet",
			},
			{
				bindro,
				path.Join(kubeSpawnDir, "/k8s/kubeadm"),
				"/usr/bin/kubeadm",
			},
			{
				bindro,
				path.Join(kubeSpawnDir, "/k8s/kubectl"),
				"/usr/bin/kubectl",
			},
			// service files
			{
				bindro,
				path.Join(kubeSpawnDir, "/k8s/kubelet.service"),
				"/usr/lib/systemd/system/kubelet.service",
			},
			{
				bindro,
				path.Join(kubeSpawnDir, "/k8s/10-kubeadm.conf"),
				"/etc/systemd/system/kubelet.service.d/10-kubeadm.conf",
			},
		}, nil
	}
}

func getRktBindOpts(kubeSpawnDir, rktBinDir, rktletBinDir string) []bindOption {
	return []bindOption{
		// rktlet
		{
			bindro,
			filepath.Join(rktBinDir, "rkt"),
			"/usr/bin/rkt",
		},
		{
			bindro,
			filepath.Join(rktBinDir, "stage1-coreos.aci"),
			"/usr/bin/stage1-coreos.aci",
		},
		{
			bindro,
			filepath.Join(rktletBinDir, "rktlet"),
			"/usr/bin/rktlet",
		},
		{
			bindro,
			filepath.Join(kubeSpawnDir, "etc/rktlet.service"),
			"/usr/lib/systemd/system/rktlet.service",
		},
	}
}

func getDockerBindOpts(kubeSpawnDir string) []bindOption {
	return []bindOption{
		{
			bindro,
			filepath.Join(kubeSpawnDir, "etc/docker_20-kubeadm-extra-args.conf"),
			"/etc/systemd/system/docker.service.d/20-kubeadm-extra-args.conf",
		},
	}
}

type Node struct {
	Name       string
	K8sVersion string
	Runtime    string
	bindOpts   []bindOption
}

func (n *Node) buildBindsListKubernetes(kubeSpawnDir, goPath string) error {
	k8sBindOpts, err := getK8sBindOpts(n.K8sVersion, kubeSpawnDir, goPath)
	if err != nil {
		return err
	}
	n.bindOpts = append(n.bindOpts, k8sBindOpts...)

	// NOTE: workaround for making kubelet work with port-forward
	bo := bindOption{
		bindPrefix: bindro,
		srcMount:   path.Join(kubeSpawnDir, "/extras/socat"),
		dstMount:   "/usr/bin/socat"}
	n.bindOpts = append(n.bindOpts, bo)
	return nil
}

func (n *Node) buildBindsList(kubeSpawnDir, rktBinDir, rktletBinDir string) ([]string, error) {
	var err error

	if goPath, err = utils.GetValidGoPath(); err != nil {
		return nil, errors.Errorf("invalid GOPATH %q", goPath)
	}
	if cniPath, err = utils.GetValidCniPath(goPath); err != nil {
		return nil, errors.Errorf("invalid CNI_PATH %q", cniPath)
	}

	n.bindOpts = append(n.bindOpts, getDefaultBindOpts(kubeSpawnDir, cniPath)...)

	if err := n.buildBindsListKubernetes(kubeSpawnDir, goPath); err != nil {
		return nil, errors.Wrap(err, "error getting kubernetes bind mounts list: %v")
	}

	var runtimeDir string
	switch n.Runtime {
	case "docker":
		runtimeDir = "/var/lib/docker"
		n.bindOpts = append(n.bindOpts, getDockerBindOpts(kubeSpawnDir)...)
	case "rkt":
		runtimeDir = "/var/lib/rktlet"
		n.bindOpts = append(n.bindOpts, getRktBindOpts(kubeSpawnDir, rktBinDir, rktletBinDir)...)
	case "crio":
		return nil, errors.Errorf("%v runtime not supported", n.Runtime)
	}

	// mount directory ./.kube-spawn/default/MACHINE_NAME/mount into
	// /var/lib/{docker, rktlet} inside the node
	mountDir := path.Join(kubeSpawnDir, "default", n.Name, "mount")
	if err := os.MkdirAll(mountDir, os.FileMode(0755)); err != nil {
		return nil, errors.Wrapf(err, "unable to create dir %q", mountDir)
	}
	bo := bindOption{
		bindPrefix: bindrw,
		srcMount:   mountDir,
		dstMount:   runtimeDir,
	}
	n.bindOpts = append(n.bindOpts, bo)

	var listopts []string
	for _, bo := range n.bindOpts {
		opt, err := bo.composeBindOption()
		if err != nil {
			return nil, fmt.Errorf("unable to compose build option: %q: %v", bo.srcMount, err)
		}
		listopts = append(listopts, opt)
	}
	return listopts, err
}

func (n *Node) Run(kubeSpawnDir, rktBinDir, rktletBinDir string) error {
	var err error

	if err := os.MkdirAll(kubeSpawnDir, os.FileMode(0755)); err != nil {
		return errors.Wrapf(err, "unable to create directory %q", kubeSpawnDir)
	}
	if err := bootstrap.PathSupportsOverlay(kubeSpawnDir); err != nil {
		return errors.Wrapf(err, "unable to create overlayfs on %q. Try to pass a directory with a different filesystem (like ext4 or XFS) to --kube-spawn-dir.", kubeSpawnDir)
	}

	args := []string{
		"cnispawn",
		"-d",
		"--machine", n.Name,
	}

	listopts, err := n.buildBindsList(kubeSpawnDir, rktBinDir, rktletBinDir)
	if err != nil {
		return errors.Wrap(err, "error processing bind options")
	}
	args = append(args, listopts...)

	c := exec.Cmd{
		Path:   "cnispawn",
		Args:   args,
		Env:    os.Environ(),
		Stderr: os.Stderr,
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "error creating stdout pipe")
	}
	defer stdout.Close()

	if err := c.Start(); err != nil {
		return errors.Wrap(err, "error running cnispawn")
	}

	cniDataJSON, err := ioutil.ReadAll(stdout)
	if err != nil {
		return errors.Wrap(err, "error reading cni data from stdin")
	}

	if _, err := cniversion.NewResult(cniversion.Current(), cniDataJSON); err != nil {
		log.Printf("unexpected result output: %s", cniDataJSON)
		return errors.Wrap(err, "unable to parse result")
	}

	if err := c.Wait(); err != nil {
		var cniError cnitypes.Error
		if err := json.Unmarshal(cniDataJSON, &cniError); err != nil {
			return errors.Wrap(err, "error unmarshaling cni error")
		}
		return errors.Wrap(err, "error running cnispawn")
	}

	log.Printf("Waiting for %s to start up", n.Name)
	ready := false
	retries := 0
	for !ready {
		check := exec.Command("systemctl", "--machine", n.Name, "status", "basic.target", "--state=running")
		check.Run()
		if ready = check.ProcessState.Success(); !ready {
			time.Sleep(2 * time.Second)
			retries++
		}
		if retries >= 10 {
			return errors.Errorf("timeout waiting for %s to start", n.Name)
		}
	}

	return nil
}
