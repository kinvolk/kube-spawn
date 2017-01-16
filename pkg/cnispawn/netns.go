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

package cnispawn

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/containernetworking/cni/pkg/ns"
)

var (
	gopath string = os.Getenv("GOPATH")
)

type CniNetns struct {
	netns ns.NetNS
}

func NewCniNetns() (*CniNetns, error) {
	netns, err := ns.NewNS()
	if err != nil {
		return nil, err
	}

	sNetnsPath := strings.Split(netns.Path(), "/")
	containerId := sNetnsPath[len(sNetnsPath)-1]

	cniPath := path.Join(gopath, "src", "github.com", "containernetworking", "cni", "bin")
	cniPluginPath := path.Join(cniPath, "bridge")

	env := os.Environ()
	env = append(env, "CNI_COMMAND=ADD")
	env = append(env, fmt.Sprintf("CNI_CONTAINERID=%s", containerId))
	env = append(env, fmt.Sprintf("CNI_NETNS=%s", netns.Path()))
	env = append(env, "CNI_IFNAME=eth0")
	env = append(env, fmt.Sprintf("CNI_PATH=%s", cniPath))

	c := exec.Cmd{
		Path:   cniPluginPath,
		Args:   nil,
		Env:    env,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	stdin, err := c.StdinPipe()
	if err != nil {
		return nil, err
	}

	netconfig, err := ioutil.ReadFile("/etc/cni/net.d/10-mynet.conf")
	if err != nil {
		return nil, err
	}

	if _, err := stdin.Write(netconfig); err != nil {
		return nil, err
	}
	stdin.Close()

	if err := c.Run(); err != nil {
		return nil, err
	}

	return &CniNetns{
		netns: netns,
	}, nil
}

func (c *CniNetns) Set() error {
	return c.netns.Set()
}

func (c *CniNetns) Close() error {
	return c.netns.Close()
}
