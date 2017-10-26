package bootstrap

import (
	"path"
	"sync"

	"github.com/kinvolk/kube-spawn/pkg/config"
	"github.com/kinvolk/kube-spawn/pkg/utils/fs"
	"github.com/pkg/errors"
)

func CopyFiles(cfg *config.ClusterConfiguration) error {
	var err error
	var wg sync.WaitGroup
	wg.Add(len(cfg.Copymap))
	for _, pm := range cfg.Copymap {
		go func(dst, src string) {
			defer wg.Done()
			// dst is relative to the machine rootfs
			dst = path.Join(cfg.KubeSpawnDir, cfg.Name, "rootfs", dst)
			// log.Println(src, "->", dst)
			if !fs.Exists(src) {
				err = errors.Errorf("'%s' not found", src)
			}
			if !fs.Exists(dst) {
				if copyErr := fs.Copy(src, dst); copyErr != nil {
					err = copyErr
				}
			}
		}(pm.Dst, pm.Src)
	}
	wg.Wait()
	return err
}
