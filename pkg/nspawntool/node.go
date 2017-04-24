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
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"k8s.io/kubernetes/pkg/util/procfs"
)

type Node struct {
	Name string
	PID  int
	IP   net.IP
}

func RunningNodes() ([]Node, error) {
	var nodes []Node

	pids, err := procfs.PidOf("systemd-nspawn")
	if err != nil {
		return nil, err
	}
	if len(pids) <= 0 {
		return nodes, nil
	}

	for i, pid := range pids {
		nodes = append(nodes, Node{
			Name: GetNodeName(i),
			PID:  pid,
			IP:   getIP(i),
		})
	}

	return nodes, nil
}

func getIP(n int) net.IP {
	cmd := exec.Command("sudo", "machinectl", "shell", "-q", GetNodeName(n), "/usr/sbin/ip", "-f", "inet", "-4", "-o", "address", "show", "eth0")
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	b, err := cmd.Output()
	if err != nil {
		return nil
	}

	raw := strings.Fields(string(b))

	ip, _, err := net.ParseCIDR(raw[3])
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return ip
}
