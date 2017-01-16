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
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"

	"github.com/kinvolk/kubeadm-systemd/pkg/ssh"
)

var (
	scripts []string = []string{"bootstrap.sh", "init.sh", "weave-daemonset.yaml"}
)

func copyScript(src, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return nil
}

func prepareScript(containerName, scriptName string) error {
	src := path.Join("scripts", scriptName)
	dest := path.Join(containerName, "root", scriptName)

	if err := copyScript(src, dest); err != nil {
		return err
	}

	return os.Chmod(dest, 0755)
}

func prepareScripts(containerName string) error {
	for _, script := range scripts {
		if err := prepareScript(containerName, script); err != nil {
			return err
		}
	}
	return nil
}

func ContainerRootfsExists(name string) (bool, error) {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func BootstrapContainer(name string) error {
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

	if err := prepareScripts(name); err != nil {
		return err
	}

	systemdNspawn, err := exec.LookPath("systemd-nspawn")
	if err != nil {
		return err
	}

	nspawnBootstrapArgs := []string{systemdNspawn, "-D", name, "/root/bootstrap.sh"}

	pid, err := syscall.ForkExec(systemdNspawn, nspawnBootstrapArgs, &syscall.ProcAttr{
		Dir:   "",
		Env:   os.Environ(),
		Files: []uintptr{uintptr(syscall.Stdin), uintptr(syscall.Stdout), uintptr(syscall.Stderr)},
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

	if err := ssh.PrepareAuthorizedKeys(name); err != nil {
		return err
	}

	log.Println("Bootstraped container")

	return nil
}
