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
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	cnitypes "github.com/containernetworking/cni/pkg/types"
)

func RunContainer(name string) (string, error) {
	args := []string{
		"cnispawn",
		"-path",
		name,
	}

	c := exec.Cmd{
		Path:   "cnispawn",
		Args:   args,
		Env:    os.Environ(),
		Stderr: os.Stderr,
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := c.Start(); err != nil {
		return "", err
	}

	cniDataJson, err := ioutil.ReadAll(bufio.NewReader(stdout))
	if err != nil {
		return "", err
	}

	var cniData cnitypes.Result
	if err := json.Unmarshal(cniDataJson, &cniData); err != nil {
		return "", err
	}

	if err := c.Wait(); err != nil {
		var cniError cnitypes.Error
		if err := json.Unmarshal(cniDataJson, &cniError); err == nil {
			return "", &cniError
		}
		return "", err
	}

	log.Printf("Container %s running\n", name)
	return cniData.IP4.IP.IP.String(), nil
}
