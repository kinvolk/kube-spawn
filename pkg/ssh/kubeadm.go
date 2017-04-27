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

package ssh

import (
	"fmt"
	"log"
	"strings"
)

func InitializeMaster(ipAddress string) (string, error) {
	log.Println("Initializing master")

	token, err := generateToken(ipAddress)
	if err != nil {
		return "", err
	}

	// return executeCommand(ipAddress, "./init.sh")
	command := fmt.Sprintf("kubeadm reset && KUBE_HYPERKUBE_IMAGE=10.22.0.1:5000/hyperkube-amd64 kubeadm init --skip-preflight-checks --config /etc/kubeadm/kubeadm.yml && kubeadm token create '%s' --description 'systemd-nspawn bootstrap token' --ttl 0", token)

	if err := executeCommand(ipAddress, command); err != nil {
		return "", err
	}

	if err := executeCommand(ipAddress, "kubectl apply -f https://git.io/weave-kube-1.6"); err != nil {
		return "", err
	}

	return token, nil
}

func generateToken(ipAddress string) (string, error) {
	session, err := getSession(ipAddress)
	if err != nil {
		return "", err
	}
	defer session.Close()

	out, err := session.Output("kubeadm token generate")
	if err != nil {
		return "", err
	}

	// remove trailing '\n\r'
	token := strings.TrimSpace(string(out))

	log.Printf("Generated token: %s\n", token)
	return token, nil
}

func JoinNode(ipAddress, masterIpAddress string, token string) error {
	log.Println("Joining node")

	command := fmt.Sprintf("kubeadm reset && KUBE_HYPERKUBE_IMAGE=10.22.0.1:5000/hyperkube-amd64 kubeadm join --skip-preflight-checks --token '%s' %s:6443", token, masterIpAddress)
	// TODO(robertgzr): copy /etc/kubernetes/kubelet.conf to ~/.kube/config to be able to use kubectl on workers
	return executeCommand(ipAddress, command)
}
