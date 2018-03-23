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

package bootstrap

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	FsMagicAUFS     = 0x61756673 // https://goo.gl/CBwx43
	FsMagicECRYPTFS = 0xF15F     // https://goo.gl/4akUXJ
	FsMagicZFS      = 0x2FC12FC1 // https://goo.gl/xTvzO5
)

// PathSupportsOverlay checks whether the given path is compatible with OverlayFS.
// This method also calls isOverlayfsAvailable().
// It returns error if OverlayFS is not supported.
//  - taken from https://github.com/rkt/rkt/blob/master/common/common.go
func PathSupportsOverlay(path string) error {
	ensureOverlayfs()
	if !isOverlayfsAvailable() {
		return fmt.Errorf("overlayfs is not available")
	}

	var data syscall.Statfs_t
	if err := syscall.Statfs(path, &data); err != nil {
		return fmt.Errorf("cannot statfs %q", path)
	}

	switch data.Type {
	case FsMagicAUFS:
		return fmt.Errorf("unsupported filesystem: aufs")
	case FsMagicECRYPTFS:
		return fmt.Errorf("unsupported filesystem: ecryptfs")
	case FsMagicZFS:
		return fmt.Errorf("unsupported filesystem: zfs")
	}

	dir, err := os.OpenFile(path, syscall.O_RDONLY|syscall.O_DIRECTORY, 0755)
	if err != nil {
		return fmt.Errorf("cannot open %q", path)
	}
	defer dir.Close()

	buf := make([]byte, 4096)
	// ReadDirent forwards to the raw syscall getdents(3),
	// passing the buffer size.
	n, err := syscall.ReadDirent(int(dir.Fd()), buf)
	if err != nil {
		return fmt.Errorf("cannot read directory %q", path)
	}

	offset := 0
	for offset < n {
		// offset overflow cannot happen, because Reclen
		// is being maintained by getdents(3), considering the buffer size.
		dirent := (*syscall.Dirent)(unsafe.Pointer(&buf[offset]))
		offset += int(dirent.Reclen)

		if dirent.Ino == 0 { // File absent in directory.
			continue
		}

		if dirent.Type == syscall.DT_UNKNOWN {
			return fmt.Errorf("unsupported filesystem: missing d_type support")
		}
	}

	return nil
}

func isSameFilesystem(a, b *syscall.Statfs_t) bool {
	return a.Fsid == b.Fsid
}

func checkMountpoint(dir string) error {
	sfs1 := &syscall.Statfs_t{}
	if err := syscall.Statfs(dir, sfs1); err != nil {
		return fmt.Errorf("error calling statfs on %q: %v", dir, err)
	}
	sfs2 := &syscall.Statfs_t{}
	if err := syscall.Statfs(filepath.Dir(dir), sfs2); err != nil {
		return fmt.Errorf("error calling statfs on %q: %v", dir, err)
	}
	if isSameFilesystem(sfs1, sfs2) {
		return fmt.Errorf("%q is not a mount point", dir)
	}

	return nil
}

// get free space of volume mounted on volPath (in bytes)
func getVolFreeSpace(volPath string) (uint64, error) {
	var stat syscall.Statfs_t

	if err := syscall.Statfs(volPath, &stat); err != nil {
		log.Printf("statfs error: %v\n", err)
		return 0, err
	}

	freeSpace := stat.Bavail * uint64(stat.Bsize)

	return freeSpace, nil
}

// get allocated size of file (in bytes)
func getAllocatedFileSize(filename string) (int64, error) {
	fi, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}

	stat_t, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("cannot determine allocated filesize")
	}

	// stat(2) returns allocated filesize in blocks, each of which is
	// a fixed 512 bytes
	return (stat_t.Blocks * 512), nil
}
