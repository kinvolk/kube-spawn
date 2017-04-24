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

package nspawntool

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/mholt/archiver"

	"github.com/kinvolk/kubeadm-nspawn/pkg/bootstrap"
)

const (
	osImage               string = "rootfs.tar.xz"
	containerNameTemplate string = "kubeadm-nspawn-%d"
)

var (
	centosImageUrl string   = os.Getenv("IMAGE_URL") // "https://uk.images.linuxcontainers.org/images/centos/7/amd64/default/20170403_02:16/rootfs.tar.xz"
	packageList    []string = []string{
		"bind-utils",
		"ebtables",
		"ethtool",
		"docker-1.10.3",
		"dnf",
		"findutils",
		"iproute",
		"jq",
		"less",
		"net-tools",
		"openssh-server",
		"procps",
		"socat",
		"strace",
		"tmux",
		"util-linux",
		"vim",
		"wget",
	}
)

var (
	ErrInvalidImageMethod error = errors.New("Invalid image method")
	ErrNoURL              error = errors.New("No image url provided")
)

func GetNodeName(no int) string {
	return fmt.Sprintf(containerNameTemplate, no)
}

func CreateImage(method string) error {
	if _, err := os.Stat(osImage); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		log.Println("OS image exists")
		return nil
	}

	switch method {
	case "mkosi":
		return createImageMkosi()
	case "download":
		if err := createImageDL(); err != nil {
			return err
		}
		return bootstrapCentosImage()
	default:
		return ErrInvalidImageMethod
	}
}

func createImageMkosi() error {
	log.Println("Image method: mkosi")

	args := []string{
		"--distribution", "fedora",
		"--release", "24",
		"--format", "tar",
		"--output", path.Join(osImage),
		"--cache", "mkosi.cache",
		"--extra-tree", path.Join("scripts", "node"),
		"--postinst-script", path.Join("scripts", "bootstrap.sh"),
	}
	for _, p := range packageList {
		args = append(args, "-p="+p)
	}
	cmd := exec.Command("mkosi", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("Creating OS image")
	return cmd.Run()
}

func createImageDL() error {
	log.Println("Image method: download")
	if centosImageUrl == "" {
		return ErrNoURL
	}

	imageFile, err := os.Create(osImage)
	if err != nil {
		return err
	}
	defer imageFile.Close()

	response, err := http.Get(centosImageUrl)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected repsonse status: %v", response.Status)
	}

	if _, err := io.Copy(imageFile, response.Body); err != nil {
		return err
	}
	log.Println("Downloaded file")

	return nil
}

func bootstrapCentosImage() error {
	log.Println("Bootstrapping CentOS image")

	if err := ExtractImage("tmp"); err != nil {
		return err
	}

	args := []string{
		"-D", "tmp",
		"yum", "-y", "install", "https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm", "&&",
		"yum", "-y", "install", strings.Join(packageList, " "),
	}
	installPkgs := exec.Command("systemd-nspawn", args...)
	installPkgs.Stdout = os.Stdout
	installPkgs.Stderr = os.Stderr

	log.Println("Installing packages")
	if err := installPkgs.Run(); err != nil {
		return err
	}

	log.Println("Installing scripts")
	bootstrap.InstallFile("tmp", path.Join("scripts", "bootstrap.sh"), path.Join("root", "bootstrap.sh"))
	bootstrap.InstallFile("tmp", path.Join("scripts", "node", "root", "init.sh"), path.Join("root", "init.sh"))
	bootstrap.InstallFile("tmp", path.Join("scripts", "node", "root", "weave-daemonset.yaml"), path.Join("root", "weave-daemonset.yaml"))

	log.Println("Running bootstrap script")
	script := exec.Command("systemd-nspawn", "-D", "tmp", "/root/bootstrap.sh")
	script.Stdout = os.Stdout
	script.Stderr = os.Stderr
	if err := script.Run(); err != nil {
		return err
	}

	log.Println("Removing old rootfs")
	// remove unprovisioned centos image
	if err := os.Remove(osImage); err != nil {
		return err
	}

	log.Println("Compressing final image")
	if err := archiver.TarXZ.Make(osImage, []string{"./tmp/"}); err != nil {
		return err
	}

	// remove after rootfs.tar.xz creation
	return os.RemoveAll("tmp")
}

func ExtractImage(dest string) error {
	log.Println("Unpacking rootfs to", dest)
	return archiver.TarXZ.Open(osImage, dest)
}
