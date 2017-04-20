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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"

	"golang.org/x/crypto/ssh"
)

const (
	privKeyPath string = "id_rsa"
	pubKeyPath  string = "id_rsa.pub"
)

func generateSSHKeys() error {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	privKeyFile, err := os.Create(privKeyPath)
	if err != nil {
		return err
	}
	defer privKeyFile.Close()

	privKeyPem := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)}
	if err := pem.Encode(privKeyFile, privKeyPem); err != nil {
		return err
	}

	pubKey, err := ssh.NewPublicKey(&privKey.PublicKey)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(pubKeyPath, ssh.MarshalAuthorizedKey(pubKey), 0600); err != nil {
		return err
	}

	log.Println("Generated SSH keypair")

	return nil
}

func chmodSSHKeys() error {
	return os.Chmod(privKeyPath, 0600)
}

func keysExist() (bool, error) {
	if _, err := os.Stat(privKeyPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func PrepareSSHKeys() error {
	exist, err := keysExist()
	if err != nil {
		return err
	}
	if exist {
		return nil
	}

	if err := generateSSHKeys(); err != nil {
		return err
	}
	return chmodSSHKeys()
}

func getSSHPath(name string) string {
	return path.Join(name, "root", ".ssh")
}

func createAuthorizedKeys(name string) error {
	sshPath := getSSHPath(name)
	if err := os.Mkdir(sshPath, 0700); err != nil {
		return err
	}

	authorizedKeysPath := path.Join(sshPath, "authorized_keys")
	out, err := os.Create(authorizedKeysPath)
	if err != nil {
		return err
	}
	defer out.Close()

	in, err := os.Open(pubKeyPath)
	if err != nil {
		return err
	}
	defer in.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return nil
}

func chmodAuthorizedKeys(name string) error {
	authorizedKeysPath := path.Join(getSSHPath(name), "authorized_keys")
	return os.Chmod(authorizedKeysPath, 0600)
}

func PrepareAuthorizedKeys(name string) error {
	if err := createAuthorizedKeys(name); err != nil {
		return err
	}
	return chmodAuthorizedKeys(name)
}

func getPublicKey() (ssh.AuthMethod, error) {
	buf, err := ioutil.ReadFile(privKeyPath)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(key), nil
}
