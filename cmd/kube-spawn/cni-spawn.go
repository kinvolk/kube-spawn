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
	"os"

	"github.com/spf13/cobra"

	"github.com/kinvolk/kube-spawn/pkg/cnispawn"
)

var (
	cniSpawnCmd = &cobra.Command{
		Use:    "cni-spawn",
		Short:  "Spawn systemd-nspawn containers in a new network namespace",
		Hidden: true,
		Run:    runCNISpawn,
	}
)

func init() {
	kubespawnCmd.AddCommand(cniSpawnCmd)
}

func runCNISpawn(cmd *cobra.Command, args []string) {
	if err := cnispawn.Spawn(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
