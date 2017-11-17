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

package nspawntool

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/kinvolk/kube-spawn/pkg/config"
	"github.com/kinvolk/kube-spawn/pkg/machinetool"
	"github.com/kinvolk/kube-spawn/pkg/utils"
)

func Run(cfg *config.ClusterConfiguration, mNo int) error {
	// This check is necessary to avoid panic with "index out of range".
	if mNo > len(cfg.Machines)-1 {
		return fmt.Errorf("cannot get a machine kubespawn%d", mNo)
	}

	if cfg.Machines[mNo].Running {
		return nil
	}

	if err := machinetool.Clone(cfg.Image, cfg.Machines[mNo].Name); err != nil {
		return errors.Wrap(err, "error cloning image")
	}

	lowerRoot, err := filepath.Abs(path.Join(cfg.KubeSpawnDir, cfg.Name, "rootfs"))
	if err != nil {
		return err
	}
	upperRoot, err := filepath.Abs(path.Join(cfg.KubeSpawnDir, cfg.Name, cfg.Machines[mNo].Name, "rootfs"))
	if err != nil {
		return err
	}

	args := []string{
		// TODO: v
		"-E", fmt.Sprintf("CNI_PATH=%s", os.Getenv("CNI_PATH")),
		"cnispawn",
		"-i",
		"--machine", cfg.Machines[mNo].Name,
		optionsOverlay("--overlay", "/etc", lowerRoot, upperRoot),
		optionsOverlay("--overlay", "/opt", lowerRoot, upperRoot),
		optionsOverlay("--overlay", "/usr/bin", lowerRoot, upperRoot),
	}

	args = append(args, optionsFromBindmountConfig(cfg.Bindmount)...)
	args = append(args, optionsFromBindmountConfig(cfg.Machines[mNo].Bindmount)...)

	c := utils.Command("systemd-run", args...)
	c.Stderr = os.Stderr

	// log.Printf(">>> runnning: %q", strings.Join(c.Args, " "))

	// stdout, err := c.StdoutPipe()
	// if err != nil {
	// 	return errors.Wrap(err, "error creating stdout pipe")
	// }
	// defer stdout.Close()

	if err := c.Start(); err != nil {
		return errors.Wrap(err, "error running cnispawn")
	}

	// cniDataJSON, err := ioutil.ReadAll(stdout)
	// if err != nil {
	// 	return errors.Wrap(err, "error reading cni data from stdin")
	// }

	// if _, err := cniversion.NewResult(cniversion.Current(), cniDataJSON); err != nil {
	// 	log.Printf("unexpected result output: %s", cniDataJSON)
	// 	return errors.Wrap(err, "unable to parse result")
	// }

	if err := c.Wait(); err != nil {
		// var cniError cnitypes.Error
		// if err := json.Unmarshal(cniDataJSON, &cniError); err != nil {
		// 	return errors.Wrap(err, "error unmarshaling cni error")
		// }
		// return errors.Wrap(&cniError, "error running cnispawn")
		return errors.Wrap(err, "error running cnispawn")
	}

	if err := waitMachinesRunning(cfg.Machines[mNo].Name); err != nil {
		return err
	}
	cfg.Machines[mNo].Running = true
	return nil
}

func waitMachinesRunning(machineName string) error {
	for retries := 0; retries <= 30; retries++ {
		if machinetool.IsRunning(machineName) {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return errors.Errorf("timeout waiting for %q to start", machineName)
}

func optionsOverlay(prefix, targetDir, lower, upper string) string {
	return fmt.Sprintf("%s=+%s:%s:%s:%s", prefix, targetDir, path.Join(lower, targetDir), path.Join(upper, targetDir), targetDir)
}
