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
	"os"
	"os/exec"
	"path/filepath"
)

func IsExecBinary(path string) bool {
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if fi.IsDir() {
		return false
	}
	return (fi.Mode().Perm() & 0111) == 0
}

// Command creates an exec.Cmd instance like exec.Command does - it
// fills only Path and Args fields. But the difference is that the
// binary in current working directory takes precedence over the
// binary in PATH.
func Command(name string, arg ...string) *exec.Cmd {
	cmd := &exec.Cmd{
		Path: name,
		Args: append([]string{name}, arg...),
	}
	if filepath.Base(name) == name {
		if IsExecBinary(name) {
			return cmd
		}
		if lp, err := exec.LookPath(name); err == nil {
			cmd.Path = lp
		}
	}
	return cmd
}
