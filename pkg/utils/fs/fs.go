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

package fs

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func CreateFileFromReader(path string, reader io.Reader) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.FileMode(0755)); err != nil {
		return errors.Wrapf(err, "error creating directory %q", dir)
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return errors.Wrapf(err, "error creating %q", path)
	}
	defer f.Close()
	if _, err := io.Copy(f, reader); err != nil {
		return errors.Wrapf(err, "error writing %q", path)
	}
	return nil
}

func CreateFileFromString(path string, content string) error {
	buf := bytes.NewBuffer([]byte(content))
	return CreateFileFromReader(path, buf)
}

func CopyFile(src, dst string) error {
	f, err := os.OpenFile(src, os.O_RDONLY, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	return CreateFileFromReader(dst, f)
}
