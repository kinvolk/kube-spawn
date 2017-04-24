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

	"github.com/kinvolk/kubeadm-nspawn/pkg/bootstrap"
	"github.com/kinvolk/kubeadm-nspawn/pkg/distribution"
	"github.com/kinvolk/kubeadm-nspawn/pkg/nspawntool"
	"github.com/kinvolk/kubeadm-nspawn/pkg/ssh"
	"github.com/kinvolk/kubeadm-nspawn/pkg/utils"
)

const (
	pushImageRetries int = 10
)

var (
	gopath      string = os.Getenv("GOPATH")
	nodes       int
	imageMethod string
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

	if err := ssh.PrepareSSHKeys(); err != nil {
		log.Fatal("Error when generating SSH keys: ", err)
	}

	if err := nspawntool.CreateImage(imageMethod); err != nil {
		log.Fatal("Error when creating image: ", err)
	}

	for i := 0; i < nodes; i++ {
		name := nspawntool.GetNodeName(i)

		rootfsExists, err := bootstrap.NodeRootfsExists(name)
		if err != nil {
			log.Fatal("Error when checking node rootfs directory: ", err)
		}

		if !rootfsExists {
			if err := nspawntool.ExtractImage(name); err != nil {
				log.Fatal("Error when extracting image: ", err)
			}
			if err := bootstrap.BootstrapNode(name); err != nil {
				log.Fatal("Error when bootstrapping node :", err)
			}
		}

		if err := nspawntool.RunNode(name); err != nil {
			if err := nspawntool.Cleanup(nodes); err != nil {
				log.Fatal("Error when cleaning up: ", err)
			}
			log.Fatal("Error when running node: ", err)
		}
	}
}

func newUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Start nodes",
		Run:   runUp,
	}
	cmd.Flags().IntVarP(&nodes, "nodes", "n", 1, "number of nodes to spawn")
	cmd.Flags().StringVarP(&imageMethod, "image-method", "m", "mkosi", "method to use for setting up node rootfs [mkosi, download]")
	return cmd
}

func runInit(cmd *cobra.Command, args []string) {
	log.Println("Warning: experimental!")

	nodes, err := nspawntool.RunningNodes()
	if err != nil {
		log.Fatal("Error listing running nodes: ", err)
	}

	if err := ssh.InitializeMaster(nodes[0].IP.String()); err != nil {
		if err := nspawntool.Cleanup(len(nodes)); err != nil {
			log.Fatal("Error when cleaning up: ", err)
		}
		log.Fatal("Error when initializing master: ", err)
	}

	token, err := utils.GetToken("kubeadm-nspawn-0")
	if err != nil {
		if err := nspawntool.Cleanup(len(nodes)); err != nil {
			log.Fatal("Error when cleaning up: ", err)
		}
		log.Fatal("Error when getting token: ", err)
	}

	for i, node := range nodes {
		if i != 0 {
			if err := ssh.JoinNode(node.IP.String(), token, nodes[0].IP.String()); err != nil {
				if err := nspawntool.Cleanup(len(nodes)); err != nil {
					log.Fatal("Error when cleaning up: ", err)
				}
				log.Fatal("Error when joining node: ", err)
			}
		}
	}
}

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Execute kubeadm",
		Run:   runInit,
	}
	return cmd
}

func runList(cmd *cobra.Command, args []string) {
	nodes, err := nspawntool.RunningNodes()
	if err != nil {
		log.Fatal("Error listing running nodes: ", err)
	}

	if len(nodes) > 0 {
		fmt.Println("NODE\t\t\tPID\tIP")
		for _, n := range nodes {
			fmt.Printf("%v\t%v\t%v\n", n.Name, n.PID, n.IP.String())
		}
		fmt.Printf("\n%v nodes listed.\n", len(nodes))
	} else {
		fmt.Println("No nodes.")
	}
}

func newListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List running nodes",
		Run:     runList,
	}
	return cmd
}

func runDown(cmf *cobra.Command, args []string) {
	log.Println("! For now this runs the cleanup code !")

	nodes, err := nspawntool.RunningNodes()
	if err != nil {
		log.Fatal("Error listing running nodes: ", err)
	}

	if err := nspawntool.Cleanup(len(nodes)); err != nil {
		log.Fatal("Error when bringing down nodes: ", err)
	}
}

func newDownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Stop running nodes",
		Run:   runDown,
	}
	return cmd
}

func newKubeadmSystemdCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubeadm-nspawn",
		Short: "kubeadm-nspawn is a tool for creating a multi-node dev Kubernetes cluster",
		Long:  "kubeadm-nspawn is a tool for creating a multi-node dev Kubernetes cluster, by using the local source code and systemd-nspawn containers",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Usage(); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.AddCommand(newUpCommand())
	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newDownCommand())
	return cmd
}

func main() {
	if err := newKubeadmSystemdCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
