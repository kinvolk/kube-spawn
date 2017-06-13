package bootstrap

import "os"

const tmpDir string = "tmp"

func CreateSharedTmpdir() {
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		os.MkdirAll(tmpDir, os.ModePerm)
		// optional tmpfs
		// if err := syscall.Mount("tmpfs", tmpDir, "tmpfs", syscall.MS_NOEXEC|syscall.MS_NOSUID|syscall.MS_NODEV, "size=10m"); err != nil {
		// 	return err
		// }
	}
}
