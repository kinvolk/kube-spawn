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

// +build integration

package tests

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func runCommand(command string) (string, string, error) {
	log.Printf(command)

	var stdoutBytes, stderrBytes bytes.Buffer
	args := strings.Split(command, " ")
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = &stdoutBytes
	cmd.Stderr = &stderrBytes
	err := cmd.Run()
	return stdoutBytes.String(), stderrBytes.String(), err
}

func runCommandCombinedOutput(command string) (string, error) {
	log.Printf(command)

	args := strings.Split(command, " ")
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func existInSlice(key string, inSlice []string) bool {
	for _, n := range inSlice {
		if n == key {
			return true
		}
	}
	return false
}

func waitForNReadyNodes(expectedNodes int) (map[string]string, error) {
	nodeStates := make(map[string]string, 0)
	timeout := 180 * time.Second
	alarm := time.After(timeout)

	getReadyNodes := func(nodeStates map[string]string) int {
		nReadyNodes := 0

		for _, s := range nodeStates {
			if s != "Ready" {
				continue
			}
			nReadyNodes += 1
		}

		return nReadyNodes
	}

	ticker := time.Tick(500 * time.Millisecond)
loop:
	for {
		select {
		case <-alarm:
			return nodeStates, fmt.Errorf("failed to find %d ready nodes within %v", expectedNodes, timeout)
		case <-ticker:
			stdout, _, err := runCommand(fmt.Sprintf("%s get nodes --no-headers=true", kubeCtlPath))
			if err != nil {
				continue
			}

			outStr := strings.TrimSpace(stdout)
			scanner := bufio.NewScanner(strings.NewReader(outStr))
			nodeStates := make(map[string]string, 0)
			for scanner.Scan() {
				if len(strings.TrimSpace(scanner.Text())) == 0 {
					continue
				}
				name := strings.Fields(scanner.Text())[0]
				state := strings.Fields(scanner.Text())[1]
				nodeStates[name] = state
			}

			if getReadyNodes(nodeStates) != expectedNodes {
				continue
			}

			break loop
		}
	}

	return nodeStates, nil
}

func waitForNDeployments(expectedDeps int) (map[string]string, error) {
	deploys := make(map[string]string, 0)
	timeout := 60 * time.Second
	alarm := time.After(timeout)

	ticker := time.Tick(500 * time.Millisecond)
loop:
	for {
		select {
		case <-alarm:
			return deploys, fmt.Errorf("failed to find %d deployments within %v", expectedDeps, timeout)
		case <-ticker:
			stdout, _, err := runCommand(fmt.Sprintf("%s get deployments --no-headers=true", kubeCtlPath))
			if err != nil {
				continue
			}

			outStr := strings.TrimSpace(stdout)
			scanner := bufio.NewScanner(strings.NewReader(outStr))
			deploys := make(map[string]string, 0)
			for scanner.Scan() {
				if len(strings.TrimSpace(scanner.Text())) == 0 {
					continue
				}
				// example of output:
				//
				// NAME               DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
				// nginx-deployment   2         2         2            2           3m
				depName := strings.Fields(scanner.Text())[0]
				depAvailable := strings.Fields(scanner.Text())[4]
				deploys[depName] = depAvailable
			}

			actualNodes, _ := strconv.Atoi(deploys[deploymentName])
			if actualNodes != expectedDeps {
				continue
			}

			break loop
		}
	}

	return deploys, nil
}
