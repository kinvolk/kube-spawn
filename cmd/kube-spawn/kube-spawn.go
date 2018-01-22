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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sys/unix"
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
	// set from ldflags to current git version during build
	version string

	printVersion bool

	cfgFile string
)

func init() {
	log.SetFlags(0)

	cobra.OnInitialize(initConfig)

	kubespawnCmd.Flags().BoolVarP(&printVersion, "version", "V", false, "Output version and exit")
	kubespawnCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default \"/etc/kube-spawn/config.yaml\")")
	kubespawnCmd.PersistentFlags().StringP("dir", "d", "/var/lib/kube-spawn", "Path to kube-spawn asset directory")
	kubespawnCmd.PersistentFlags().StringP("cluster-name", "c", "default", "Name for the cluster")

	viper.BindPFlags(kubespawnCmd.PersistentFlags())

	kubespawnCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cmdName := cmd.Use
		if cmdName == "create" || cmdName == "destroy" || cmdName == "start" || cmdName == "stop" || cmdName == "up" {
			if unix.Geteuid() != 0 {
				cmd.SilenceUsage = true
				return fmt.Errorf("root privileges required for command %q, aborting", cmdName)
			}
		}
		return nil
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		config := fmt.Sprintf("/etc/kube-spawn")
		viper.AddConfigPath(config)
	}
	viper.SetEnvPrefix("KUBE_SPAWN")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		log.Printf("Using config file %q", viper.ConfigFileUsed())
	}
}

func main() {
	if err := kubespawnCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
