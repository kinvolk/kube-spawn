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

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/kinvolk/kubeadm-systemd/pkg/bootstrap"
	"github.com/kinvolk/kubeadm-systemd/pkg/distribution"
	"github.com/kinvolk/kubeadm-systemd/pkg/nspawntool"
	"github.com/kinvolk/kubeadm-systemd/pkg/ssh"
	"github.com/kinvolk/kubeadm-systemd/pkg/utils"
)

const (
	containerNameTemplate string = "kubeadm-systemd-%d"
	pushImageRetries      int    = 10
)

var (
	gopath string = os.Getenv("GOPATH")
	nodes  int
)

func runUp(cmd *cobra.Command, args []string) {
	if err := bootstrap.EnsureBridge(); err != nil {
		log.Fatal("Error when checking CNI bridge: ", err)
	}
	if err := distribution.StartRegistry(); err != nil {
		log.Fatal("Error when starting registry: ", err)
	}

	var err error
	for i := 0; i < pushImageRetries; i++ {
		err = distribution.PushImage()
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.Fatal("Error when pushing image: ", err)
	}

	if err := nspawntool.DownloadImage(); err != nil {
		log.Fatal("Error when downloading image: ", err)
	}
	if err := ssh.PrepareSSHKeys(); err != nil {
		log.Fatal("Error when generating SSH keys: ", err)
	}

	masterIpAddresses := make([]string, 0, 1)
	nodeIpAddresses := make([]string, 0, nodes-1)

	for i := 0; i < nodes; i++ {
		name := getContainerName(i)

		rootfsExists, err := bootstrap.ContainerRootfsExists(name)
		if err != nil {
			log.Fatal("Error when checking container rootfs directory: ", err)
		}

		if !rootfsExists {
			if err := nspawntool.ExtractImage(name); err != nil {
				log.Fatal("Error when extracting image: ", err)
			}
			if err := bootstrap.BootstrapContainer(name); err != nil {
				log.Fatal("Error when bootstraping container :", err)
			}
		}

		ip, err := nspawntool.RunContainer(name)
		if err != nil {
			log.Fatal("Error when running container: ", err)
		}
		if i < 1 {
			masterIpAddresses = append(masterIpAddresses, ip)
		} else {
			nodeIpAddresses = append(nodeIpAddresses, ip)
		}
	}

	for _, masterIpAddress := range masterIpAddresses {
		if err := ssh.InitializeMaster(masterIpAddress); err != nil {
			log.Fatal("Error when initializing master: ", err)
		}
	}

	token, err := utils.GetToken("kubeadm-systemd-0")
	if err != nil {
		log.Fatal("Error when getting token: ", err)
	}

	for _, nodeIpAddress := range nodeIpAddresses {
		if err := ssh.JoinNode(nodeIpAddress, token, masterIpAddresses[0]); err != nil {
			log.Fatal("Error when joining node: ", err)
		}
	}
}

func newUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "up",
		Run: runUp,
	}
	cmd.Flags().IntVarP(&nodes, "nodes", "n", 1, "number of nodes to spawn")
	return cmd
}

func runDown(cmf *cobra.Command, args []string) {
	log.Fatal("Not implemented yet. Please remove the containers by machinectl.")
}

func newDownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "down",
		Run: runDown,
	}
	return cmd
}

func newKubeadmSystemdCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubeadm-systemd",
		Short: "kubeadm-systemd is a tool for creating a multi-node dev Kubernetes cluster",
		Long:  "kubeadm-systemd is a tool for creating a multi-node dev Kubernetes cluster, by using the local source code and systemd-nspawn containers",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Usage(); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.AddCommand(newUpCommand())
	cmd.AddCommand(newDownCommand())
	return cmd
}

func getContainerName(no int) string {
	return fmt.Sprintf(containerNameTemplate, no)
}

func main() {
	if err := newKubeadmSystemdCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
