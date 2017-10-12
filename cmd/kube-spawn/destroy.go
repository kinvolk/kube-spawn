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

	"github.com/kinvolk/kube-spawn/pkg/config"
)

var (
	destroyCmd = &cobra.Command{
		Use: "destroy",
		// Aliases: []string{"setup, up"},
		Short: "Start the nodes of a generated cluster. You should have run `kube-spawn create` before this",
		Run:   runDestroy,
	}
)

func init() {
	kubespawnCmd.AddCommand(destroyCmd)
}

func runDestroy(cmd *cobra.Command, args []string) {
	cfg := loadConfig()
	doDestroy(cfg)
}

func doDestroy(cfg *config.ClusterConfiguration) {
	log.Printf("destroying cluster %q", cfg.Name)

	doStop(cfg, true)

	cDir := path.Join(cfg.KubeSpawnDir, cfg.Name)
	if err := os.RemoveAll(cDir); err != nil {
		log.Fatal(errors.Wrapf(err, "error removing cluster dir at %q", cDir))
	}
	log.Printf("%q destroyed", cfg.Name)
}
