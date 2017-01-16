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
)

func InitializeMaster(ipAddress string) error {
	log.Println("Initializing master")

	return executeCommand(ipAddress, "./init.sh")
}

func JoinNode(ipAddress, token, masterIpAddress string) error {
	log.Println("Joining node")

	command := fmt.Sprintf("kubeadm reset && KUBE_HYPERKUBE_IMAGE=10.22.0.1:5000/hyperkube-amd64 kubeadm join --discovery token://%s@%s:9898 --config /etc/kubeadm/kubeadm.yml --skip-preflight-checks", token, masterIpAddress)
	return executeCommand(ipAddress, command)
}
