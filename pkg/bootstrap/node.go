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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"

	"github.com/Masterminds/semver"
	"github.com/kinvolk/kube-spawn/pkg/machinectl"
	"github.com/pkg/errors"
)

const (
	containerNameTemplate string = "kubespawn%d"
	ctHashsizeModparam    string = "/sys/module/nf_conntrack/parameters/hashsize"
	ctHashsizeValue       string = "131072"
	ctMaxSysctl           string = "/proc/sys/net/nf_conntrack_max"
	machinesDir           string = "/var/lib/machines"
	machinesImage         string = "/var/lib/machines.raw"
	coreosStableVersion   string = "1478.0.0"
)

func GetPoolSize(baseImage string, nodes int) (int64, error) {
	var poolSize, extraSize, biSize int64 // in bytes

	// Give 50% more space for each cloned image.
	// NOTE: this is just a workaround, as how much space we should add more
	// to the image might depend on estimations during run-time operations.
	// In the long run, systemd itself should be able to reserve more space
	// for the storage pool, every time when it pulls an image to store in
	// the pool.
	var extraSizeRatio float64 = 0.5
	var err error

	baseImageAbspath := path.Join(machinesDir, baseImage+".raw")

	if poolSize, err = getAllocatedFileSize(machinesImage); err != nil {
		return 0, err
	}
	extraSize = int64(float64(poolSize) * extraSizeRatio)

	if biSize, err = getAllocatedFileSize(baseImageAbspath); err != nil {
		return 0, err
	}
	extraSize += int64(float64(biSize)*extraSizeRatio) * int64(nodes)

	varDir, _ := path.Split(machinesImage)
	freeVolSpace, err := getVolFreeSpace(varDir)
	if err != nil {
		return 0, err
	}

	// extraSize, space to be allocated, shoud be 90% of freeVolSpace,
	// actual free space on the target volume. Here 90% is simply a
	// pre-defined estimation of how much space can be occupied.
	// We should reserve some unallocated free space for the whole host,
	// as 100% usage of rootfs could badly affect the system's reliability.
	if extraSize >= int64((float64(freeVolSpace))*0.9) {
		biSizeMB := int64(biSize / 1024 / 1024)
		return 0, fmt.Errorf("not enough space on disk for %d nodes, Each node needs about %d MB, so in total you'll need about %d MB available.", nodes, biSizeMB, int64(nodes)*biSizeMB)
	}

	poolSize += extraSize

	return poolSize, nil
}

func EnlargeStoragePool(poolSize int64) error {
	// It is equivalent to the following shell commands:
	//  # umount /var/lib/machines
	//  # qemu-img resize -f raw /var/lib/machines.raw <poolsize>
	//  # mount -t btrfs -o loop /var/lib/machines.raw /var/lib/machines
	//  # btrfs filesystem resize max /var/lib/machines
	//  # btrfs quota disable /var/lib/machines
	if err := checkMountpoint(machinesDir); err == nil {
		// It means machinesDir is mountpoint, so do unmount
		if err := syscall.Unmount(machinesDir, 0); err != nil {
			// if it's already unmounted, umount(2) returns EINVAL, then continue
			if !os.IsNotExist(err) && err != syscall.EINVAL {
				return err
			}
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

func EnsureRequirements() error {
	// TODO: should be moved to pkg/config/defaults.go
	if err := WriteNetConf(); err != nil {
		errors.Wrap(err, "error writing CNI configuration")
	}
	// Ensure that the system requirements are satisfied for starting
	// kube-spawn. It's just like running the commands below:
	//
	// modprobe overlay
	// modprobe nf_conntrack
	// echo "131072" > /sys/module/nf_conntrack/parameters/hashsize
	ensureOverlayfs()
	ensureConntrackHashsize()

	// insert an iptables rules to allow traffic through cni0
	ensureIptables()
	// check for SELinux enforcing mode
	ensureSelinux()
	// check for Container Linux version
	// TODO: this hardcodes usage of coreos
	ensureCoreosVersion()
	return nil
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
	hsByte, err := ioutil.ReadFile(ctHashsizeModparam)
	if err != nil {
		log.Printf("cannot read from %s: %v\n", ctHashsizeModparam, err)
		return false
	}
	hsStr := strings.TrimSpace(string(hsByte))
	hs, err := strconv.Atoi(hsStr)
	if err != nil {
		log.Printf("parse error on %s: %v\n", hsStr, err)
		return false
	}

	ctmaxByte, err := ioutil.ReadFile(ctMaxSysctl)
	if err != nil {
		log.Printf("cannot open %s: %v\n", ctMaxSysctl, err)
		return false
	}
	ctmaxStr := strings.TrimSpace(string(ctmaxByte))
	ctmax, err := strconv.Atoi(ctmaxStr)
	if err != nil {
		log.Printf("parse error on %s: %v\n", ctmaxStr, err)
		return false
	}

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

func setIptablesForwardPolicy() error {
	var cmdPath string
	var err error

	log.Println("making iptables FORWARD chain defaults to ACCEPT...")

	if cmdPath, err = exec.LookPath("iptables"); err != nil {
		// fall back to an ordinary abspath
		cmdPath = "/sbin/iptables"
	}

	// set the default policy for FORWARD chain to ACCEPT
	// : iptables -P FORWARD ACCEPT
	args := []string{
		cmdPath,
		"-P",
		"FORWARD",
		"ACCEPT",
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

func isCniRuleLoaded() bool {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("iptables"); err != nil {
		// fall back to an ordinary abspath
		cmdPath = "/sbin/iptables"
	}

	// check if a cni iptables rules already exists
	// : iptables -C FORWARD -i cni0 -j ACCEPT
	args := []string{
		cmdPath,
		"-C",
		"FORWARD",
		"-i",
		"cni0",
		"-j",
		"ACCEPT",
	}

	cmd := exec.Cmd{
		Path:   cmdPath,
		Args:   args,
		Env:    os.Environ(),
		Stdout: os.Stdout,
	}

	if err := cmd.Run(); err != nil {
		// error means that the rule does not exist
		return false
	}

	return true
}

func setAllowCniRule() error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("iptables"); err != nil {
		// fall back to an ordinary abspath
		cmdPath = "/sbin/iptables"
	}

	// insert an iptables rules to allow traffic through cni0
	// : iptables -I FORWARD 1 -i cni0 -j ACCEPT
	args := []string{
		cmdPath,
		"-I",
		"FORWARD",
		"1",
		"-i",
		"cni0",
		"-j",
		"ACCEPT",
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

func ensureIptables() {
	setIptablesForwardPolicy()

	if !isCniRuleLoaded() {
		log.Println("setting iptables rule to allow CNI traffic...")
		if err := setAllowCniRule(); err != nil {
			log.Printf("error running iptables: %v\n", err)
			return
		}
	}
}

func isSELinuxEnforcing() bool {
	var cmdGetPath string
	var err error

	if cmdGetPath, err = exec.LookPath("getenforce"); err != nil {
		// fall back to an ordinary abspath
		cmdGetPath = "/usr/sbin/getenforce"
	}

	argsGet := []string{
		cmdGetPath,
	}

	cmdGet := exec.Cmd{
		Path:   cmdGetPath,
		Args:   argsGet,
		Env:    os.Environ(),
		Stderr: os.Stderr,
	}

	// As getenforce always returns non-error, we should ignore the error.
	// Instead, parse the output string directly to determine the current
	// SELinux mode.
	outstr, _ := cmdGet.Output()
	sestatus := strings.TrimSpace(string(outstr))
	if sestatus == "Enforcing" {
		return true
	}

	return false
}

func ensureSelinux() {
	if isSELinuxEnforcing() {
		log.Fatalln("ERROR: SELinux enforcing mode is enabled. You will need to disable it with 'sudo setenforce 0' for kube-spawn to work properly.")
	}
}

func checkCoreosSemver(coreosVer string) error {
	v, err := semver.NewVersion(coreosVer)
	if err != nil {
		return err
	}

	c, err := semver.NewConstraint(">=" + coreosStableVersion)
	if err != nil {
		log.Printf("cannot get constraint for >= %s: %v", coreosStableVersion, err)
		return err
	}

	if c.Check(v) {
		return nil
	} else {
		return fmt.Errorf("ERROR: Container Linux version %s is too low in your local image.", coreosVer)
	}
}

func checkCoreosVersion() error {
	args := []string{
		"image-status",
		"coreos",
	}

	cmd := exec.Command("machinectl", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	b, err := cmd.Output()
	if err != nil {
		return err
	}

	checkCoreosVersionField := func(values []string) error {
		for _, v := range values {
			if err := checkCoreosSemver(strings.TrimSpace(v)); err != nil {
				if err == semver.ErrInvalidSemVer {
					// just meaning it's not a version field, so continue to the next field
					continue
				} else {
					return err
				}
			} else {
				return nil
			}
		}
		return fmt.Errorf("cannot find a version field")
	}

	s := bufio.NewScanner(strings.NewReader(string(b)))
	for s.Scan() {
		// an example line from machinectl image-status:
		//  OS: Container Linux by CoreOS 1478.0.0 (Ladybug)

		line := strings.Split(s.Text(), ":")
		if len(line) <= 1 {
			continue
		}

		keyStr := strings.TrimSpace(line[0])
		valueStr := strings.TrimSpace(line[1])
		if keyStr != "OS" {
			continue
		}

		// now the line has the key "OS", so get the version field in the values
		values := strings.Fields(valueStr)
		if err := checkCoreosVersionField(values); err != nil {
			return err
		}
	}

	return nil
}

func pullRawCoreosImage() error {
	var cmdPath string
	var err error

	// TODO: use machinectl pkg
	if cmdPath, err = exec.LookPath("machinectl"); err != nil {
		// fall back to an ordinary abspath to machinectl
		cmdPath = "/usr/bin/machinectl"
	}

	args := []string{
		cmdPath,
		"pull-raw",
		"--verify=no",
		"https://alpha.release.core-os.net/amd64-usr/current/coreos_developer_container.bin.bz2",
		"coreos",
	}

	cmd := exec.Cmd{
		Path:   cmdPath,
		Args:   args,
		Env:    os.Environ(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running machinectl pull-raw: %s", err)
	}

	return nil
}

func ensureCoreosVersion() {
	if err := checkCoreosVersion(); err != nil {
		log.Println(err)
		log.Fatalf("You will need to remove the image by 'sudo machinectl remove coreos' then the next run of kube-spawn will download version %s of coreos image automatically.", coreosStableVersion)
	}
}

func PrepareCoreosImage() error {
	// If no coreos image exists, just download it
	if !machinectl.ImageExists("coreos") {
		log.Printf("pulling coreos image...")
		if err := pullRawCoreosImage(); err != nil {
			return err
		}
	} else {
		// If coreos image is not new enough, remove the existing image,
		// then next time `kube-spawn up` will download a new image again.
		ensureCoreosVersion()
	}
	return nil
}
