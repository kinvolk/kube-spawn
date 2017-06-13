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
)

const (
	pushImageRetries int = 10
)

var (
	version      string
	gopath       string = os.Getenv("GOPATH")
	nodes        int
	printVersion bool
	baseImage    string
)

func runUp(cmd *cobra.Command, args []string) {
	if err := bootstrap.EnsureBridge(); err != nil {
		log.Fatalf("Error checking CNI bridge: %s", err)
	}
	if err := distribution.StartRegistry(); err != nil {
		log.Fatalf("Error starting registry: %s", err)
	}

	if err := bootstrap.WriteNetConf(); err != nil {
		log.Fatal("Error writing CNI configuration: ", err)
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
		log.Fatalf("Error pushing hyperkube image: %s", err)
	}

	log.Printf("Checking base image")
	if baseImage == "" {
		log.Fatal("No base image specified.")
	}
	if !bootstrap.NodeExists(baseImage) {
		log.Fatal("Base image not found.")
	}

	for i := 0; i < nodes; i++ {
		name := bootstrap.GetNodeName(i)

		if !bootstrap.NodeExists(name) {
			if err := bootstrap.NewNode(baseImage, name); err != nil {
				log.Fatalf("Error cloning base image: %s", err)
			}
		}

		if err := nspawntool.RunNode(name); err != nil {
			log.Fatalf("Error running node: %s", err)
		}

		if err := nspawntool.RunBootstrapScript(name); err != nil {
			log.Fatalf("Error running bootstrap script: %s", err)
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
	cmd.Flags().StringVarP(&baseImage, "image", "i", "", "base image for nodes")
	return cmd
}

func runInit(cmd *cobra.Command, args []string) {
	nodes, err := bootstrap.GetRunningNodes()
	if err != nil {
		log.Fatalf("Error listing running nodes: %s", err)
	}
	if len(nodes) == 0 {
		log.Fatal("No node running. Is systemd-nspawn running correctly?")
	}

	token, err := ssh.InitializeMaster(nodes[0].IP)
	if err != nil {
		log.Fatalf("Error initializing master: %s", err)
	}

	for i, node := range nodes {
		if i != 0 {
			if err := ssh.JoinNode(node.IP, nodes[0].IP, token); err != nil {
				log.Fatalf("Error joining node: %s", err)
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

func newKubeadmNspawnCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubeadm-nspawn",
		Short: "kubeadm-nspawn is a tool for creating a multi-node dev Kubernetes cluster",
		Long:  "kubeadm-nspawn is a tool for creating a multi-node dev Kubernetes cluster, by using the local source code and systemd-nspawn containers",
		Run: func(cmd *cobra.Command, args []string) {
			if printVersion {
				fmt.Printf("kubeadm-nspawn %s\n", version)
				os.Exit(0)
			}
			if err := cmd.Usage(); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.Flags().BoolVarP(&printVersion, "version", "V", false, "output version information")
	cmd.AddCommand(newUpCommand())
	cmd.AddCommand(newInitCommand())
	return cmd
}

func main() {
	if err := newKubeadmNspawnCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
