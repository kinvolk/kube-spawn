package machinetool

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const machineDir = "/var/lib/machines"

func machinectl(stdout, stderr io.Writer, cmd, machine string, args ...string) ([]byte, error) {
	mPath, err := exec.LookPath("machinectl")
	if err != nil {
		return nil, err
	}
	run := exec.Cmd{
		Path:   mPath,
		Args:   []string{"machinectl", cmd, machine},
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

func Shell(machine string, cmd ...string) error {
	_, err := machinectl(os.Stdout, os.Stderr, "shell", machine, cmd...)
	return err
}

func Output(cmd, machine string, args ...string) ([]byte, error) {
	return machinectl(nil, nil, cmd, machine, args...)
}

func Exec(machine string, cmd ...string) error {
	_, err := machinectl(nil, nil, "shell", machine, cmd...)
	return err
}

func Clone(base, dest string) error {
	_, err := machinectl(nil, nil, "clone", base, dest)
	return err
}

func Poweroff(machine string) error {
	_, err := machinectl(nil, nil, "poweroff", machine)
	return err
}

func Terminate(machine string) error {
	_, err := machinectl(nil, nil, "terminate", machine)
	return err
}

func RemoveImage(image string) error {
	_, err := machinectl(nil, nil, "remove", image)
	return err
}

func IsRunning(machine string) bool {
	check := exec.Command("systemctl", "--machine", machine, "status", "basic.target", "--state=running")
	check.Run()
	return check.ProcessState.Success()
}

func IsNotKnown(err error) bool {
	re := regexp.MustCompile(`(.*)No (machine|image) '(.*)' known`)
	return re.MatchString(err.Error())
}
