//
// Copyright 2018 Kinvolk GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package utils

import (
	"crypto/sha1"
	"encoding/base64"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
)

func VerifySha1(binFilePath, checksumPath string) error {
	outB, err := ioutil.ReadFile(binFilePath)
	if err != nil {
		return errors.Wrapf(err, "error reading file %s", binFilePath)
	}

	outC, err := ioutil.ReadFile(checksumPath)
	if err != nil {
		return errors.Wrapf(err, "error reading file %s", checksumPath)
	}

	hasher := sha1.New()
	if _, err := hasher.Write(outB); err != nil {
		return errors.Wrapf(err, "error reading for hash from %s", binFilePath)
	}

	hashSha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	if strings.TrimSpace(hashSha) != strings.TrimSpace(string(outC)) {
		return errors.Wrapf(err, "error verifying checksum for file %s", binFilePath)
	}

	return nil
}
