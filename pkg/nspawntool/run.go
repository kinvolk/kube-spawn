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
	"strings"
	"time"

	cnitypes "github.com/containernetworking/cni/pkg/types"
	cniversion "github.com/containernetworking/cni/pkg/version"
)

const bindro string = "--bind-ro="
const bindrw string = "--bind="

var (
	gopath  string = os.Getenv("GOPATH")
	cnipath string = os.Getenv("CNI_PATH")
	// binariesDir  string = path.Join(gopath, "src", "k8s.io", "kubernetes", "_output", "bin")
	// binariesDest string = path.Join("usr", "bin")

	// cniDir  string = path.Join(gopath, "src", "github.com", "containernetworking", "cni", "bin")
	// cniDest string = path.Join("opt", "cni", "bin")

	// kubeletConfigPath string = path.Join(gopath, "src", "k8s.io", "release", "rpm", "10-kubeadm.conf")
	// kubeletConfigDest string = path.Join("etc", "systemd", "system", "kubelet.service.d", "10-kubeadm.conf")

	// systemdUnitDir  string = path.Join(gopath, "src", "k8s.io", "kubernetes", "build", "debs")
	// systemdUnitDest string = path.Join("usr", "lib", "systemd", "system")
)

var k8sbinds []string
var defaultBinds = []string{
	// kube-spawn bins
	bindrw + parseBind("$PWD/scripts:/opt/kube-spawn"),
	bindro + parseBind("$PWD/kube-spawn-runc:/opt/kube-spawn-runc"),
	// shared tmpdir
	bindrw + parseBind("$PWD/tmp:/tmp/kube-spawn"),
	// extra configs
	bindro + parseBind("$PWD/etc/daemon.json:/etc/docker/daemon.json"),
	bindro + parseBind("$PWD/etc/kubeadm.yml:/etc/kubeadm/kubeadm.yml"),
	bindro + parseBind("$PWD/etc/docker_20-kubeadm-extra-args.conf:/etc/systemd/system/docker.service.d/20-kubeadm-extra-args.conf"),
	bindro + parseBind("$PWD/etc/kube_20-kubeadm-extra-args.conf:/etc/systemd/system/kubelet.service.d/20-kubeadm-extra-args.conf"),
	// cni bins
	bindrw + path.Join(cnipath+":/opt/cni/bin"),
}

func parseBind(bindstring string) string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return strings.Replace(bindstring, "$PWD", pwd, 1)
}

func RunNode(k8srelease, name string) error {
	args := []string{
		"cnispawn",
		"-d",
		"--machine", name,
	}
	args = append(args, defaultBinds...)

	// TODO: we should have something like a "bind builder" that reuses code
	if k8srelease != "" {
		k8sbinds = []string{
			// bins
			bindro + parseBind("$PWD/k8s/kubelet:/usr/bin/kubelet"),
			bindro + parseBind("$PWD/k8s/kubeadm:/usr/bin/kubeadm"),
			bindro + parseBind("$PWD/k8s/kubectl:/usr/bin/kubectl"),
			// service files
			bindro + parseBind("$PWD/k8s/kubelet.service:/usr/lib/systemd/system/kubelet.service"),
			bindro + parseBind("$PWD/k8s/10-kubeadm.conf:/etc/systemd/system/kubelet.service.d/10-kubeadm.conf"),
		}
	} else {
		k8sbinds = []string{
			// bins
			bindro + path.Join(gopath, "/src/k8s.io/kubernetes/_output/bin/kubelet:/usr/bin/kubelet"),
			bindro + path.Join(gopath, "/src/k8s.io/kubernetes/_output/bin/kubeadm:/usr/bin/kubeadm"),
			bindro + path.Join(gopath, "/src/k8s.io/kubernetes/_output/bin/kubectl:/usr/bin/kubectl"),
			// service files
			bindro + path.Join(gopath, "/src/k8s.io/kubernetes/build/debs/kubelet.service:/usr/lib/systemd/system/kubelet.service"),
			bindro + path.Join(gopath, "/src/k8s.io/release/rpm/10-kubeadm.conf:/etc/systemd/system/kubelet.service.d/10-kubeadm.conf"),
		}
	}
	args = append(args, k8sbinds...)

	c := exec.Cmd{
		Path:   "cnispawn",
		Args:   args,
		Env:    os.Environ(),
		Stderr: os.Stderr,
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %s", err)
	}
	defer stdout.Close()

	if err := c.Start(); err != nil {
		return fmt.Errorf("error running cnispawn: %s", err)
	}

	cniDataJSON, err := ioutil.ReadAll(stdout)
	if err != nil {
		return fmt.Errorf("error reading cni data from stdin: %s", err)
	}

	if _, err := cniversion.NewResult(cniversion.Current(), cniDataJSON); err != nil {
		log.Printf("unexpected result output: %s", cniDataJSON)
		return fmt.Errorf("unable to parse result: %s", err)
	}

	if err := c.Wait(); err != nil {
		var cniError cnitypes.Error
		if err := json.Unmarshal(cniDataJSON, &cniError); err == nil {
			return fmt.Errorf("error unmarshaling cni error: %s", err)
		}
		return fmt.Errorf("error running cnispawn: %s", cniError)
	}

	log.Printf("Waiting for %s to start up", name)
	ready := false
	retries := 0
	for !ready {
		check := exec.Command("systemctl", "--machine", name, "status", "basic.target", "--state=running")
		check.Run()
		if ready = check.ProcessState.Success(); !ready {
			time.Sleep(2 * time.Second)
			retries++
		}
		if retries >= 10 {
			return fmt.Errorf("timeout waiting for %s to start", name)
		}
	}

	return nil
}
