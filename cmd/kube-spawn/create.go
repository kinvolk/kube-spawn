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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sys/unix"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/config"
	"github.com/kinvolk/kube-spawn/pkg/utils"
)

var (
	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Generate the environment for a cluster",
		Long: `Generate the environment for a cluster.
If you change 'kspawn.toml' this needs to be run again.`,
		Example: `
# Create an environment to run a 3 node cluster initialized with components from $GOPATH/k8s.io/kubernetes
$ sudo -E kube-spawn create --nodes 3 --dev -t mytag

# Create a cluster environment using rkt as the container runtime
# You can specify paths to the binaries necessary using environment variables (in case they are not in your PATH)
$ sudo -E \
	KUBE_SPAWN_RKT_BIN=/opt/bin/rkt \
	KUBE_SPAWN_RKTLET_BIN=/opt/bin/rktlet \
	KUBE_SPAWN_RKT_STAGE1_IMAGE=/opt/bin/stage1-coreos.aci \
	kube-spawn create --container-runtime rkt`,
		Run: runCreate,
	}
)

func init() {
	kubespawnCmd.AddCommand(createCmd)

	// do not set defaults here
	// intead use:
	// pkg/config/defaults.go
	// and call from the if uninitialized {} block below
	//
	createCmd.Flags().StringP("container-runtime", "r", "", "runtime to use for the spawned cluster (docker or rkt)")
	createCmd.Flags().String("kubernetes-version", "", `version kubernetes used to initialize the cluster. Irrelevant if used with --dev. Only accepts semantic version strings like "v1.7.5"`)
	createCmd.Flags().StringP("hyperkube-tag", "t", "latest", `Docker tag of the hyperkube image to use. Only with --dev`)
	createCmd.Flags().Bool("dev", false, "create a cluster from a local build of Kubernetes")
	createCmd.Flags().IntP("nodes", "n", 0, "number of nodes to spawn")
	createCmd.Flags().StringP("image", "i", "", "base image for nodes")
	viper.BindPFlags(createCmd.Flags())

	viper.BindEnv("runtime-config.rkt.rkt-bin", "KUBE_SPAWN_RKT_BIN")
	viper.BindEnv("runtime-config.rkt.stage1-image", "KUBE_SPAWN_RKT_STAGE1_IMAGE")
	viper.BindEnv("runtime-config.rkt.rktlet-bin", "KUBE_SPAWN_RKTLET_BIN")

	viper.BindEnv("runtime-config.crio.crio-bin", "KUBE_SPAWN_CRIO_BIN")
	viper.BindEnv("runtime-config.crio.runc-bin", "KUBE_SPAWN_RUNC_BIN")
	viper.BindEnv("runtime-config.crio.conmon-bin", "KUBE_SPAWN_CONMON_BIN")

	config.SetDefaults_Viper(viper.GetViper())
}

func runCreate(cmd *cobra.Command, args []string) {
	if unix.Geteuid() != 0 {
		log.Fatalf("non-root user cannot create clusters. abort.")
	}

	if len(args) > 0 {
		log.Fatalf("too many arguments: %v", args)
	}

	doCreate()
}

func doCreate() {
	cfg, err := config.LoadConfig()
	if err != nil {
		// ignore if config not found
		// it means we started from scratch and need to generate one
		if !config.IsNotFound(err) {
			log.Fatal(errors.Wrap(err, "unable to load config"))
		}
	}
	log.Printf("creating cluster environment %q", cfg.Name)
	if cfg.DevCluster {
		log.Printf("spawning from local kubernetes build")
	} else {
		log.Printf("spawning kubernetes version %q", cfg.KubernetesVersion)
	}
	if cfg.ContainerRuntime != config.RuntimeDocker {
		log.Printf("spawning with container runtime %q", cfg.ContainerRuntime)
	}

	if utils.CheckVersionConstraint(cfg.KubernetesVersion, "<1.7.5") {
		log.Fatal("minimum supported version is 'v1.7.5'")
	}

	// download files into cache
	if !cfg.DevCluster {
		if err := bootstrap.DownloadK8sBins(cfg); err != nil {
			log.Fatal(err)
		}
	}
	if err := bootstrap.DownloadSocatBin(cfg); err != nil {
		log.Fatal(err)
	}

	if err := config.SetDefaults_Kubernetes(cfg); err != nil {
		log.Fatal(errors.Wrap(err, "error settting kubernetes defaults"))
	}

	if err := config.SetDefaults_BindmountConfiguration(cfg); err != nil {
		log.Fatal(errors.Wrap(err, "error setting bindmount defaults"))
	}

	if err := config.SetDefaults_RuntimeConfiguration(cfg); err != nil {
		log.Fatal(errors.Wrap(err, "error setting container runtime defaults"))
	}

	// note: this is a workaround the keyctl issue with runc
	// can be removed when systemd v235 is common
	// TODO: move this somewhere else and reuse code from utils pkg
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		log.Fatal("GOPATH was not set")
	}
	cfg.Copymap = append(cfg.Copymap, config.Pathmap{
		Dst: "/usr/bin/kube-spawn-runc",
		Src: path.Join(goPath, "src/github.com/kinvolk/kube-spawn/kube-spawn-runc"),
	})

	if cfg.Image == config.DefaultBaseImage {
		if err := bootstrap.PrepareCoreosImage(); err != nil {
			log.Fatal(errors.Wrap(err, "error setting up default base image"))
		}
	}

	// TODO: check config + env
	// - check version
	// - check version of k8s binaries
	// - cni bridge works
	// - base image exists
	// - ??? coreos version correct
	// - overlayfs works
	// - conntrack hashsize
	// - iptables setup correct
	// - selinux setup correct
	// if err := checks.RunCreateChecks(cfg); err != nil {
	// 	log.Fatal(errors.Wrap(err, "check failed"))
	// }

	log.Print("ensuring environment")
	if err := bootstrap.EnsureRequirements(cfg); err != nil {
		log.Fatal(err)
	}

	if err := bootstrap.PathSupportsOverlay(cfg.KubeSpawnDir); err != nil {
		log.Fatalf("unable to use overlayfs on %q: %v. Try to pass a directory with a different filesystem (like ext4 or XFS) to --dir.", cfg.KubeSpawnDir, err)
	}

	log.Print("generating scripts")
	if err := bootstrap.GenerateScripts(cfg); err != nil {
		log.Fatal(errors.Wrap(err, "error generating files"))
	}

	log.Print("copy files into environment")
	if err := bootstrap.CopyFiles(cfg); err != nil {
		log.Fatal(errors.Wrap(err, "error copying files"))
	}

	saveConfig(cfg)
	log.Println("created cluster config")
}
