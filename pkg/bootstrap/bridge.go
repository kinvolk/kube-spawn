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
	"syscall"

	"github.com/vishvananda/netlink"
)

const (
	brName string = "cni0"
)

func EnsureBridge() error {
	if _, err := netlink.LinkByName(brName); err == nil {
		return nil
	}

	pid, err := syscall.ForkExec("cni-noop", nil, &syscall.ProcAttr{
		Dir:   "",
		Env:   os.Environ(),
		Files: []uintptr{uintptr(syscall.Stdout), uintptr(syscall.Stderr)},
		Sys:   &syscall.SysProcAttr{},
	})
	if err != nil {
		return err
	}

	var status syscall.WaitStatus
	var rusage syscall.Rusage
	if _, err := syscall.Wait4(pid, &status, 0, &rusage); err != nil {
		return err
	}

	return nil
}
