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
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
)

var (
	cmdStop = &cobra.Command{
		Use:   "stop",
		Short: "Stop nodes by turning off machines",
		Run:   runStop,
	}

	isForce bool
)

func init() {
	cmdKubeSpawn.AddCommand(cmdStop)
	cmdStop.Flags().BoolVarP(&isForce, "force", "f", false, "force the machine to be terminated")
}

func runStop(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(1)
	}

	var machs []string

	nodes, err := bootstrap.GetRunningNodes()
	if err != nil {
		log.Printf("%v", err)
	}
	if len(nodes) > 0 {
		for _, n := range nodes {
			machs = append(machs, n.Name)
		}

		log.Printf("turning off machines %v...\n", machs)
		if err := stopMachines(machs); err != nil {
			log.Printf("%v\n", err)
		}
	}
	log.Printf("All nodes are stopped.")
}

func stopMachines(machs []string) error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("machinectl"); err != nil {
		// fall back to an ordinary abspath to machinectl
		cmdPath = "/usr/bin/machinectl"
	}

	argsOff := []string{
		cmdPath,
	}

	if isForce {
		argsOff = append(argsOff, "terminate")
	} else {
		argsOff = append(argsOff, "poweroff")
	}

	for _, m := range machs {
		args := append(argsOff, m)

		cmd := exec.Cmd{
			Path:   cmdPath,
			Args:   args,
			Env:    os.Environ(),
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		}

		if err := cmd.Run(); err != nil {
			log.Printf("error running %v: %s", args, err)
		}
	}

	return nil
}
