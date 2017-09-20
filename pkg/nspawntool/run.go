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
	"time"

	cnitypes "github.com/containernetworking/cni/pkg/types"
	cniversion "github.com/containernetworking/cni/pkg/version"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
)

var (
	goPath  string
	cniPath string
)

type Node struct {
	Name       string
	K8sVersion string
	Runtime    string
	bindOpts   []bindOption
}

func (n *Node) Run(kubeSpawnDir, rktBin, rktStage1Image, rktletBin string) error {
	var err error

	if err := os.MkdirAll(kubeSpawnDir, os.FileMode(0755)); err != nil {
		return fmt.Errorf("unable to create directory %q: %v.", kubeSpawnDir, err)
	}
	if err := bootstrap.PathSupportsOverlay(kubeSpawnDir); err != nil {
		return fmt.Errorf("RunNode: unable to create overlayfs on %q: %v. Try to pass a directory with a different filesystem (like ext4 or XFS) to --kube-spawn-dir.", kubeSpawnDir, err)
	}

	args := []string{
		"cnispawn",
		"-d",
		"--machine", n.Name,
	}

	listopts, err := n.buildBindsList(kubeSpawnDir, rktBin, rktStage1Image, rktletBin)
	if err != nil {
		return fmt.Errorf("RunNode: error processing bind options: %v", err)
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
		if err := json.Unmarshal(cniDataJSON, &cniError); err != nil {
			return fmt.Errorf("error unmarshaling cni error: %s", err)
		}
		return fmt.Errorf("error running cnispawn: %s", cniError)
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
			return fmt.Errorf("timeout waiting for %s to start", n.Name)
		}
	}

	return nil
}
