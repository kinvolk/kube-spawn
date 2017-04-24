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
	"os"
	"os/exec"
	"syscall"
)

func Cleanup(nodes int) error {
	for i := 0; i < nodes; i++ {
		name := GetNodeName(i)
		if err := StopNode(name); err != nil {
			return err
		}
	}

	cmd := exec.Command("systemctl", "reset-failed")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func StopNode(name string) error {
	nodes, err := RunningNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		if node.Name == name {
			proc, err := os.FindProcess(node.PID)
			if err != nil {
				return err
			}
			if err := proc.Signal(syscall.SIGTERM); err != nil {
				return err
			}
		}
	}

	return nil
}
