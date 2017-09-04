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

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/coreos/ioprogress"
	"github.com/spf13/cobra"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"golang.org/x/crypto/openpgp/packet"
)

const (
	imageUrl     = "https://alpha.release.core-os.net/amd64-usr/current/coreos_developer_container.bin.bz2"
	signatureUrl = "https://alpha.release.core-os.net/amd64-usr/current/coreos_developer_container.bin.bz2.sig"
)

var (
	cmdUp = &cobra.Command{
		Use:   "up",
		Short: "Up performs together: pulling raw image, setup and init",
		Run:   runUp,
	}

	upNumNodes     int
	upBaseImage    string
	upKubeSpawnDir string
)

func init() {
	cmdKubeSpawn.AddCommand(cmdUp)

	cmdUp.Flags().IntVarP(&upNumNodes, "nodes", "n", 1, "number of nodes to spawn")
	cmdUp.Flags().StringVarP(&upBaseImage, "image", "i", "coreos", "base image for nodes")
	cmdUp.Flags().StringVarP(&upKubeSpawnDir, "kube-spawn-dir", "d", "", "path to directory where .kube-spawn directory is located")
}

func runUp(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(1)
	}

	bootstrap.PrepareCoreosImage()
	if err := showImage(); err != nil {
		if err := importRawImage(); err != nil {
			log.Fatalf("%v\n", err)
		}
	}

	// e.g: sudo ./kube-spawn setup --nodes=2 --image=coreos
	doSetup(upNumNodes, upBaseImage, upKubeSpawnDir)

	// sudo ./kube-spawn init
	doInit()

	log.Printf("All nodes are started.")
}

func downloadFile(dest, url string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	progress := &ioprogress.Reader{
		Reader: response.Body,
		Size:   response.ContentLength,
	}

	if _, err := io.Copy(f, progress); err != nil {
		return err
	}

	if err := f.Sync(); err != nil {
		return err
	}

	return nil
}

func downloadImage() error {
	return downloadFile("/tmp/coreos_developer_container.bin.bz2", imageUrl)
}

func downloadSignature() error {
	return downloadFile("/tmp/coreos_developer_container.bin.bz2.sig", signatureUrl)
}

func verifyImage() error {
	// TODO(nhlfr): Try to use or implement some library for managing PGP keys instead
	// of executing gpg binary.
	gpgCmdPath, err := exec.LookPath("gpg")
	if err != nil {
		gpgCmdPath = "/usr/bin/gpg"
	}

	importPubKeyArgs := []string{
		gpgCmdPath,
		"--keyserver",
		"keyserver.ubuntu.com",
		"--recv-key",
		"50E0885593D2DCB4",
	}

	importPubKeyCmd := exec.Cmd{
		Path:   gpgCmdPath,
		Args:   importPubKeyArgs,
		Env:    os.Environ(),
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	log.Printf("kopytko")

	if err := importPubKeyCmd.Run(); err != nil {
		return err
	}

	exportPubKeyArgs := []string{
		gpgCmdPath,
		"--export",
		"50E0885593D2DCB4",
		"--export-options",
		"export-minimal,no-export-attributes",
	}

	exportPubKeyCmd := exec.Cmd{
		Path:   gpgCmdPath,
		Args:   exportPubKeyArgs,
		Env:    os.Environ(),
		Stderr: os.Stderr,
	}

	exportStdout, err := exportPubKeyCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %s", err)
	}
	defer exportStdout.Close()

	if err := exportPubKeyCmd.Start(); err != nil {
		return fmt.Errorf("error running gpg: %s", err)
	}

	pubKeyArmor, err := ioutil.ReadAll(exportStdout)
	if err != nil {
		return fmt.Errorf("error reading public key from stdout: %s", err)
	}

	if err := exportPubKeyCmd.Wait(); err != nil {
		return fmt.Errorf("error running gpg: %s", err)
	}

	hexdumpCmdPath, err := exec.LookPath("hexdump")
	if err != nil {
		hexdumpCmdPath = "/usr/bin/hexdump"
	}

	hexdumpArgs := []string{
		hexdumpCmdPath,
		"/dev/stdin",
		"-v",
		"-e",
		"/1 \"%02X\"",
	}

	hexdumpCmd := exec.Cmd{
		Path:   hexdumpCmdPath,
		Args:   hexdumpArgs,
		Env:    os.Environ(),
		Stderr: os.Stderr,
	}

	hexdumpStdin, err := hexdumpCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("error creating stdin pipe: %s", err)
	}
	// defer hexdumpStdin.Close()

	hexdumpStdout, err := hexdumpCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %s", err)
	}
	defer hexdumpStdout.Close()

	if _, err := hexdumpStdin.Write(pubKeyArmor); err != nil {
		return fmt.Errorf("error writing public key to stdin: %s", err)
	}

	if err := hexdumpCmd.Start(); err != nil {
		return fmt.Errorf("error running hexdump: %s", err)
	}

	hexdumpStdin.Close()

	pubKeyHex, err := ioutil.ReadAll(hexdumpStdout)
	if err != nil {
		return fmt.Errorf("error reading public key hex from stdout: %s", err)
	}

	if err := hexdumpCmd.Wait(); err != nil {
		return fmt.Errorf("error running hexdump: %s", err)
	}

	imageContent, err := ioutil.ReadFile("/tmp/coreos_developer_container.bin.bz2")
	if err != nil {
		return err
	}

	signatureFile, err := os.Open("/tmp/coreos_developer_container.bin.bz2.sig")
	if err != nil {
		return err
	}
	defer signatureFile.Close()

	signatureContent, err := packet.Read(signatureFile)
	if err != nil {
		return err
	}

	signature, ok := signatureContent.(*packet.Signature)
	if !ok {
		return fmt.Errorf("invalid signature file")
	}

	hash := signature.Hash.New()

	pubKeyBin, err := hex.DecodeString(string(pubKeyHex[:]))
	if err != nil {
		return err
	}

	pubKeyContent, err := packet.Read(bytes.NewReader(pubKeyBin))
	if err != nil {
		return err
	}

	pubKey, ok := pubKeyContent.(*packet.PublicKey)
	if !ok {
		return fmt.Errorf("invalid public key")
	}

	if _, err := hash.Write(imageContent); err != nil {
		return err
	}

	if err := pubKey.VerifySignature(hash, signature); err != nil {
		return fmt.Errorf("signature verification failed: %s", err)
	}

	return nil
}

func importRawImage() error {
	var cmdPath string
	var err error

	if err := downloadImage(); err != nil {
		return err
	}
	if err := downloadSignature(); err != nil {
		return err
	}
	if err := verifyImage(); err != nil {
		return err
	}
	log.Printf("Image downloaded and verified successfully")

	if cmdPath, err = exec.LookPath("machinectl"); err != nil {
		// fall back to an ordinary abspath to machinectl
		cmdPath = "/usr/bin/machinectl"
	}

	args := []string{
		cmdPath,
		"pull-raw",
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

func showImage() error {
	var cmdPath string
	var err error

	if cmdPath, err = exec.LookPath("machinectl"); err != nil {
		// fall back to an ordinary abspath to machinectl
		cmdPath = "/usr/bin/machinectl"
	}

	args := []string{
		cmdPath,
		"show-image",
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
		return fmt.Errorf("error running machinectl show-image: %s", err)
	}

	return nil
}
