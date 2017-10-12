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
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/config"
	"github.com/kinvolk/kube-spawn/pkg/nspawntool"
	"github.com/kinvolk/kube-spawn/pkg/utils"
)

var (
	cmdSetup = &cobra.Command{
		Use:   "setup",
		Short: "Set up nodes bringing up nspawn containers",
		Run:   runSetup,
	}
)

func init() {
	cmdKubeSpawn.AddCommand(cmdSetup)
}

func runSetup(cmd *cobra.Command, args []string) {
	cfg, err := config.ReadConfigFromFile(viper.GetString("kube-spawn-dir"), viper.GetString("cluster-name"))
	if err != nil {
		log.Fatal(errors.Wrap(err, "unable to read config file"))
	}

	doSetup(cfg)
}

func doSetup(cfg *config.ClusterConfiguration) {
	if err := bootstrap.EnsureBridge(); err != nil {
		log.Fatalf("Error checking CNI bridge: %s", err)
	}

	if err := bootstrap.WriteNetConf(); err != nil {
		log.Fatalf("Error writing CNI configuration: %s", err)
	}

	log.Printf("Checking base image")
	if !bootstrap.MachineImageExists(baseImage) {
		log.Fatal("Base image not found.")
	}

	bootstrap.CreateSharedTmpdir()
	bootstrap.EnsureRequirements()

	doCheckK8sStableRelease(cfg.KubernetesVersion)
	if !utils.IsK8sDev(cfg.KubernetesVersion) {
		if err := bootstrap.DownloadK8sBins(cfg.KubernetesVersion, path.Join(cfg.KubeSpawnDir, "k8s")); err != nil {
			log.Fatalf("Error downloading k8s files: %s", err)
		}
	}

	// Several kubeadm configs or scripts must be written differently
	// according to k8s versions or container runtimes.
	if err := bootstrap.GenerateScripts(cfg.KubeSpawnDir, cfg.ContainerRuntime); err != nil {
		log.Fatal(err)
	}
	if err := bootstrap.GenerateConfigs(cfg.KubeSpawnDir, cfg.ContainerRuntime, cfg.KubernetesVersion); err != nil {
		log.Fatal(err)
	}

	// Copy config files from $PWD/etc to kubeSpawnDir/cName, without change.
	copyConfigFiles(path.Join(cfg.KubeSpawnDir, cfg.Name))

	// NOTE: workaround for making kubelet work with port-forward.
	// Ideally we should solve the port-forward issue by either
	// creating general add-ons based on torcx, or creating our own
	// container image, or at least building socat statically on our own.
	ksExtraDir := path.Join(cfg.KubeSpawnDir, "extras")
	if err := os.MkdirAll(ksExtraDir, os.FileMode(0755)); err != nil {
		log.Fatalf("Unable to create directory %q: %v.", ksExtraDir, err)
	}
	if err := bootstrap.DownloadSocatBin(ksExtraDir); err != nil {
		log.Fatalf("Error downloading socat files: %s", err)
	}

	// estimate get pool size based on sum of virtual image sizes.
	var poolSize int64
	var err error
	if poolSize, err = bootstrap.GetPoolSize(cfg.Image, cfg.Nodes); err != nil {
		// fail hard in case of error, to avoid running unnecessary nodes
		log.Fatalf("cannot get pool size: %v", err)
	}

	var nodesToRun []nspawntool.Node

	for i := 0; i < cfg.Nodes; i++ {
		var node = nspawntool.Node{
			Name:       bootstrap.GetNodeName(i),
			K8sVersion: cfg.KubernetesVersion,
			Runtime:    cfg.ContainerRuntime,
		}
		if !bootstrap.MachineImageExists(node.Name) {
			if err := bootstrap.NewNode(cfg.Image, node.Name); err != nil {
				log.Fatalf("Error cloning base image: %s", err)
			}
		}
		if bootstrap.IsNodeRunning(node.Name) {
			continue
		}
		nodesToRun = append(nodesToRun, node)
	}
	if len(nodesToRun) == 0 {
		log.Printf("No nodes to create. stop")
		os.Exit(1)
	}

	if err := bootstrap.EnlargeStoragePool(poolSize); err != nil {
		log.Printf("Warning: cannot enlarge storage pool: %v", err)
	}

	for _, node := range nodesToRun {
		if err := node.Run(cfg); err != nil {
			log.Fatalf("Error running node: %v", err)
		}

		if err := node.Bootstrap(); err != nil {
			log.Fatalf("Error running bootstrap script: %v", err)
		}

		log.Printf("Success! %s started.", node.Name)
	}

	log.Print("All nodes are running. Use machinectl to login/stop/etc.")
	log.Printf("KUBECONFIG is in %q", path.Join(cfg.KubeSpawnDir, cfg.Name))
}

func copyConfigFiles(basedir string) {
	etcSrc := utils.LookupPwd("$PWD/etc")
	etcDst := filepath.Join(basedir, "etc")

	fName := "daemon.json"
	if _, err := utils.CopyFileToDir(filepath.Join(etcSrc, fName), etcDst); err != nil {
		log.Fatalf("Error copying file %s: %v", fName, err)
	}

	fName = "kube_tmpfiles_kubelet.conf"
	if _, err := utils.CopyFileToDir(filepath.Join(etcSrc, fName), etcDst); err != nil {
		log.Fatalf("Error copying file %s: %v", fName, err)
	}

	fName = "weave_50-weave.network"
	if _, err := utils.CopyFileToDir(filepath.Join(etcSrc, fName), etcDst); err != nil {
		log.Fatalf("Error copying file %s: %v", fName, err)
	}

	if k8sruntime == "docker" {
		fName = "docker_20-kubeadm-extra-args.conf"
		if _, err := utils.CopyFileToDir(filepath.Join(etcSrc, fName), etcDst); err != nil {
			log.Fatalf("Error copying file %s: %v", fName, err)
		}
	} else if k8sruntime == "rkt" {
		fName = "rktlet.service"
		if _, err := utils.CopyFileToDir(filepath.Join(etcSrc, fName), etcDst); err != nil {
			log.Fatalf("Error copying file %s: %v", fName, err)
		}
	}
}
