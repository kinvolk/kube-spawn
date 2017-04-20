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

package ssh

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	sshConnRetries int = 10
)

func getSession(ipAddress string) (*ssh.Session, error) {
	var conn *ssh.Client
	var err error

	auth, err := getPublicKey()
	if err != nil {
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{auth},
	}

	sshUri := fmt.Sprintf("%s:22", ipAddress)
	for i := 0; i < sshConnRetries; i++ {
		conn, err = ssh.Dial("tcp", sshUri, config)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		return nil, err
	}

	session, err := conn.NewSession()
	if err != nil {
		return nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		session.Close()
		return nil, err
	}

	return session, nil
}

func executeCommand(ipAddress, command string) error {
	session, err := getSession(ipAddress)
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = os.Stdout

	return session.Run(command)
}
