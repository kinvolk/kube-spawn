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
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"regexp"
)

var (
	gopath            string = os.Getenv("GOPATH")
	binariesDir       string = path.Join(gopath, "src", "k8s.io", "kubernetes", "_output", "bin")
	binariesDest      string = path.Join("usr", "bin")
	cniDir            string = path.Join(gopath, "src", "github.com", "containernetworking", "cni", "bin")
	cniDest           string = path.Join("opt", "cni", "bin")
	kubeletConfigPath string = path.Join(gopath, "src", "k8s.io", "release", "rpm", "10-kubeadm.conf")
	kubeletConfigDest string = path.Join("etc", "systemd", "system", "kubelet.service.d", "10-kubeadm.conf")
	systemdUnitDir    string = path.Join(gopath, "src", "k8s.io", "kubernetes", "build", "debs")
	systemdUnitDest   string = path.Join("usr", "lib", "systemd", "system")
)

func installFile(containerName, filePath, dest string) error {
	installPath, err := exec.LookPath("install")
	if err != nil {
		return err
	}

	dest = path.Join(containerName, dest)

	args := []string{
		installPath,
		"-D",
		filePath,
		dest,
	}

	c := exec.Cmd{
		Path:   installPath,
		Args:   args,
		Dir:    "",
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := c.Run(); err != nil {
		return err
	}

	return nil
}

func installBinary(containerName, binaryPath string) error {
	fullPath := path.Join(binariesDir, binaryPath)
	return installFile(containerName, fullPath, binariesDest)
}

func installSystemdUnit(containerName, systemdUnitPath string) error {
	fullPath := path.Join(systemdUnitDir, systemdUnitPath)
	return installFile(containerName, fullPath, systemdUnitDest)
}

func installBinaries(containerName string) error {
	binaries, err := ioutil.ReadDir(binariesDir)
	if err != nil {
		return err
	}

	for _, binary := range binaries {
		if err := installBinary(containerName, binary.Name()); err != nil {
			return err
		}
	}

	return nil
}

func installCniBinary(containerName, binaryPath string) error {
	fullPath := path.Join(cniDir, binaryPath)
	fullDest := path.Join(cniDest, binaryPath)
	return installFile(containerName, fullPath, fullDest)
}

func installCniBinaries(containerName string) error {
	binaries, err := ioutil.ReadDir(cniDir)
	if err != nil {
		return err
	}

	for _, binary := range binaries {
		if err := installCniBinary(containerName, binary.Name()); err != nil {
			return err
		}
	}

	return nil
}

func installKubeletConfig(containerName string) error {
	return installFile(containerName, kubeletConfigPath, kubeletConfigDest)
}

func installSystemdUnits(containerName string) error {
	distFiles, err := ioutil.ReadDir(systemdUnitDir)
	if err != nil {
		return err
	}

	r, err := regexp.Compile(".service$")
	if err != nil {
		return err
	}

	for _, distFile := range distFiles {
		if !r.MatchString(distFile.Name()) {
			continue
		}

		if err := installSystemdUnit(containerName, distFile.Name()); err != nil {
			return err
		}
	}

	return nil
}
