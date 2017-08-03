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

	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/nspawntool"
)

var (
	numNodes     int
	baseImage    string
	kubeSpawnDir string

	cmdSetup = &cobra.Command{
		Use:   "setup",
		Short: "Set up nodes bringing up nspawn containers",
		Run:   runSetup,
	}
)

func init() {
	cmdKubeSpawn.AddCommand(cmdSetup)

	cmdSetup.Flags().IntVarP(&numNodes, "nodes", "n", 1, "number of nodes to spawn")
	cmdSetup.Flags().StringVarP(&baseImage, "image", "i", "coreos", "base image for nodes")
	cmdSetup.Flags().StringVarP(&kubeSpawnDir, "kube-spawn-dir", "d", "", "path to directory where .kube-spawn directory is located")
}

func checkK8sStableRelease(k8srel string) bool {
	v, err := semver.NewVersion(k8srel)
	if err != nil {
		// fallback to default version
		v, _ = semver.NewVersion(k8sStableVersion)
	}

	c, err := semver.NewConstraint(">=" + k8sStableVersion)
	if err != nil {
		log.Printf("Cannot get constraint for >= %s: %v", k8sStableVersion, err)
		return false
	}

	return c.Check(v)
}

func doCheckK8sStableRelease(k8srel string) {
	if !checkK8sStableRelease(k8srelease) {
		log.Printf("WARNING: K8s with version <%s is not compatible with kube-spawn.",
			k8sStableVersion)
		log.Printf("It's highly recommended to upgrade K8s to 1.7 or newer.")
		// Print a kind warning, and continue to run.
		// It should still allow kubeadm to run with Kubernetes <1.7,
		// to be able to allow end users to flexibly handle various cases.
	}
}

func runSetup(cmd *cobra.Command, args []string) {
	doSetup(numNodes, baseImage, kubeSpawnDir)
}

func doSetup(numNodes int, baseImage, kubeSpawnDir string) {
	if err := bootstrap.EnsureBridge(); err != nil {
		log.Fatalf("Error checking CNI bridge: %s", err)
	}

	if err := bootstrap.WriteNetConf(); err != nil {
		log.Fatalf("Error writing CNI configuration: %s", err)
	}

	log.Printf("Checking base image")
	if baseImage == "" {
		log.Fatal("No base image specified.")
	}
	if !bootstrap.MachineImageExists(baseImage) {
		log.Fatal("Base image not found.")
	}

	bootstrap.CreateSharedTmpdir()
	bootstrap.EnsureRequirements()

	doCheckK8sStableRelease(k8srelease)

	if !isDev(k8srelease) {
		if err := bootstrap.DownloadK8sBins(k8srelease, "./k8s"); err != nil {
			log.Fatalf("Error downloading k8s files: %s", err)
		}
	}

	// NOTE: workaround for making kubelet work with port-forward.
	// Ideally we should solve the port-forward issue by either
	// creating general add-ons based on torcx, or creating our own
	// container image, or at least building socat statically on our own.
	ksExtraDir := ".kube-spawn/extras"
	if err := os.MkdirAll(ksExtraDir, os.FileMode(0755)); err != nil {
		log.Fatalf("Unable to create directory %q: %v.", ksExtraDir, err)
	}
	if err := bootstrap.DownloadSocatBin(ksExtraDir); err != nil {
		log.Fatalf("Error downloading socat files: %s", err)
	}

	var nodesToRun []string

	for i := 0; i < numNodes; i++ {
		name := bootstrap.GetNodeName(i)
		if !bootstrap.MachineImageExists(name) {
			if err := bootstrap.NewNode(baseImage, name); err != nil {
				log.Fatalf("Error cloning base image: %s", err)
			}
		}
		if bootstrap.IsNodeRunning(name) {
			continue
		}
		nodesToRun = append(nodesToRun, name)
	}
	if len(nodesToRun) == 0 {
		log.Printf("No nodes to create. stop")
		os.Exit(1)
	}

	if err := bootstrap.EnlargeStoragePool(baseImage, len(nodesToRun)); err != nil {
		log.Printf("Warning: cannot enlarge storage pool: %s", err)
	}

	for _, name := range nodesToRun {
		if err := nspawntool.RunNode(k8srelease, name, kubeSpawnDir); err != nil {
			log.Fatalf("Error running node: %s", err)
		}

		if err := nspawntool.RunBootstrapScript(name); err != nil {
			log.Fatalf("Error running bootstrap script: %s", err)
		}

		log.Printf("Success! %s started.", name)
	}

	log.Printf("All nodes are running. Use machinectl to login/stop/etc.")
}
