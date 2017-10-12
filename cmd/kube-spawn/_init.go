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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/config"
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
	cfg, err := config.ReadConfigFromFile(viper.GetString("kube-spawn-dir"), viper.GetString("cluster-name"))
	if err != nil {
		log.Fatal(errors.Wrap(err, "unable to read config file"))
	}

	doInit(cfg)
}

func doInit(cfg *config.ClusterConfiguration) {
	doCheckK8sStableRelease(cfg.KubernetesVersion)

	if utils.IsK8sDev(cfg.KubernetesVersion) {
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

	if len(cfg.Machines) == 0 {
		log.Fatal("No node running. Is systemd-nspawn running correctly?")
	}

	bootstrap.EnsureRequirements()

	log.Println("Note: init on master can take a couple of minutes until every k8s pod came up.")

	os.Remove(filepath.Join(cfg.KubeSpawnDir, cfg.Name, "token"))

	if err := nspawntool.InitializeMaster(cfg.KubernetesVersion, cfg.Machines[0].Name); err != nil {
		log.Fatalf("Error initializing master node %s: %s", cfg.Machines[0].Name, err)
	}

	token, err := getToken()
	if err != nil {
		log.Fatalf("Error reading token: %v", err)
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

func writeKubeadmJoin(cfg *config.ClusterConfiguration) error {
	switch cfg.ContainerRuntime {
	case "", "docker":
		kubeadmContainerRuntime = "docker"
	case "rkt":
		kubeadmContainerRuntime = "rktlet"
	case "crio":
		kubeadmContainerRuntime = "crio"
	default:
		return fmt.Errorf("runtime %s is not supported", k8sruntime)
	}

	outbuf, err := script.GetKubeadmJoin(script.KubeadmJoinOpts{
		ContainerRuntime: kubeadmContainerRuntime,
		Token:            cfg.Token,
		MasterIP:         cfg.Machines[0].IP,
	})
	if err != nil {
		errors.Wrap(err, "error generating kubeadm join script")
	}

	joinScript := filepath.Join(cfg.KubeSpawnDir, cfg.Name, "rootfs/opt/kube-spawn/join.sh")
	if err := ioutil.WriteFile(joinScript, outbuf.Bytes(), os.FileMode(0755)); err != nil {
		return errors.Wrapf(err, "error writing script file %q", joinScript)
	}
	return nil
}
