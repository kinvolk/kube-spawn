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
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
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
	ctHashsizeModparam    string = "/sys/module/nf_conntrack/parameters/hashsize"
	ctHashsizeValue       string = "131072"
	ctMaxSysctl           string = "/proc/sys/net/nf_conntrack_max"
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

func MachineImageExists(machine string) bool {
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

	args := []string{
		"list",
		"--no-legend",
	}

	cmd := exec.Command("machinectl", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	b, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(strings.NewReader(string(b)))
	for s.Scan() {
		line := strings.Fields(s.Text())
		if len(line) <= 2 {
			continue
		}

		// an example line from systemd v232 or newer:
		//  kube-spawn-0 container systemd-nspawn coreos 1478.0.0 10.22.0.130...
		//
		// systemd v231 or older:
		//  kube-spawn-0 container systemd-nspawn

		var ipaddr string
		machineName := line[0]
		if len(line) >= 6 {
			ipaddr = strings.TrimSuffix(line[5], "...")
		} else {
			ipaddr, err = GetIPAddressLegacy(machineName)
			if err != nil {
				return nil, err
			}
		}
		node := Node{
			Name: machineName,
			IP:   ipaddr,
		}
		nodes = append(nodes, node)
	}

	return nodes, nil
}

func GetIPAddressLegacy(mach string) (string, error) {
	// machinectl status kube-spawn-0 --no-pager | grep Address
	args := []string{
		"status",
		mach,
		"--no-pager",
	}

	cmd := exec.Command("machinectl", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	b, err := cmd.Output()
	if err != nil {
		return "", err
	}

	s := bufio.NewScanner(strings.NewReader(string(b)))
	for s.Scan() {
		// an example line is like this:
		//
		//  Address: 10.22.0.4
		if strings.Contains(s.Text(), "Address:") {
			line := strings.TrimSpace(s.Text())
			fields := strings.Fields(line)
			if len(fields) <= 1 {
				continue
			}
			return fields[1], nil
		}
	}

	return "", err
}

func IsNodeRunning(nodeName string) bool {
	var err error
	var runNodes []Node
	if runNodes, err = GetRunningNodes(); err != nil {
		return false
	}
	for _, n := range runNodes {
		if strings.TrimSpace(nodeName) == strings.TrimSpace(n.Name) {
			return true
		}
	}
	return false
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

func EnsureRequirements() {
	// Ensure that the system requirements are satisfied for starting
	// kube-spawn. It's just like running the commands below:
	//
	// modprobe overlay
	// modprobe nf_conntrack
	// echo "131072" > /sys/module/nf_conntrack/parameters/hashsize
	ensureOverlayfs()
	ensureConntrackHashsize()

	// TODO: handle SELinux as well as firewalld
}

func isOverlayfsAvailable() bool {
	f, err := os.Open("/proc/filesystems")
	if err != nil {
		log.Fatalf("cannot open /proc/filesystems: %v", err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	for s.Scan() {
		if s.Text() == "nodev\toverlay" {
			return true
		}
	}
	return false
}

func runModprobe(moduleName string) error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("modprobe"); err != nil {
		// fall back to an ordinary abspath
		cmdPath = "/usr/sbin/modprobe"
	}

	args := []string{
		cmdPath,
		moduleName,
	}

	cmd := exec.Cmd{
		Path:   cmdPath,
		Args:   args,
		Env:    os.Environ(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func ensureOverlayfs() {
	if isOverlayfsAvailable() {
		return
	}

	log.Println("Warning: overlayfs not found, docker would not run.")
	log.Println("loading overlay module... ")

	if err := runModprobe("overlay"); err != nil {
		log.Printf("error running modprobe overlay: %v\n", err)
		return
	}
}

func isConntrackLoaded() bool {
	if _, err := os.Stat(ctHashsizeModparam); os.IsNotExist(err) {
		log.Printf("nf_conntrack module is not loaded: %v\n", err)
		return false
	}

	return true
}

func isConntrackHashsizeCorrect() bool {
	hsStr, err := ioutil.ReadFile(ctHashsizeModparam)
	if err != nil {
		log.Printf("cannot read from %s: %v\n", ctHashsizeModparam, err)
		return false
	}
	hs, _ := strconv.Atoi(string(hsStr))

	ctmaxStr, err := ioutil.ReadFile(ctMaxSysctl)
	if err != nil {
		log.Printf("cannot open %s: %v\n", ctMaxSysctl, err)
		return false
	}
	ctmax, _ := strconv.Atoi(string(ctmaxStr))

	if hs < (ctmax / 4) {
		log.Printf("hashsize(%d) should be greater than nf_conntrack_max/4 (%d).\n", hs, ctmax/4)
		return false
	}

	return true
}

func setConntrackHashsize() error {
	if err := ioutil.WriteFile(ctHashsizeModparam, []byte(ctHashsizeValue), os.FileMode(0600)); err != nil {
		return err
	}

	return nil
}

func ensureConntrackHashsize() {
	if !isConntrackLoaded() {
		log.Println("Warning: nf_conntrack module is not loaded.")
		log.Println("loading nf_conntrack module... ")

		if err := runModprobe("nf_conntrack"); err != nil {
			log.Printf("error running modprobe nf_conntrack: %v\n", err)
			return
		}
	}

	if isConntrackHashsizeCorrect() {
		return
	}

	log.Println("Warning: kube-proxy could crash due to insufficient nf_conntrack hashsize.")
	log.Printf("setting nf_conntrack hashsize to %s... ", ctHashsizeValue)

	if err := setConntrackHashsize(); err != nil {
		log.Printf("error setting conntrack hashsize: %v\n", err)
		return
	}
}
