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
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	cmdUp = &cobra.Command{
		Use:   "up",
		Short: "Up performs together: pulling raw image, setup and init",
		Run:   runUp,
	}
)

func init() {
	cmdKubeSpawn.AddCommand(cmdUp)
}

func runUp(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(1)
	}

	if err := showImage(); err != nil {
		if err := pullRawImage(); err != nil {
			log.Fatalf("%v\n", err)
		}
	}

	// sudo ./kube-spawn setup --nodes=2 --image=coreos
	doSetup(2, "coreos")

	// sudo ./kube-spawn init
	doInit()

	log.Printf("All nodes are started.")
}

func pullRawImage() error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("machinectl"); err != nil {
		// fall back to an ordinary abspath to machinectl
		cmdPath = "/usr/bin/machinectl"
	}

	args := []string{
		cmdPath,
		"pull-raw",
		"--verify=no",
		"https://alpha.release.core-os.net/amd64-usr/current/coreos_developer_container.bin.bz2",
		"coreos",
	}

	cmd := exec.Cmd{
		Path:   cmdPath,
		Args:   args,
		Env:    os.Environ(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running machinectl pull-raw: %s", err)
	}

	return nil
}

func showImage() error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("machinectl"); err != nil {
		// fall back to an ordinary abspath to machinectl
		cmdPath = "/usr/bin/machinectl"
	}

	args := []string{
		cmdPath,
		"show-image",
		"coreos",
	}

	cmd := exec.Cmd{
		Path:   cmdPath,
		Args:   args,
		Env:    os.Environ(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running machinectl show-image: %s", err)
	}

	return nil
}
