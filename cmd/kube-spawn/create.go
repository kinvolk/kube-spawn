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

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/cache"
	"github.com/kinvolk/kube-spawn/pkg/cluster"
	"github.com/kinvolk/kube-spawn/pkg/utils/fs"
)

var (
	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new cluster environment",
		Example: `
# Create an environment to run a 3 node cluster from a hyperkube image
$ sudo -E kube-spawn create --nodes 3 --hyperkube-image 10.22.0.1:5000/my-hyperkube-amd64-image:my-test

# Create a cluster using rkt as the container runtime
$ sudo kube-spawn create --container-runtime rkt --rktlet-binary-path $GOPATH/src/github.com/kubernetes-incubator/rktlet/bin/rktlet`,
		Run: runCreate,
	}
)

func init() {
	kubespawnCmd.AddCommand(createCmd)

	createCmd.Flags().String("container-runtime", "docker", "Runtime to use for the cluster (can be docker or rkt)")
	createCmd.Flags().String("machinectl-image", "coreos", "Name of the machinectl image to use for the kube-spawn containers")
	createCmd.Flags().String("kubernetes-version", "v1.9.2", "Kubernetes version to install")
	createCmd.Flags().String("kubernetes-source-dir", "", "Path to directory with Kubernetes sources")
	createCmd.Flags().String("hyperkube-image", "", "Kubernetes hyperkube image to use (if unset, upstream k8s is installed)")
	createCmd.Flags().String("cni-plugin-dir", "/opt/cni/bin", "Path to directory with CNI plugins")
	createCmd.Flags().String("rkt-binary-path", "/usr/local/bin/rkt", "Path to rkt binary")
	createCmd.Flags().String("rkt-stage1-image-path", "/usr/local/bin/stage1-coreos.aci", "Path to rkt stage1-coreos.aci image")
	createCmd.Flags().String("rktlet-binary-path", "/usr/local/bin/rktlet", "Path to rktlet binary")

	viper.BindPFlags(createCmd.Flags())
}

func runCreate(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		log.Fatalf("Command create doesn't take arguments, got: %v", args)
	}

	kubespawnDir := viper.GetString("dir")
	clusterName := viper.GetString("cluster-name")
	clusterDir := path.Join(kubespawnDir, "clusters", clusterName)
	if exists, err := fs.PathExists(clusterDir); err != nil {
		log.Fatalf("Failed to stat directory %q: %s\n", err)
	} else if exists {
		log.Fatalf("Cluster directory exists already at %q", clusterDir)
	}

	// TODO
	if err := bootstrap.PathSupportsOverlay(kubespawnDir); err != nil {
		log.Fatalf("Unable to use overlayfs on underlying filesystem of %q: %v", kubespawnDir, err)
	}

	kluster, err := cluster.New(clusterDir, clusterName)
	if err != nil {
		log.Fatalf("Failed to create cluster object: %v", err)
	}

	clusterSettings := &cluster.ClusterSettings{
		KubernetesVersion:   viper.GetString("kubernetes-version"),
		KubernetesSourceDir: viper.GetString("kubernetes-source-dir"),
		CNIPluginDir:        viper.GetString("cni-plugin-dir"),
		ContainerRuntime:    viper.GetString("container-runtime"),
		RktBinaryPath:       viper.GetString("rkt-binary-path"),
		RktStage1ImagePath:  viper.GetString("rkt-stage1-image-path"),
		RktletBinaryPath:    viper.GetString("rktlet-binary-path"),
		HyperkubeImage:      viper.GetString("hyperkube-image"),
	}

	clusterCache, err := cache.New(path.Join(kubespawnDir, "cache"))
	if err != nil {
		log.Fatalf("Failed to create cache object: %v", err)
	}

	if err := kluster.Create(clusterSettings, clusterCache); err != nil {
		log.Fatalf("Failed to create cluster: %v", err)
	}

	log.Printf("Cluster %s created", clusterName)
}
