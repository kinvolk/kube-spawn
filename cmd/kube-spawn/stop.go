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
)

var (
	stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop nodes by turning off machines",
		Run:   runStop,
	}

	flagForce bool
)

func init() {
	kubespawnCmd.AddCommand(stopCmd)
	stopCmd.Flags().BoolVarP(&flagForce, "force", "f", false, "terminate machines instead of trying graceful shutdown")
}

func runStop(cmd *cobra.Command, args []string) {
	cfg := loadConfig()
	doStop(cfg, flagForce)
}

func doStop(cfg *config.ClusterConfiguration, force bool) {
	var forceTxt = ""
	if force {
		forceTxt = "force "
	}
	log.Printf("%sstopping %d machines", forceTxt, len(cfg.Machines))

	if !force && config.RunningMachines(cfg) == 0 {
		log.Print("nothing to stop")
		return
	}

	stopMachines(cfg, force)
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
			if !force {
				// graceful stop
				if err := machinetool.Poweroff(cfg.Machines[i].Name); err != nil {
					if !machinetool.IsNotKnown(err) {
						log.Print(errors.Wrapf(err, "error powering off machine %q, maybe try with `kube-spawn stop -f`", cfg.Machines[i].Name))
						return
					}
				}
			} else {
				if err := machinetool.Terminate(cfg.Machines[i].Name); err != nil {
					if !machinetool.IsNotKnown(err) {
						log.Print(errors.Wrapf(err, "error terminating machine %q", cfg.Machines[i].Name))
						return
					}
				}
			}
			cfg.Machines[i].Running = false
			cfg.Machines[i].IP = ""
			log.Printf("%q stopped", cfg.Machines[i].Name)
		}(i)
	}
	wg.Wait()
}

func removeImages(cfg *config.ClusterConfiguration) {
	var wg sync.WaitGroup
	wg.Add(len(cfg.Machines))
	for i := 0; i < len(cfg.Machines); i++ {
		go func(i int) {
			defer wg.Done()
			// clean up image
			if machinetool.ImageExists(cfg.Machines[i].Name) {
				if err := removeImage(cfg.Machines[i].Name); err != nil {
					log.Print(err)
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
			return nil
		} else {
			time.Sleep(500 * time.Millisecond)
		}
	}
	return errors.Wrapf(err, "error removing machine image for %q", machineName)
}
