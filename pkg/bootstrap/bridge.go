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
	"os"

	"github.com/kinvolk/kube-spawn/pkg/utils"
	"github.com/vishvananda/netlink"
)

const (
	brName string = "cni0"
)

func EnsureBridge() error {
	if _, err := netlink.LinkByName(brName); err == nil {
		return nil
	}

	cmd := utils.Command("cni-noop")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
