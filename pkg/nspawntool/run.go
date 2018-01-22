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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	// "strings"
	"time"

	cnitypes "github.com/containernetworking/cni/pkg/types"
	cniversion "github.com/containernetworking/cni/pkg/version"
	"github.com/pkg/errors"

	"github.com/kinvolk/kube-spawn/pkg/machinectl"
	"github.com/kinvolk/kube-spawn/pkg/utils"
)

func Run(machinectlImage, lowerRootPath, upperRootPath, machineName, cniPluginDir string) error {
	// TODO(schu): add a check if a machine with this name is runnin already

	if err := machinectl.Clone(machinectlImage, machineName); err != nil {
		return errors.Wrap(err, "error cloning image")
	}

	if err := os.MkdirAll(lowerRootPath, 0755); err != nil {
		return err
	}

	if err := os.MkdirAll(upperRootPath, 0755); err != nil {
		return err
	}

	// Create all directories which will be overlay mounts (see below)
	// Otherwise systemd-nspawn will fail:
	// `overlayfs: failed to resolve '/var/lib/kube-spawn/...'`
	if err := os.MkdirAll(path.Join(upperRootPath, "etc"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Join(upperRootPath, "opt"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(path.Join(upperRootPath, "usr/bin"), 0755); err != nil {
		return err
	}

	// Create all directories that will be bind mounted
	bindmountDirs := []string{
		"/var/lib/docker",
		"/var/lib/rktlet",
	}
	for _, d := range bindmountDirs {
		if err := os.MkdirAll(path.Join(upperRootPath, d), 0755); err != nil {
			return err
		}
	}

	args := []string{
		"cni-spawn",
		"--cni-plugin-dir", cniPluginDir,
		"--",
		"--machine", machineName,
		optionsOverlay("--overlay", "/etc", lowerRootPath, upperRootPath),
		optionsOverlay("--overlay", "/opt", lowerRootPath, upperRootPath),
		optionsOverlay("--overlay", "/usr/bin", lowerRootPath, upperRootPath),
	}

	for _, d := range bindmountDirs {
		args = append(args, fmt.Sprintf("--bind=%s:%s", path.Join(upperRootPath, d), d))
	}

	c := utils.Command("kube-spawn", args...)
	c.Stderr = os.Stderr

	// log.Printf("cni-spawn command: %s", strings.Join(c.Args, " "))

	stdout, err := c.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "error creating stdout pipe")
	}
	defer stdout.Close()

	if err := c.Start(); err != nil {
		return errors.Wrap(err, "error running cnispawn")
	}

	cniDataJSON, err := ioutil.ReadAll(stdout)
	if err != nil {
		return errors.Wrap(err, "error reading cni data from stdin")
	}

	if _, err := cniversion.NewResult(cniversion.Current(), cniDataJSON); err != nil {
		log.Printf("unexpected result output: %s", cniDataJSON)
		return errors.Wrap(err, "unable to parse result")
	}

	if err := c.Wait(); err != nil {
		var cniError cnitypes.Error
		if err := json.Unmarshal(cniDataJSON, &cniError); err != nil {
			return errors.Wrap(err, "error unmarshaling cni error")
		}
		return errors.Wrap(&cniError, "error running cnispawn")
	}

	if err := waitMachinesRunning(machineName); err != nil {
		return err
	}
	return nil
}

func waitMachinesRunning(machineName string) error {
	for retries := 0; retries <= 30; retries++ {
		if machinectl.IsRunning(machineName) {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return errors.Errorf("timeout waiting for %q to start", machineName)
}

func optionsOverlay(prefix, targetDir, lower, upper string) string {
	return fmt.Sprintf("%s=+%s:%s:%s:%s", prefix, targetDir, path.Join(lower, targetDir), path.Join(upper, targetDir), targetDir)
}
