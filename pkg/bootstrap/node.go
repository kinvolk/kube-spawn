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
	containerNameTemplate  string = "kubespawn%d"
	ctHashsizeModparam     string = "/sys/module/nf_conntrack/parameters/hashsize"
	ctHashsizeValue        string = "131072"
	ctMaxSysctl            string = "/proc/sys/net/nf_conntrack_max"
	machinesDir            string = "/var/lib/machines"
	machinesImage          string = "/var/lib/machines.raw"
	baseImageStableVersion string = "1478.0.0"
)

var (
	BaseImageName string = "flatcar"
	baseImageURL  string = "https://alpha.release.flatcar-linux.net/amd64-usr/current/flatcar_developer_container.bin.bz2"
)

func GetPoolSize(baseImageName string, nodes int) (int64, error) {
	var poolSize, extraSize, biSize int64 // in bytes

	// Give 50% more space for each cloned image.
	// NOTE: this is just a workaround, as how much space we should add more
	// to the image might depend on estimations during run-time operations.
	// In the long run, systemd itself should be able to reserve more space
	// for the storage pool, every time when it pulls an image to store in
	// the pool.
	var extraSizeRatio float64 = 0.5
	var err error

	baseImageAbspath := path.Join(machinesDir, baseImageName+".raw")

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

func setPoolLimit(poolSize int64) error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("machinectl"); err != nil {
		return fmt.Errorf("machinectl not installed: %s", err)
	}

	args := []string{
		cmdPath,
		"set-limit",
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
		// We do expect this might occur. E.g. if they have
		// already started using the pool
		return fmt.Errorf("error running machinectl: %s", err)
	}

	return nil
}

func EnlargeStoragePool(poolSize int64) error {
	// Check to see if the filesystem is already >= poolSize
	var stat syscall.Statfs_t
	if err := syscall.Statfs(machinesDir, &stat); err == nil {
		if stat.Bsize*int64(stat.Blocks) >= poolSize {
			return nil
		}
	}

	// First call `machinectl set-limit size`. If this succeeds,
	// we are done
	if err := setPoolLimit(poolSize); err == nil {
		return nil
	}
	// If this fails, it means that the FS has been used.
	// We attempt to work around this with the following shell commands:
	//  # umount /var/lib/machines
	//  # qemu-img resize -f raw /var/lib/machines.raw <poolsize>
	//  # mount -t btrfs -o loop /var/lib/machines.raw /var/lib/machines
	//  # btrfs filesystem resize max /var/lib/machines
	//  # btrfs quota disable /var/lib/machines
	// Note: this fails if anything is in use
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
		return fmt.Errorf("qemu-img not installed: %s", err)
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
		return fmt.Errorf("Cannot find mount command: %s", err)
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
		return fmt.Errorf("btrfs not installed: %s", err)
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
		return fmt.Errorf("btrfs not installed: %s", err)
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
		return errors.Wrap(err, "error writing CNI configuration")
	}
	// Ensure that the system requirements are satisfied for starting
	// kube-spawn. It's just like running the commands below:
	//
	// modprobe overlay
	// modprobe nf_conntrack
	// echo "131072" > /sys/module/nf_conntrack/parameters/hashsize
	if err := ensureOverlayfs(); err != nil {
		return errors.Wrap(err, "error ensuring overlayfs loaded")
	}
	if err := ensureConntrackHashsize(); err != nil {
		return err
	}

	// insert an iptables rules to allow traffic through cni0
	if err := ensureIptables(); err != nil {
		return err
	}
	// check for SELinux enforcing mode
	if err := ensureSelinux(); err != nil {
		return err
	}
	// check for BaseImage version, either Container Linux or Flatcar Linux
	return ensureBaseImageVersion()
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
		return fmt.Errorf("Cannot find modprobe command: %s", err)
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

func ensureOverlayfs() error {
	if !isOverlayfsAvailable() {

		// This is an incorrect assumption. It depends on the docker
		// storage driver
		log.Println("Warning: overlayfs not found, docker would not run.")
		log.Println("loading overlay module... ")

		if err := runModprobe("overlay"); err != nil {
			return fmt.Errorf("error running modprobe overlay: %v\n", err)
		}
	}
	return nil
}

func isConntrackLoaded() bool {
	if _, err := os.Stat(ctHashsizeModparam); os.IsNotExist(err) {
		return false
	}

	return true
}

func isConntrackHashsizeCorrect() (bool, error) {
	hsByte, err := ioutil.ReadFile(ctHashsizeModparam)
	if err != nil {
		return false, fmt.Errorf("cannot read from %s: %v\n", ctHashsizeModparam, err)
	}
	hsStr := strings.TrimSpace(string(hsByte))
	hs, err := strconv.Atoi(hsStr)
	if err != nil {
		return false, fmt.Errorf("parse error on %s: %v\n", hsStr, err)
	}

	ctmaxByte, err := ioutil.ReadFile(ctMaxSysctl)
	if err != nil {
		return false, fmt.Errorf("cannot open %s: %v\n", ctMaxSysctl, err)
	}
	ctmaxStr := strings.TrimSpace(string(ctmaxByte))
	ctmax, err := strconv.Atoi(ctmaxStr)
	if err != nil {
		return false, fmt.Errorf("parse error on %s: %v\n", ctmaxStr, err)
	}

	if hs < (ctmax / 4) {
		return false, fmt.Errorf("hashsize(%d) should be greater than nf_conntrack_max/4 (%d).\n", hs, ctmax/4)
	}

	return true, nil
}

func setConntrackHashsize() error {
	if err := ioutil.WriteFile(ctHashsizeModparam, []byte(ctHashsizeValue), os.FileMode(0600)); err != nil {
		return err
	}

	return nil
}

func ensureConntrackHashsize() error {
	if result := isConntrackLoaded(); !result {
		log.Println("Warning: nf_conntrack module is not loaded.")
		log.Println("loading nf_conntrack module... ")

		if err := runModprobe("nf_conntrack"); err != nil {
			return fmt.Errorf("error running modprobe nf_conntrack: %v\n", err)
		}
	}

	if _, err := isConntrackHashsizeCorrect(); err != nil {
		return err
	}

	log.Println("Warning: kube-proxy could crash due to insufficient nf_conntrack hashsize.")
	log.Printf("setting nf_conntrack hashsize to %s... ", ctHashsizeValue)

	if err := setConntrackHashsize(); err != nil {
		return fmt.Errorf("error setting conntrack hashsize: %v\n", err)
	}
	return nil
}

func setIptablesForwardPolicy() error {
	var cmdPath string
	var err error

	log.Println("making iptables FORWARD chain defaults to ACCEPT...")

	if cmdPath, err = exec.LookPath("iptables"); err != nil {
		return fmt.Errorf("Cannot find iptables: %s", err)
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

func isCniRuleLoaded() (bool, error) {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("iptables"); err != nil {
		return false, fmt.Errorf("Cannot find iptables: %s", err)
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
		return false, err
	}

	return true, nil
}

func setAllowCniRule() error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("iptables"); err != nil {
		return fmt.Errorf("Cannot find iptables: %s", err)
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

func ensureIptables() error {
	if err := setIptablesForwardPolicy(); err != nil {
		return fmt.Errorf("error running iptables: %v\n", err)
	}

	if result, _ := isCniRuleLoaded(); !result {
		log.Println("setting iptables rule to allow CNI traffic...")
		if err := setAllowCniRule(); err != nil {
			return fmt.Errorf("error running iptables: %v\n", err)
		}
	}
	return nil
}

func isSELinuxEnforcing() bool {
	var cmdGetPath string
	var err error

	if cmdGetPath, err = exec.LookPath("getenforce"); err != nil {
		// If we do not have getenforce, we do not have selinux
		// (e.g. Ubuntu host). So by definition it cannot
		// be enforcing
		return false
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

func ensureSelinux() error {
	if isSELinuxEnforcing() {
		log.Fatalln("ERROR: SELinux enforcing mode is enabled. You will need to disable it with 'sudo setenforce 0' for kube-spawn to work properly.")
	}
	return nil
}

func checkBaseImageSemver(baseImageVer string) error {
	v, err := semver.NewVersion(baseImageVer)
	if err != nil {
		return err
	}

	c, err := semver.NewConstraint(">=" + baseImageStableVersion)
	if err != nil {
		log.Printf("cannot get constraint for >= %s: %v", baseImageStableVersion, err)
		return err
	}

	if c.Check(v) {
		return nil
	} else {
		return fmt.Errorf("ERROR: Container Linux version %s is too low in your local image.", baseImageVer)
	}
}

func checkBaseImageVersion() error {
	args := []string{
		"image-status",
		BaseImageName,
	}

	cmd := exec.Command("machinectl", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	b, err := cmd.Output()
	if err != nil {
		return err
	}

	checkBaseImageVersionField := func(values []string) error {
		for _, v := range values {
			if err := checkBaseImageSemver(strings.TrimSpace(v)); err != nil {
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
		if err := checkBaseImageVersionField(values); err != nil {
			return err
		}
	}

	return nil
}

func pullBaseImage() error {
	var cmdPath string
	var err error

	// TODO: use machinectl pkg
	if cmdPath, err = exec.LookPath("machinectl"); err != nil {
		return fmt.Errorf("systemd-nspawn / machinectl not installed: %s", err)
	}

	args := []string{
		cmdPath,
		"pull-raw",
		"--verify=no",
		baseImageURL,
		BaseImageName,
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

func ensureBaseImageVersion() error {
	if err := checkBaseImageVersion(); err != nil {
		log.Println(err)
		log.Fatalf("You will need to remove the image by 'sudo machinectl remove %s' then the next run of kube-spawn will download version %s of coreos image automatically.", BaseImageName, baseImageStableVersion)
	}
	return nil
}

func PrepareBaseImage() error {
	// If no image exists, just download it
	if !machinectl.ImageExists(BaseImageName) {
		log.Printf("pulling %s image...", BaseImageName)
		if err := pullBaseImage(); err != nil {
			return err
		}
	} else {
		// If BaseImageName is not new enough, remove the existing image,
		// then next time `kube-spawn up` will download a new image again.
		if err := ensureBaseImageVersion(); err != nil {
			return err
		}
	}
	return nil
}
