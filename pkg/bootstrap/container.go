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

package bootstrap

import (
	"log"
	"os"

	"github.com/kinvolk/kubeadm-nspawn/pkg/ssh"
)

func NodeRootfsExists(name string) (bool, error) {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	log.Printf("Rootfs for node '%s' exists\n", name)
	return true, nil
}

func BootstrapNode(name string) error {
	log.Println("Bootstrapping", name)

	if err := installBinaries(name); err != nil {
		return err
	}
	if err := installCniBinaries(name); err != nil {
		return err
	}
	if err := installSystemdUnits(name); err != nil {
		return err
	}
	if err := installKubeletConfig(name); err != nil {
		return err
	}

	if err := ssh.PrepareAuthorizedKeys(name); err != nil {
		return err
	}

	log.Println("Bootstrapped node")
	return nil
}
