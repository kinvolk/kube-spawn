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

	"github.com/kinvolk/kube-spawn/pkg/utils"
)

var (
	runcPath string = os.Getenv("KUBE_SPAWN_RUNC_BINARY_PATH")
	logPath  string = os.Getenv("KUBE_SPAWN_RUNC_LOG_PATH")
)

func main() {
	var newArgs []string

	if logPath != "" {
		fd, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
		if err != nil {
			log.Printf("error opening logs, skipping: %s", err)
		}
		log.SetOutput(fd)
		defer fd.Close()
	}

	log.Printf("old args: %#v", os.Args[1:])

	for _, a := range os.Args[1:] {
		newArgs = append(newArgs, a)
		if a == "create" || a == "run" {
			newArgs = append(newArgs, "--no-new-keyring")
		}
	}

	log.Printf("new args: %#v", newArgs)

	if runcPath == "" {
		var err error
		runcPath, err = exec.LookPath("docker-runc")
		if err != nil {
			// unable to find default
			log.Fatal(err)
		}
	}
	cmd := exec.Command(runcPath, newArgs...)

	// Selectively pass Stdout/Stderr, by determining if they are terminal or not.
	// If we always pass them, interactive mode of connection to containers
	// will fail with error messages like "container not started".
	// If we never pass them, then "kubectl logs" won't be able to print any logs.
	if !utils.IsTerminal(os.Stdout.Fd()) && !utils.IsTerminal(os.Stderr.Fd()) {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
