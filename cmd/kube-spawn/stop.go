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
	"sync"
	"time"

	"github.com/kinvolk/kube-spawn/pkg/config"
	"github.com/kinvolk/kube-spawn/pkg/machinetool"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var (
	stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop the running cluster",
		Run:   runStop,
	}

	flagForce bool
)

func init() {
	kubespawnCmd.AddCommand(stopCmd)
	stopCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "terminate machines instead of trying graceful shutdown")
}

func runStop(cmd *cobra.Command, args []string) {
	if unix.Geteuid() != 0 {
		log.Fatalf("non-root user cannot stop clusters. abort.")
	}

	if len(args) > 0 {
		log.Fatalf("too many arguments: %v", args)
	}

	cfg := loadConfig()
	doStop(cfg, flagForce)
}

func doStop(cfg *config.ClusterConfiguration, force bool) {
	var forceTxt = ""
	if force {
		forceTxt = "force "
	}
	log.Printf("%sstopping %d machines", forceTxt, len(cfg.Machines))

	if config.RunningMachines(cfg) != 0 {
		stopMachines(cfg, force)
	} else {
		log.Print("nothing to stop")
	}

	removeImages(cfg)

	// clear cluster config
	cfg.Token = ""
	saveConfig(cfg)
}

func stopMachines(cfg *config.ClusterConfiguration, force bool) {
	var wg sync.WaitGroup
	wg.Add(len(cfg.Machines))
	for i := 0; i < len(cfg.Machines); i++ {
		go func(i int) {
			defer wg.Done()
			if err := doGracefulStop(cfg.Machines[i].Name, force); err != nil {
				return
			}
			cfg.Machines[i].Running = false
			// machinectl output for machines with no IP
			// is '-', hence use '-' here was well
			cfg.Machines[i].IP = "-"
			log.Printf("%q stopped", cfg.Machines[i].Name)
		}(i)
	}
	wg.Wait()
}

func doGracefulStop(machineName string, force bool) error {
	if !force {
		for retries := 0; retries < 5; retries++ {
			// graceful stop
			if err := machinetool.Poweroff(machineName); err != nil {
				if !machinetool.IsNotKnown(err) {
					log.Print(errors.Wrapf(err, "error powering off machine %q, maybe try with `kube-spawn stop -f`", machineName))
					return err
				}
				time.Sleep(500 * time.Millisecond)
				continue
			}
			return nil
		}
		log.Printf("Tried to stop %s 5 times, but it didn't work, terminating.", machineName)
		// fall back to force shutdown
	}

	// Either it's force mode from the beginning,
	// or it's a fallback from a retry loop of a graceful stop.
	if err := machinetool.Terminate(machineName); err != nil {
		if !machinetool.IsNotKnown(err) {
			log.Print(errors.Wrapf(err, "error terminating machine %q", machineName))
			return err
		}
	}

	return nil
}

func removeImages(cfg *config.ClusterConfiguration) {
	var wg sync.WaitGroup
	wg.Add(len(cfg.Machines))
	for i := 0; i < len(cfg.Machines); i++ {
		go func(i int) {
			defer wg.Done()
			// clean up image
			machName := cfg.Machines[i].Name
			if machinetool.ImageExists(machName) {
				for retries := 0; retries < 15; retries++ {
					if err := removeImage(machName); err != nil {
						log.Printf("failed to remove image %q: %v. retrying...", machName, err)
						time.Sleep(500 * time.Millisecond)
						continue
					}
					log.Printf("successfully removed image %q", machName)
					break
				}
			}
		}(i)
	}
	wg.Wait()
}

func removeImage(machineName string) error {
	var err error
	for retries := 0; retries < 5; retries++ {
		if err = machinetool.RemoveImage(machineName); err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		} else {
			return nil
		}
	}
	return errors.Wrapf(err, "error removing machine image for %q", machineName)
}
