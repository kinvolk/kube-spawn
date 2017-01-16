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
	"encoding/csv"
	"fmt"
	"os"
	"path"
)

func GetToken(containerName string) (string, error) {
	tokenFilePath := path.Join(containerName, "etc", "kubernetes", "pki", "tokens.csv")
	f, err := os.Open(tokenFilePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	lines, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return "", err
	}

	for _, line := range lines {
		for _, row := range line {
			return row, nil
		}
	}

	return "", fmt.Errorf("No token generated")
}
