package bootstrap

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

func Exec(stdin io.Reader, stdout, stderr io.Writer, machine string, cmd ...string) error {
	systemdRunPath, err := exec.LookPath("machinectl")
	if err != nil {
		return err
	}
	run := exec.Cmd{
		Path:   systemdRunPath,
		Args:   []string{"machinectl", "shell", machine},
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}
	run.Args = append(run.Args, cmd...)
	return run.Run()
}

func ExecQuiet(machine string, cmd ...string) error {
	var buf bytes.Buffer
	if err := Exec(nil, &buf, &buf, machine, cmd...); err != nil {
		return fmt.Errorf("error running command on '%s': %s", machine, buf.String())
	}
	return nil
}
