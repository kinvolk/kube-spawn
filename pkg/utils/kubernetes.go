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

package utils

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	kubeSpawnDir string = "/var/lib/kube-spawn"
	ksHiddenDir  string = ".kube-spawn"
	kcRelPath    string = "default/kubeconfig"
	ksRelPath    string = "src/github.com/kinvolk/kube-spawn"
)

var (
	homePath string = os.Getenv("HOME")
	goPath   string = os.Getenv("GOPATH")
	cniPath  string = os.Getenv("CNI_PATH")

	kcUserPath   string = filepath.Join(ksHiddenDir, kcRelPath)
	kcSystemPath string = filepath.Join(kubeSpawnDir, kcRelPath)
)

func CheckValidDir(inPath string) error {
	if fi, err := os.Stat(inPath); os.IsNotExist(err) {
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("%q is not a directory.", inPath)
	}
	return nil
}

func CheckValidFile(inPath string) error {
	if fi, err := os.Stat(inPath); os.IsNotExist(err) {
		return err
	} else if !fi.Mode().IsRegular() {
		return fmt.Errorf("%q is not a file.", inPath)
	}
	return nil
}

func GetValidGoPath() (string, error) {
	if err := CheckValidDir(goPath); err != nil {
		// fall back to $HOME/go
		goPath = path.Join(homePath, "go")
		if err := CheckValidDir(goPath); err != nil {
			return "", err
		}
	}

	return goPath, nil
}

func GetValidKubeConfig() string {
	kcPath := kcSystemPath
	if err := CheckValidFile(kcPath); err != nil {
		// fall back to $GOPATH/src/github.com/kinvolk/kube-spawn/.kube-spawn/default/kubeconfig
		kcPath = filepath.Join(goPath, ksRelPath, kcUserPath)
		log.Printf("fall back to %s...\n", kcPath)

		if err := CheckValidFile(kcPath); err != nil {
			// fall back to $HOME/go/src/github.com/kinvolk/kube-spawn/.kube-spawn/default/kubeconfig
			kcPath = filepath.Join(homePath, "go", ksRelPath, kcUserPath)
			log.Printf("fall back to %s...\n", kcPath)
			if err := CheckValidFile(kcPath); err != nil {
				return ""
			}
		}
	}

	return kcPath
}

func GetK8sBuildOutputDir() (string, error) {
	goPath, err := GetValidGoPath()
	if err != nil {
		return "", err
	}
	k8sRepoPath := filepath.Join(goPath, "/src/k8s.io/kubernetes")
	// first try to use "_output/dockerized/bin/linux/amd64"
	outputPath := filepath.Join(k8sRepoPath, "_output/dockerized/bin/linux/amd64")
	if err := CheckValidDir(outputPath); err != nil {
		// fall back to "_output/bin"
		outputPath = filepath.Join(k8sRepoPath, "_output/bin")
		if err := CheckValidDir(outputPath); err != nil {
			return "", err
		}
	}

	return outputPath, nil
}

func GetK8sBuildAssetDir() (string, error) {
	goPath, err := GetValidGoPath()
	if err != nil {
		return "", err
	}
	k8sAssetPath := filepath.Join(goPath, "/src/k8s.io/kubernetes/build")
	if err := CheckValidDir(k8sAssetPath); err != nil {
		return "", err
	}
	return k8sAssetPath, nil
}

// IsTerminal returns true if the given file descriptor is a terminal.
func IsTerminal(fd uintptr) bool {
	var termios syscall.Termios
	_, _, err := unix.Syscall(unix.SYS_IOCTL, fd, uintptr(syscall.TCGETS), uintptr(unsafe.Pointer(&termios)))
	return err == 0
}
