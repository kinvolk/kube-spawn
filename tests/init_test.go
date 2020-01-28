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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kinvolk/kube-spawn/pkg/utils"
)

const (
	k8sStableVersion    string = "v1.7.5"
	defaultKubeSpawnDir string = "/var/lib/kube-spawn"
	deploymentName      string = "nginx-deployment"
)

var (
	numNodes   int = 2
	numDeploys int = 2

	kubeSpawnDir     string = defaultKubeSpawnDir
	kubeSpawnK8sPath string = filepath.Join(kubeSpawnDir, "k8s")
	kubeSpawnPath    string
	kubeCtlPath      string

	machineCtlPath string
	goPath         string
	cniPath        string
)

type Node struct {
	Name string
	IP   string
}

type Service struct {
	IP   string
	Port string
}

func checkRequirements(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Fatal("smoke test requires root privileges")
	}
}

func initPath(t *testing.T) {
	var err error

	// go one dir upper, from "tests" to the top source directory
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}

	kubeSpawnPath = "./kube-spawn"
	if err := utils.CheckValidFile(kubeSpawnPath); err != nil {
		if kubeSpawnPath, err = exec.LookPath("kube-spawn"); err != nil {
			// fall back to an ordinary abspath to kube-spawn
			kubeSpawnPath = "/usr/bin/kube-spawn"
		}
	}

	_ = os.MkdirAll(kubeSpawnK8sPath, os.FileMode(0755))

	kubeCtlPath = filepath.Join(kubeSpawnK8sPath, "kubectl")
	if err := utils.CheckValidFile(kubeCtlPath); err != nil {
		if kubeCtlPath, err = exec.LookPath(kubeCtlPath); err != nil {
			// fall back to an ordinary abspath to kubectl
			kubeCtlPath = "/usr/bin/kubectl"
		}
	}

	machineCtlPath, err = exec.LookPath("machinectl")
	if err != nil {
		// fall back to an ordinary abspath to machinectl
		machineCtlPath = "/usr/bin/machinectl"
	}

	goPath = os.Getenv("GOPATH")
	if goPath == "" {
		t.Fatalf("GOPATH was not set")
	}
	cniPath = filepath.Join(goPath, "bin")
	os.Setenv("CNI_PATH", cniPath)
}

func initNode(t *testing.T) {
	// If no coreos image exists, just download it
	if _, _, err := runCommand(fmt.Sprintf("%s show-image coreos", machineCtlPath)); err != nil {
		if stdout, stderr, err := runCommand(fmt.Sprintf("%s pull-raw --verify=no %s %s",
			machineCtlPath,
			"https://alpha.release.core-os.net/amd64-usr/current/coreos_developer_container.bin.bz2",
			"coreos",
		)); err != nil {
			t.Fatalf("error running machinectl pull-raw: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
		}
	}
}

func getMachines(profileName string) ([]string, error) {
	var machNames []string

	files, err := ioutil.ReadDir(filepath.Join(kubeSpawnDir, profileName))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "kubespawn") {
			continue
		}

		machNames = append(machNames, file.Name())
	}

	return machNames, nil
}

func getListImages() ([]string, error) {
	var imageNames []string

	stdout, stderr, err := runCommand(fmt.Sprintf("%s list-images --no-legend", machineCtlPath))
	if err != nil {
		return nil, fmt.Errorf("error running machinectl list-images: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	s := bufio.NewScanner(strings.NewReader(strings.TrimSpace(stdout)))
	for s.Scan() {
		line := strings.Fields(s.Text())
		if len(line) <= 2 {
			continue
		}

		// an example line:
		//   kubespawn0 raw  no  1.4G  Wed 2017-10-25 02:15:19 CEST Wed 2017-10-25 02:15:19 CEST
		nodeName := strings.TrimSpace(line[0])
		if !strings.HasPrefix(nodeName, "kubespawn") {
			continue
		}

		imageNames = append(imageNames, nodeName)
	}

	return imageNames, nil
}

func getRunningNodes() ([]Node, error) {
	var nodes []Node

	stdout, stderr, err := runCommand(fmt.Sprintf("%s list --no-legend", machineCtlPath))
	if err != nil {
		return nil, fmt.Errorf("error running machinectl list: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	s := bufio.NewScanner(strings.NewReader(strings.TrimSpace(stdout)))
	for s.Scan() {
		line := strings.Fields(s.Text())
		if len(line) <= 2 {
			continue
		}

		// an example line from systemd v232 or newer:
		//  kubespawn0 container systemd-nspawn coreos 1478.0.0 10.22.0.130...
		//
		// systemd v231 or older:
		//  kubespawn0 container systemd-nspawn

		var ipaddr string
		machineName := strings.TrimSpace(line[0])
		if !strings.HasPrefix(machineName, "kubespawn") {
			continue
		}

		if len(line) >= 6 {
			ipaddr = strings.TrimSuffix(line[5], "...")
		} else {
			ipaddr, err = getIPAddressLegacy(machineName)
			if err != nil {
				return nil, err
			}
		}
		node := Node{
			Name: machineName,
			IP:   ipaddr,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func getIPAddressLegacy(mach string) (string, error) {
	// machinectl status kubespawn0 --no-pager | grep Address
	args := []string{
		"status",
		mach,
		"--no-pager",
	}

	cmd := exec.Command("machinectl", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	b, err := cmd.Output()
	if err != nil {
		return "", err
	}

	s := bufio.NewScanner(strings.NewReader(string(b)))
	for s.Scan() {
		// an example line is like this:
		//
		//  Address: 10.22.0.4
		if strings.Contains(s.Text(), "Address:") {
			line := strings.TrimSpace(s.Text())
			fields := strings.Fields(line)
			if len(fields) <= 1 {
				continue
			}
			return fields[1], nil
		}
	}

	return "", err
}

func checkK8sNodes(t *testing.T) {
	nodeStates, err := waitForNReadyNodes(numNodes)
	if err != nil {
		t.Fatalf("error waiting on %d ready nodes, result %v: %v\n", numNodes, nodeStates, err)
	}
}

func testCreateNodes(t *testing.T) {
	if stdout, stderr, err := runCommand(fmt.Sprintf("%s --kubernetes-version=%s create --nodes=%d",
		kubeSpawnPath, k8sStableVersion, numNodes),
	); err != nil {
		t.Fatalf("error running kube-spawn create: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	machs, err := getMachines("default")
	if err != nil {
		t.Fatalf("error getting list of machines: %v\n", err)
	}
	if len(machs) != numNodes {
		t.Fatalf("got %d machines, expected %d machines.\n", len(machs), numNodes)
	}
}

func testStartNodes(t *testing.T) {
	if out, err := runCommandCombinedOutput(fmt.Sprintf("%s start", kubeSpawnPath)); err != nil {
		t.Fatalf("error running kube-spawn start: %v\nstdout: %s\nstderr: %s", err, out)
	}

	// set env variable KUBECONFIG to /var/lib/kube-spawn/default/kubeconfig
	if err := os.Setenv("KUBECONFIG", utils.GetValidKubeConfig()); err != nil {
		t.Fatalf("error running setenv: %v\n", err)
	}

	images, err := getListImages()
	if err != nil {
		t.Fatalf("error getting list of images: %v\n", err)
	}
	if len(images) != numNodes {
		t.Fatalf("got %d images, expected %d images.\n", len(images), numNodes)
	}

	nodes, err := getRunningNodes()
	if err != nil {
		t.Fatalf("error getting list of nodes: %v\n", err)
	}
	if len(nodes) != numNodes {
		t.Fatalf("got %d nodes, expected %d nodes.\n", len(nodes), numNodes)
	}

	checkK8sNodes(t)
}

func testApplyDeploy(t *testing.T) {
	if stdout, stderr, err := runCommand(fmt.Sprintf("%s apply -f %s",
		kubeCtlPath, "./tests/fixtures/nginx-deployment.yaml"),
	); err != nil {
		t.Fatalf("error creating deployment: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	deploys, err := waitForNDeployments(numDeploys)
	if err != nil {
		t.Fatalf("error waiting on %d deployments, result %v: %v\n", numDeploys, deploys, err)
	}
}

func testExposeDeploy(t *testing.T) {
	if stdout, stderr, err := runCommand(fmt.Sprintf("%s expose deployment/nginx-deployment",
		kubeCtlPath)); err != nil {
		t.Fatalf("error exposing deployment: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
}

func testServices(t *testing.T) {
	stdout, stderr, err := runCommand(fmt.Sprintf("%s get services --no-headers=true", kubeCtlPath))
	if err != nil {
		t.Fatalf("error getting services: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	outStr := strings.TrimSpace(string(stdout))
	scanner := bufio.NewScanner(strings.NewReader(outStr))
	svcs := make(map[string]Service, 0)
	for scanner.Scan() {
		if len(strings.TrimSpace(scanner.Text())) == 0 {
			continue
		}

		name := strings.TrimSpace(strings.Fields(scanner.Text())[0])
		clusterIP := strings.TrimSpace(strings.Fields(scanner.Text())[1])
		portSet := strings.Fields(scanner.Text())[3]
		port := strings.Split(portSet, "/")[0]

		svcs[name] = Service{
			IP:   clusterIP,
			Port: port,
		}
	}

	if len(svcs) == 0 {
		t.Fatalf("cannot find any services\n")
	}

	svc, ok := svcs["nginx-deployment"]
	if !ok {
		t.Fatalf("cannot find service nginx-deployment\n")
	}

	testCheckConnectivity(t, svc)
}

func testCheckConnectivity(t *testing.T, svc Service) {
	nodes, err := getRunningNodes()
	if err != nil {
		t.Fatalf("error getting running nodes: %v\n", err)
	}

	// the cluster IP:Port should be reachable from any node
	stdout, stderr, err := runCommand(fmt.Sprintf("machinectl shell %s /usr/bin/curl %s:%s", nodes[0].Name, svc.IP, svc.Port))
	if err != nil {
		t.Fatalf("error checking for connectivity: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
}

func TestMainK8sStable(t *testing.T) {
	checkRequirements(t)
	initPath(t)
	initNode(t)

	testCreateNodes(t)
	testStartNodes(t)
	testApplyDeploy(t)
	testExposeDeploy(t)
	testServices(t)
}
