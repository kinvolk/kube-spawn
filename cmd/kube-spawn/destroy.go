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

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/config"
)

var (
	destroyCmd = &cobra.Command{
		Use:   "destroy",
		Short: "Remove the cluster environment",
		Long: `Remove the cluster environment.
Stops the cluster if it it running`,
		Run: runDestroy,
	}
)

func init() {
	kubespawnCmd.AddCommand(destroyCmd)
}

func runDestroy(cmd *cobra.Command, args []string) {
	if len(args) > 0 {
		log.Fatalf("too many arguments: %v", args)
	}

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
	RemoveCniConfig()
	log.Printf("%q destroyed", cfg.Name)
}

func RemoveCniConfig() {
	if err := os.RemoveAll(bootstrap.VarLibCniDir); err != nil {
		log.Printf("cannot remove %q: %v", bootstrap.VarLibCniDir, err)
	}
	if err := os.RemoveAll(bootstrap.NspawnNetPath); err != nil {
		log.Printf("cannot remove %q: %v", bootstrap.NspawnNetPath, err)
	}
}
