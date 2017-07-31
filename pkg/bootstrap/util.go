package bootstrap

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	tmpDir      string = ".kube-spawn/default"
	FsMagicAUFS        = 0x61756673 // https://goo.gl/CBwx43
	FsMagicZFS         = 0x2FC12FC1 // https://goo.gl/xTvzO5
)

func CreateSharedTmpdir() {
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		os.MkdirAll(tmpDir, os.ModePerm)
		// optional tmpfs
		// if err := syscall.Mount("tmpfs", tmpDir, "tmpfs", syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV, "size=10m"); err != nil {
		// 	return err
		// }
	}
}

// PathSupportsOverlay checks whether the given path is compatible with OverlayFS.
// This method also calls isOverlayfsAvailable().
// It returns error if OverlayFS is not supported.
//  - taken from https://github.com/rkt/rkt/blob/master/common/common.go
func PathSupportsOverlay(path string) error {
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
