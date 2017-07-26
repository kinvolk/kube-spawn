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
	"fmt"
	"log"
	"os"
	"path"
)

var (
	homePath string = os.Getenv("HOME")
	goPath   string = os.Getenv("GOPATH")
	cniPath  string = os.Getenv("CNI_PATH")
)

func CheckValidDir(inPath string) error {
	if fi, err := os.Stat(inPath); os.IsNotExist(err) {
		return err
	} else if !fi.IsDir() {
		return fmt.Errorf("%q is not a directory.")
	}
	return nil
}

func GetValidGoPath() (string, error) {
	if err := CheckValidDir(goPath); err != nil {
		// fall back to $HOME/go
		goPathOrig := goPath
		goPath = path.Join(homePath, "go")
		log.Printf("invalid GOPATH %q, fall back to %s...\n", goPathOrig, goPath)
		if err := CheckValidDir(goPath); err != nil {
			return "", err
		}
	}

	return goPath, nil
}

func GetValidCniPath(inGoPath string) (string, error) {
	if err := CheckValidDir(cniPath); err != nil {
		// fall back to $GOPATH/bin
		cniPathOrig := cniPath
		cniPath = path.Join(inGoPath, "bin")
		log.Printf("invalid CNI_PATH %q, fall back to %s...\n", cniPathOrig, cniPath)
		if err := CheckValidDir(cniPath); err != nil {
			return "", err
		}
	}

	return cniPath, nil
}
