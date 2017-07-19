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
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
)

const (
	containerNameTemplate string = "kube-spawn-%d"
	machinesDir           string = "/var/lib/machines"
	machinesImage         string = "/var/lib/machines.raw"
)

type Node struct {
	Name string
	IP   string
}

func GetNodeName(no int) string {
	return fmt.Sprintf(containerNameTemplate, no)
}

func NewNode(baseImage, machine string) error {
	var buf bytes.Buffer

	machinectlPath, err := exec.LookPath("machinectl")
	if err != nil {
		return err
	}

	clone := exec.Cmd{
		Path:   machinectlPath,
		Args:   []string{"machinectl", "clone", baseImage, machine},
		Stderr: &buf,
	}
	if err := clone.Run(); err != nil {
		return fmt.Errorf("error running machinectl: %s", buf.String())
	}
	return nil
}

func NodeExists(machine string) bool {
	// TODO: we could also parse machinectl list-images to find that
	if _, err := os.Stat(path.Join(machinesDir, machine+".raw")); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		log.Printf("error checking for image: %s", err)
		return false
	}
	return true
}

func GetRunningNodes() ([]Node, error) {
	var nodes []Node

	cmd := exec.Command("machinectl", "list")
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	b, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	raw := strings.Fields(string(b))
	if len(raw) <= 6 {
		return nil, fmt.Errorf("No running machines")
	}
	totaln, err := strconv.Atoi(raw[(len(raw) - 3)])
	if err != nil {
		return nil, err
	}
	for i := 1; i <= totaln; i++ {
		node := Node{
			Name: raw[i*6],
			IP:   strings.TrimSuffix(raw[(i*6)+5], "..."),
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func EnlargeStoragePool(baseImage string, nodes int) error {
	var poolSize int64 // in bytes

	// Give 50% more space for each cloned image.
	// NOTE: this is just a workaround, as how much space we should add more
	// to the image might depend on estimations during run-time operations.
	// In the long run, systemd itself should be able to reserve more space
	// for the storage pool, every time when it pulls an image to store in
	// the pool.
	var extraSizeRatio float64 = 0.5

	baseImageAbspath := path.Join(machinesDir, baseImage+".raw")

	fipool, err := os.Stat(machinesImage)
	if err != nil {
		return err
	}

	poolSize = fipool.Size()

	poolSize += int64(float64(fipool.Size()) * extraSizeRatio)

	fiBase, err := os.Stat(baseImageAbspath)
	if err != nil {
		return err
	}

	poolSize += int64(float64(fiBase.Size())*extraSizeRatio) * int64(nodes)

	// It is equivalent to the following shell commands:
	//  # umount /var/lib/machines
	//  # qemu-img resize -f raw /var/lib/machines.raw <poolsize>
	//  # mount -t btrfs -o loop /var/lib/machines.raw /var/lib/machines
	//  # btrfs filesystem resize max /var/lib/machines
	//  # btrfs quota disable /var/lib/machines
	if err := syscall.Unmount(machinesDir, 0); err != nil {
		// if it's already unmounted, umount(2) returns EINVAL, then continue
		if !os.IsNotExist(err) && err != syscall.EINVAL {
			return err
		}
	}

	if err := runImageResize(poolSize); err != nil {
		// ignore image resize error, continue
		log.Printf("image resize failed: %v\n", err)
	}

	if err := runMount(); err != nil {
		return err
	}

	if err := runBtrfsResize(); err != nil {
		// ignore image resize error, continue
		log.Printf("btrfs resize failed: %v\n", err)
	}

	if err := runBtrfsDisableQuota(); err != nil {
		return err
	}

	return nil
}

func runImageResize(poolSize int64) error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("qemu-img"); err != nil {
		// fall back to an ordinary abspath to qemu-img
		cmdPath = "/usr/bin/qemu-img"
	}

	args := []string{
		cmdPath,
		"resize",
		"-f",
		"raw",
		machinesImage,
		strconv.FormatInt(poolSize, 10),
	}

	cmd := exec.Cmd{
		Path:   cmdPath,
		Args:   args,
		Env:    os.Environ(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running qemu-img: %s", err)
	}

	return nil
}

func runMount() error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("mount"); err != nil {
		// fall back to an ordinary abspath to qemu-img
		cmdPath = "/usr/bin/mount"
	}

	args := []string{
		cmdPath,
		"-t",
		"btrfs",
		"-o",
		"loop",
		machinesImage,
		machinesDir,
	}

	cmd := exec.Cmd{
		Path:   cmdPath,
		Args:   args,
		Env:    os.Environ(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running mount: %s", err)
	}

	return nil
}

func runBtrfsResize() error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("btrfs"); err != nil {
		// fall back to an ordinary abspath to qemu-img
		cmdPath = "/usr/sbin/btrfs"
	}

	args := []string{
		cmdPath,
		"filesystem",
		"resize",
		"max",
		machinesDir,
	}

	cmd := exec.Cmd{
		Path:   cmdPath,
		Args:   args,
		Env:    os.Environ(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running btrfs resize: %s", err)
	}

	return nil
}

func runBtrfsDisableQuota() error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("btrfs"); err != nil {
		// fall back to an ordinary abspath to qemu-img
		cmdPath = "/usr/sbin/btrfs"
	}

	args := []string{
		cmdPath,
		"quota",
		"disable",
		machinesDir,
	}

	cmd := exec.Cmd{
		Path:   cmdPath,
		Args:   args,
		Env:    os.Environ(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running btrfs quota: %s", err)
	}

	return nil
}
