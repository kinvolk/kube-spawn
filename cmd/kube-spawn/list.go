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
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all kube-spawn clusters",
		Run:   runList,
	}
)

func init() {
	kubespawnCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		log.Fatalf("Command list doesn't take arguments, got: %v", args)
	}

	clusterDir := path.Join(viper.GetString("dir"), "clusters/")

	entries, err := ioutil.ReadDir(clusterDir)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to read cluster directory: %v", err)
	}

	if len(entries) == 0 {
		log.Printf("No clusters yet")
	} else {
		fmt.Println("Available clusters:")
		for _, entry := range entries {
			fmt.Printf(" %s\n", entry.Name())
		}
	}
}
