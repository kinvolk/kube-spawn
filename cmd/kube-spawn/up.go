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
	"os"

	"github.com/spf13/cobra"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
)

var (
	cmdUp = &cobra.Command{
		Use:   "up",
		Short: "Up performs together: pulling raw image, setup and init",
		Run:   runUp,
	}

	upNumNodes     int
	upBaseImage    string
	upKubeSpawnDir string
)

func init() {
	cmdKubeSpawn.AddCommand(cmdUp)

	cmdUp.Flags().IntVarP(&upNumNodes, "nodes", "n", 1, "number of nodes to spawn")
	cmdUp.Flags().StringVarP(&upBaseImage, "image", "i", "coreos", "base image for nodes")
	cmdUp.Flags().StringVarP(&upKubeSpawnDir, "kube-spawn-dir", "d", "", "path to directory where .kube-spawn directory is located")
}

func runUp(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(1)
	}

	bootstrap.PrepareCoreosImage()

	// e.g: sudo ./kube-spawn setup --nodes=2 --image=coreos
	doSetup(upNumNodes, upBaseImage, upKubeSpawnDir)

	// sudo ./kube-spawn init
	doInit()

	log.Printf("All nodes are started.")
}
