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

	"github.com/kinvolk/kube-spawn/pkg/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	kubespawnCmd = &cobra.Command{
		Use:   "kube-spawn",
		Short: "kube-spawn is a tool for creating a multi-node dev Kubernetes cluster",
		Long: `kube-spawn is a tool for creating a multi-node dev Kubernetes cluster.
You can run a release-version cluster or spawn one from your local Kubernetes repository`,
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

	version string

	printVersion bool
)

func init() {
	kubespawnCmd.PersistentFlags().StringP("dir", "d", "", "Path to kube-spawn asset directory")
	kubespawnCmd.PersistentFlags().StringP("cluster-name", "c", "", "Name for the cluster")
	viper.BindPFlags(kubespawnCmd.PersistentFlags())

	kubespawnCmd.Flags().BoolVarP(&printVersion, "version", "V", false, "Output version information")

	kubespawnCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		log.SetFlags(0)
	}
}

func main() {
	if err := kubespawnCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func loadConfig() *config.ClusterConfiguration {
	cfg, err := config.LoadConfig(viper.GetString("cluster-name"))
	if err != nil {
		log.Fatal(errors.Wrap(err, "error loading config"))
	}
	log.Printf("using config from %s/%s", cfg.KubeSpawnDir, cfg.Name)
	return cfg
}

func saveConfig(cfg *config.ClusterConfiguration) {
	if err := config.WriteConfigToFile(cfg); err != nil {
		log.Fatal(errors.Wrap(err, "failed to write to config file"))
	}
}
