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
	"bufio"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	cnitypes "github.com/containernetworking/cni/pkg/types"
	cniversion "github.com/containernetworking/cni/pkg/version"
)

const bindro string = "--bind-ro="

var (
	gopath       string = os.Getenv("GOPATH")
	binariesDir  string = path.Join(gopath, "src", "k8s.io", "kubernetes", "_output", "bin")
	binariesDest string = path.Join("usr", "bin")

	cniDir  string = path.Join(gopath, "src", "github.com", "containernetworking", "cni", "bin")
	cniDest string = path.Join("opt", "cni", "bin")

	kubeletConfigPath string = path.Join(gopath, "src", "k8s.io", "release", "rpm", "10-kubeadm.conf")
	kubeletConfigDest string = path.Join("etc", "systemd", "system", "kubelet.service.d", "10-kubeadm.conf")

	systemdUnitDir  string = path.Join(gopath, "src", "k8s.io", "kubernetes", "build", "debs")
	systemdUnitDest string = path.Join("usr", "lib", "systemd", "system")
)

var defaultArgs = []string{
	bindro + path.Join(gopath, "/src/github.com/containernetworking/cni/bin:/opt/cni/bin"),
	bindro + parseBind("$PWD/etc/daemon.json:/etc/docker/daemon.json"),
	bindro + parseBind("$PWD/etc/kubeadm.yml:/etc/kubeadm/kubeadm.yml"),

	bindro + path.Join(gopath, "/src/k8s.io/kubernetes/build/debs/kubelet.service:/usr/lib/systemd/system/kubelet.service"),
	bindro + path.Join(gopath, "/src/k8s.io/release/rpm/10-kubeadm.conf:/etc/systemd/system/kubelet.service.d/10-kubeadm.conf"),

	bindro + parseBind("$PWD/etc/docker_20-kubeadm-extra-args.conf:/etc/systemd/system/docker.service.d/20-kubeadm-extra-args.conf"),
	bindro + parseBind("$PWD/etc/kube_20-kubeadm-extra-args.conf:/etc/systemd/system/kubelet.service.d/20-kubeadm-extra-args.conf"),

	bindro + path.Join(gopath, "/src/k8s.io/kubernetes/_output/bin/kubelet:/usr/bin/kubelet"),
	bindro + path.Join(gopath, "/src/k8s.io/kubernetes/_output/bin/kubeadm:/usr/bin/kubeadm"),
	bindro + path.Join(gopath, "/src/k8s.io/kubernetes/_output/bin/kubectl:/usr/bin/kubectl"),
}

func parseBind(bindstring string) string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return strings.Replace(bindstring, "$PWD", pwd, 1)
}

func RunNode(name string) error {
	args := []string{
		"cnispawn",
		"-d",
		// "--ephemeral",
		"--machine", name,
	}
	args = append(args, defaultArgs...)

	c := exec.Cmd{
		Path:   "cnispawn",
		Args:   args,
		Env:    os.Environ(),
		Stderr: os.Stderr,
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return err
	}

	if err := c.Start(); err != nil {
		return err
	}

	cniDataJSON, err := ioutil.ReadAll(bufio.NewReader(stdout))
	if err != nil {
		return err
	}

	if _, err := cniversion.NewResult(cniversion.Current(), cniDataJSON); err != nil {
		log.Printf("unexpected result output: %s", cniDataJSON)
		return fmt.Errorf("unable to parse result: %s", err)
	}

	if err := c.Wait(); err != nil {
		var cniError cnitypes.Error
		if err := json.Unmarshal(cniDataJson, &cniError); err == nil {
			return &cniError
		}
		return err
	}

	return nil
}
