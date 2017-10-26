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
	"github.com/spf13/cobra"
)

var (
	restartCmd = &cobra.Command{
		Use:   "restart",
		Short: "Stop and start the cluster",
		Long: `Stop and start the cluster.
Shortcut for running
	kube-spawn stop
	kube-spawn start

You should have run 'kube-spawn create' before this.`,
		Run: runRestart,
	}
)

func init() {
	kubespawnCmd.AddCommand(restartCmd)
	restartCmd.Flags().BoolVar(&flagSkipInit, "skip-cluster-init", false, "skips the initialization of a Kubernetes-Cluster with kubeadm")
	restartCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "terminate machines instead of trying graceful shutdown")
}

func runRestart(cmd *cobra.Command, args []string) {
	cfg := loadConfig()
	doStop(cfg, flagForce)
	doStart(cfg, flagSkipInit)
}
