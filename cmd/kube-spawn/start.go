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
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/config"
	"github.com/kinvolk/kube-spawn/pkg/distribution"
	"github.com/kinvolk/kube-spawn/pkg/machinetool"
	"github.com/kinvolk/kube-spawn/pkg/nspawntool"
	"github.com/kinvolk/kube-spawn/pkg/utils"
)

var (
	startCmd = &cobra.Command{
		Use: "start",
		// Aliases: []string{"setup, up"},
		Short: "Start the cluster. You should have run 'kube-spawn create' before this",
		Run:   runStart,
	}

	flagSkipInit bool
)

func init() {
	kubespawnCmd.AddCommand(startCmd)
	startCmd.Flags().BoolVar(&flagSkipInit, "skip-cluster-init", false, "skips the initialization of a Kubernetes-Cluster with kubeadm")
}

func runStart(cmd *cobra.Command, args []string) {
	cfg := loadConfig()
	doStart(cfg, flagSkipInit)
}

func doStart(cfg *config.ClusterConfiguration, skipInit bool) {
	if config.RunningMachines(cfg) == cfg.Nodes {
		log.Print("cluster is up")
		return
	}

	log.Printf("using %q base image from /var/lib/machines", cfg.Image)
	log.Printf("spawning cluster %q (%d machines)", cfg.Name, cfg.Nodes)

	resizeMachineDir(cfg.Image, cfg.Nodes)

	var wg sync.WaitGroup
	wg.Add(cfg.Nodes)
	for i := 0; i < cfg.Nodes; i++ {
		go func(i int) {
			defer wg.Done()
			log.Printf("waiting for machine %q to start up", config.MachineName(cfg.Name, i))
			if err := nspawntool.Run(cfg, i); err != nil {
				log.Print(errors.Wrap(err, "failed to start machine"))
				return
			}
			log.Printf("machine %q started", cfg.Machines[i].Name)

			log.Printf("bootstrapping %q", cfg.Machines[i].Name)
			if err := machinetool.Exec(cfg.Machines[i].Name, "/opt/kube-spawn/bootstrap.sh"); err != nil {
				log.Fatal(errors.Wrap(err, "failed to bootstrap"))
			}
		}(i)
	}
	wg.Wait()
	saveConfig(cfg)

	if config.RunningMachines(cfg) != cfg.Nodes {
		log.Print("not all machines started")
		return
	}
	log.Printf("cluster %q started", cfg.Name)

	if skipInit {
		return
	}

	if cfg.DevCluster {
		// bring up the docker registry
		// needed for supplying the hyperkube to kubeadm in the machines
		if err := distribution.StartRegistry(); err != nil {
			log.Fatal(errors.Wrap(err, "error starting registry"))
		}
		var err error
		for i := 0; i < distribution.PushImageRetries; i++ {
			err = distribution.PushImage(cfg.HyperkubeTag)
			if err == nil {
				break
			}
			time.Sleep(1 * time.Second)
		}
		if err != nil {
			log.Fatal(errors.Wrap(err, "error pushing hyperkube image"))
		}
	}

	// cluster init with kubeadm from here
	initMasterNode(cfg)
	if cfg.Nodes > 1 {
		joinWorkerNodes(cfg)
	}
	log.Printf("cluster %q initialized", cfg.Name)
	log.Println("Note: For kubectl to work, please set $KUBECONFIG:")
	log.Printf("export KUBECONFIG=%s\n", utils.GetKubeconfigPath(cfg.KubeSpawnDir, cfg.Name))
	saveConfig(cfg)
}

func initMasterNode(cfg *config.ClusterConfiguration) error {
	log.Println("[!] note: init on master can take a couple of minutes until all pods are up")
	if err := nspawntool.InitializeMaster(cfg); err != nil {
		log.Fatal(errors.Wrap(err, "failed to initialize master node"))
	}

	return nil
}

func joinWorkerNodes(cfg *config.ClusterConfiguration) {
	var wg sync.WaitGroup
	wg.Add(cfg.Nodes - 1)
	for i := 1; i < cfg.Nodes; i++ {
		go func(i int) {
			defer wg.Done()
			if err := nspawntool.JoinNode(cfg, i); err != nil {
				log.Fatal(errors.Wrapf(err, "failed to join %q", cfg.Machines[i].Name))
			}
		}(i)
	}
	wg.Wait()
}

func resizeMachineDir(baseImage string, nodesN int) {
	// estimate get pool size based on sum of virtual image sizes.
	var poolSize int64
	var err error

	if !bootstrap.PoolImageExists() {
		return
	}

	if poolSize, err = bootstrap.GetPoolSize(baseImage, nodesN); err != nil {
		// fail hard in case of error, to avoid running unnecessary nodes
		log.Fatalf("cannot get pool size: %v", err)
	}

	if err := bootstrap.EnlargeStoragePool(poolSize); err != nil {
		log.Printf("[!] warning: cannot enlarge storage pool: %v", err)
	}
}
