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
	"os/exec"

	"github.com/kinvolk/kube-spawn/pkg/utils"
	"github.com/spf13/cobra"
)

const k8sStableVersion string = "1.7.0"
const defaultRuntime string = "docker"
const kubeSpawnDirDefault string = "/var/lib/kube-spawn"

var (
	cmdKubeSpawn = &cobra.Command{
		Use:   "kube-spawn",
		Short: "kube-spawn is a tool for creating a multi-node dev Kubernetes cluster",
		Long:  "kube-spawn is a tool for creating a multi-node dev Kubernetes cluster, by using the local source code and systemd-nspawn containers",
		Run: func(cmd *cobra.Command, args []string) {
			if printVersion {
				fmt.Printf("kube-spawn %s\n", version)
				os.Exit(0)
			}
			if err := cmd.Usage(); err != nil {
				log.Fatal(err)
			}
		},
	}

	version        string
	k8srelease     string
	k8sruntime     string
	rktBin         string = os.Getenv("KUBE_SPAWN_RKT_BIN")
	rktStage1Image string = os.Getenv("KUBE_SPAWN_RKT_STAGE1_IMAGE")
	rktletBin      string = os.Getenv("KUBE_SPAWN_RKTLET_BIN")
	printVersion   bool
	kubeSpawnDir   string

	kubeadmCgroupDriver     string
	kubeadmRuntimeEndpoint  string
	kubeadmRequestTimeout   string = "15m"
	kubeadmContainerRuntime string
)

func init() {
	cmdKubeSpawn.Flags().BoolVarP(&printVersion, "version", "V", false, "Output version information")
	cmdKubeSpawn.PersistentFlags().StringVarP(&k8sruntime, "container-runtime", "r", defaultRuntime, "Runtime to use for the spawned cluster (docker or rkt)")
	cmdKubeSpawn.PersistentFlags().StringVarP(&k8srelease, "kubernetes-version", "k", k8sStableVersion, "Kubernetes version to spawn, \"\" or \"dev\" for self-building upstream K8s.")
	cmdKubeSpawn.PersistentFlags().StringVarP(&kubeSpawnDir, "kube-spawn-dir", "d", kubeSpawnDirDefault, "path to kube-spawn asset directory")

	cmdKubeSpawn.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// TODO: we should eventually run extensive env/config checks prior to running any subcommand
		// That would also benefit from some kind of centralized config
		var err error
		switch k8sruntime {
		case "rkt":
			if rktBin == "" {
				rktBin, err = exec.LookPath("rkt")
				if err != nil {
					log.Fatal("Unable to find rkt binary. Put it in your PATH or use KUBE_SPAWN_RKT_BIN.")
				}
			}
			if rktStage1Image == "" {
				rktStage1Image = "/usr/lib/rkt/stage1-images/stage1-coreos.aci"
				if err := utils.CheckValidFile(rktStage1Image); err != nil {
					log.Fatal("Unable to find stage1-coreos.aci, use KUBE_SPAWN_RKT_STAGE1_IMAGE.")
				}
			}
			if rktletBin == "" {
				rktletBin, err = exec.LookPath("rktlet")
				if err != nil {
					log.Fatal("Unable to find rktlet binary. Put it in your PATH or use KUBE_SPAWN_RKTLET_BIN.")
				}
			}
		case "crio":
			// need crio, runc and conmon binaries
		}
	}
}

func main() {
	var err error
	var goPath string
	if goPath, err = utils.GetValidGoPath(); err != nil {
		log.Fatalf("invalid GOPATH %q: %v", goPath, err)
	}

	if err := cmdKubeSpawn.Execute(); err != nil {
		log.Fatal(err)
	}
}
