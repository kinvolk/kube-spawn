package machinectl

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
)

type Machine struct {
	Name string
	IP   string
}

type Image struct {
	Name string
}

func List() ([]Machine, error) {
	var machines []Machine
	out, err := exec.Command("machinectl", "list", "--no-legend").Output()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		// Example `machinectl list --no-legend` output:
		// kube-spawn-default-worker-fpllng container systemd-nspawn coreos 1478.0.0 10.22.0.130...
		line := strings.Fields(scanner.Text())
		if len(line) < 6 {
			return nil, fmt.Errorf("got unexpected output from `machinectl list --no-legend`: %s", line)
		}
		machine := Machine{
			Name: strings.TrimSpace(line[0]),
			IP:   strings.TrimSuffix(line[5], "..."),
		}
		machines = append(machines, machine)
	}
	return machines, nil
}

func ListByRegexp(expStr string) ([]Machine, error) {
	machines, err := List()
	if err != nil {
		return nil, err
	}
	exp, err := regexp.Compile(expStr)
	if err != nil {
		return nil, err
	}
	var matching []Machine
	for _, machine := range machines {
		if exp.MatchString(machine.Name) {
			matching = append(matching, machine)
		}
	}
	return matching, nil
}

func ListImages() ([]Image, error) {
	var images []Image
	out, err := exec.Command("machinectl", "list-images", "--no-legend").Output()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		// Example `machinectl list-images --no-legend` output:
		// kube-spawn-default-worker-zyyios raw  no  1.4G  n/a     Fri 2018-01-26 10:54:43 CET
		line := strings.Fields(scanner.Text())
		if len(line) < 1 {
			return nil, fmt.Errorf("got unexpected output from `machinectl list-images --no-legend`: %s", line)
		}
		image := Image{
			Name: strings.TrimSpace(line[0]),
		}
		images = append(images, image)
	}
	return images, nil
}

func ListImagesByRegexp(expStr string) ([]Image, error) {
	images, err := ListImages()
	if err != nil {
		return nil, err
	}
	exp, err := regexp.Compile(expStr)
	if err != nil {
		return nil, err
	}
	var matching []Image
	for _, image := range images {
		if exp.MatchString(image.Name) {
			matching = append(matching, image)
		}
	}
	return matching, nil
}

func RunCommand(stdout, stderr io.Writer, opts, cmd, machine string, args ...string) ([]byte, error) {
	mPath, err := exec.LookPath("machinectl")
	if err != nil {
		return nil, err
	}
	cmdArgs := []string{mPath}
	if opts != "" {
		cmdArgs = append(cmdArgs, opts)
	}
	cmdArgs = append(cmdArgs, cmd)
	cmdArgs = append(cmdArgs, machine)

	run := exec.Cmd{
		Path:   mPath,
		Args:   cmdArgs,
		Stdout: stdout,
		Stderr: stderr,
	}
	run.Args = append(run.Args, args...)

	var buf []byte

	if stdout != nil {
		err = run.Run()
	} else {
		buf, err = run.Output()
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%q failed: %s", strings.Join(run.Args, " "), exitErr.Stderr)
		}
		return nil, fmt.Errorf("%q failed: %s", strings.Join(run.Args, " "), err)
	}
	return buf, nil
}

func Exec(machine string, cmd ...string) error {
	_, err := RunCommand(nil, nil, "", "shell", machine, cmd...)
	return err
}

func Clone(base, dest string) error {
	_, err := RunCommand(nil, nil, "", "clone", base, dest)
	return err
}

func Poweroff(machine string) error {
	_, err := RunCommand(nil, nil, "", "poweroff", machine)
	return err
}

func Terminate(machine string) error {
	_, err := RunCommand(nil, nil, "", "terminate", machine)
	return err
}

func Remove(image string) error {
	_, err := RunCommand(nil, nil, "", "remove", image)
	return err
}

func IsRunning(machine string) bool {
	check := exec.Command("systemctl", "--machine", machine, "status", "basic.target", "--state=running")
	check.Run()
	return check.ProcessState.Success()
}

func ImageExists(image string) bool {
	_, err := RunCommand(nil, nil, "", "show-image", image)
	return err == nil
}
