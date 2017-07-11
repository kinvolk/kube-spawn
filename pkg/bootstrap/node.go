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

package bootstrap

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

const containerNameTemplate string = "kube-spawn-%d"

type Node struct {
	Name string
	IP   string
}

func GetNodeName(no int) string {
	return fmt.Sprintf(containerNameTemplate, no)
}

func NewNode(baseImage, machine string) error {
	var buf bytes.Buffer

	machinectlPath, err := exec.LookPath("machinectl")
	if err != nil {
		return err
	}

	clone := exec.Cmd{
		Path:   machinectlPath,
		Args:   []string{"machinectl", "clone", baseImage, machine},
		Stderr: &buf,
	}
	if err := clone.Run(); err != nil {
		return fmt.Errorf("error running machinectl: %s", buf.String())
	}
	return nil
}

func NodeExists(machine string) bool {
	// TODO: we could also parse machinectl list-images to find that
	if _, err := os.Stat(path.Join("/var/lib/machines/", machine+".raw")); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Printf("error checking for image: %s", err)
		return false
	}
	return true
}

func GetRunningNodes() ([]Node, error) {
	var nodes []Node

	cmd := exec.Command("machinectl", "list")
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	b, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	raw := strings.Fields(string(b))
	if len(raw) <= 6 {
		return nil, fmt.Errorf("No running machines")
	}
	totaln, err := strconv.Atoi(raw[(len(raw) - 3)])
	if err != nil {
		return nil, err
	}
	for i := 1; i <= totaln; i++ {
		node := Node{
			Name: raw[i*6],
			IP:   strings.TrimSuffix(raw[(i*6)+5], "..."),
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}
