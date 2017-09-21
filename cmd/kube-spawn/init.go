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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/distribution"
	"github.com/kinvolk/kube-spawn/pkg/nspawntool"
	"github.com/kinvolk/kube-spawn/pkg/script"
	"github.com/kinvolk/kube-spawn/pkg/utils"
)

const pushImageRetries int = 10

var (
	cmdInit = &cobra.Command{
		Use:   "init",
		Short: "Initialize cluster running kubeadm",
		Run:   runInit,
	}
)

func init() {
	cmdKubeSpawn.AddCommand(cmdInit)
}

func runInit(cmd *cobra.Command, args []string) {
	doInit()
}

func doInit() {
	doCheckK8sStableRelease(k8srelease)

	if utils.IsK8sDev(k8srelease) {
		// we don't need to run a docker registry
		if err := distribution.StartRegistry(); err != nil {
			log.Fatalf("Error starting registry: %s", err)
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
	}

	nodes, err := bootstrap.GetRunningNodes()
	if err != nil {
		log.Fatalf("Error listing running nodes: %s", err)
	}
	if len(nodes) == 0 {
		log.Fatal("No node running. Is systemd-nspawn running correctly?")
	}

	bootstrap.EnsureRequirements()

	log.Println("Note: init on master can take a couple of minutes until every k8s pod came up.")

	os.Remove(filepath.Join(kubeSpawnDir, "default/token"))

	if err := nspawntool.InitializeMaster(k8srelease, nodes[0].Name); err != nil {
		log.Fatalf("Error initializing master node %s: %s", nodes[0].Name, err)
	}

	token, err := getToken()
	if err != nil {
		log.Fatalf("Error reading token: %v", err)
	}

	switch k8sruntime {
	case "", "docker":
		kubeadmContainerRuntime = "docker"
	case "rkt":
		kubeadmContainerRuntime = "rktlet"
	case "crio":
		kubeadmContainerRuntime = "crio"
	default:
		log.Fatalf("runtime %s is not supported", k8sruntime)
	}

	outbuf := script.GetKubeadmJoin(script.KubeadmJoinOpts{
		ContainerRuntime: kubeadmContainerRuntime,
		Token:            token,
		MasterIP:         nodes[0].IP,
	})
	if outbuf == nil {
		log.Fatalf("Error generating kubeadm join script")
	}

	joinScript := filepath.Join(kubeSpawnDir, "scripts/join.sh")
	if err := ioutil.WriteFile(joinScript, outbuf.Bytes(), os.FileMode(0755)); err != nil {
		log.Fatalf("Error writing script file %s: %v", joinScript, err)
	}

	for i, node := range nodes {
		if i != 0 {
			if err := nspawntool.JoinNode(k8srelease, node.Name, nodes[0].IP); err != nil {
				log.Fatalf("Error joining worker node %s: %s", node.Name, err)
			}
		}
	}

	log.Println("Note: For kubectl to work, please set $KUBECONFIG to " + utils.GetValidKubeConfig())
}

func getToken() (string, error) {
	buf, err := ioutil.ReadFile(filepath.Join(kubeSpawnDir, "default/token"))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(buf)), nil
}
