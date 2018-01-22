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
	"log"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kinvolk/kube-spawn/pkg/cluster"
)

var (
	startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start a cluster. You should have run 'kube-spawn create' before this",
		Run:   runStart,
	}
	flagSkipInit bool
)

func init() {
	kubespawnCmd.AddCommand(startCmd)

	startCmd.Flags().BoolVar(&flagSkipInit, "skip-cluster-init", false, "Skips cluster initialization through kubeadm")
	startCmd.Flags().IntP("nodes", "n", 3, "Number of nodes to start")
	startCmd.Flags().String("cni-plugin-dir", "/opt/cni/bin", "Path to directory with CNI plugins")

	viper.BindPFlags(startCmd.Flags())
}

func runStart(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		log.Fatalf("Command start doesn't take arguments, got: %v", args)
	}

	kubespawnDir := viper.GetString("dir")
	clusterName := viper.GetString("cluster-name")
	numberNodes := viper.GetInt("nodes")
	cniPluginDir := viper.GetString("cni-plugin-dir")

	kluster, err := cluster.New(path.Join(kubespawnDir, "clusters", clusterName), clusterName)
	if err != nil {
		log.Fatalf("Failed to create cluster object: %v", err)
	}

	if err := kluster.Start(numberNodes, cniPluginDir); err != nil {
		log.Fatalf("Failed to start cluster: %v", err)
	}

	log.Printf("Cluster %q initialized", clusterName)
	log.Println("Export $KUBECONFIG as follows for kubectl:")
	log.Printf("\n\texport KUBECONFIG=%s\n\n", kluster.AdminKubeconfigPath())
}
