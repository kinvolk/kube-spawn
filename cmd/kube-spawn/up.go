/*
Copyright 2018 Kinvolk GmbH

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
	"log"

	"github.com/spf13/cobra"
)

var (
	upCmd = &cobra.Command{
		Use:   "up",
		Short: "Create and start a new cluster",
		Example: `
# Create and start a Kubernetes v1.10.0 cluster with 4 nodes (master + 3 worker)
sudo ./kube-spawn up --kubernetes-version v1.10.0 --nodes 4`,
		Run: runUp,
	}
)

func init() {
	kubespawnCmd.AddCommand(upCmd)

	// Flags should be kept in sync with `start` and `create`

	upCmd.Flags().String("container-runtime", "docker", "Runtime to use for the cluster (can be docker or rkt)")
	upCmd.Flags().String("kubernetes-version", "v1.9.6", "Kubernetes version to install")
	upCmd.Flags().String("kubernetes-source-dir", "", "Path to directory with Kubernetes sources")
	upCmd.Flags().String("hyperkube-image", "", "Kubernetes hyperkube image to use (if unset, upstream k8s is installed)")
	upCmd.Flags().String("cni-plugin-dir", "/opt/cni/bin", "Path to directory with CNI plugins")
	upCmd.Flags().String("rkt-binary-path", "/usr/local/bin/rkt", "Path to rkt binary")
	upCmd.Flags().String("rkt-stage1-image-path", "/usr/local/bin/stage1-coreos.aci", "Path to rkt stage1-coreos.aci image")
	upCmd.Flags().String("rktlet-binary-path", "/usr/local/bin/rktlet", "Path to rktlet binary")
	upCmd.Flags().IntP("nodes", "n", 3, "Number of nodes to start")
}

func runUp(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		log.Fatalf("Command up doesn't take arguments, got: %v", args)
	}

	doCreate()
	doStart()
}
