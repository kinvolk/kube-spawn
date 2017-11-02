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

	"github.com/kinvolk/kube-spawn/pkg/config"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	setEnvCmd = &cobra.Command{
		Use:   "set-env",
		Short: "Set a cluster environment to interact with",
		Long: `Set a cluster environment to interact with.
The same as running every command with the --cluster-name flag`,
		Run: runSetEnv,
	}
)

func init() {
	kubespawnCmd.AddCommand(setEnvCmd)
}

func runSetEnv(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		log.Fatal("no environment given")
	}
	doSetEnv(args[0])
}

func doSetEnv(clusterName string) {
	if err := config.SetCurrentEnv(clusterName); err != nil {
		log.Fatal(errors.Wrap(err, "failed to write current env"))
	}
}
