package bootstrap

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
)

const (
	k8sURL       string = "https://dl.k8s.io/v$VERSION/bin/linux/amd64/"
	k8sGithubURL string = "https://raw.githubusercontent.com/kubernetes/release/master/rpm/"
)

var (
	k8sfiles = []string{
		k8sURL + "kubelet",
		k8sURL + "kubeadm",
		k8sURL + "kubectl",
		k8sGithubURL + "kubelet.service",
		k8sGithubURL + "10-kubeadm.conf",
	}
)

func Download(url, fpath string) (*os.File, error) {
	fd, err := os.OpenFile(fpath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error downloading %s: %s", url, err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(fd, resp.Body); err != nil {
		return nil, err
	}
	return fd, nil
}

func DownloadK8sBins(version, dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, os.ModePerm)
	}
	for _, url := range k8sfiles {
		// replace placeholder $VERSION with actual version parameter
		// TODO we need some way to validate this or a better way to get
		// kubelet/kubeadm/kubectl binaries
		url = strings.Replace(url, "$VERSION", version, 1)

		var fd *os.File
		fpath := path.Join(dir, path.Base(url))

		if _, err := os.Stat(fpath); os.IsNotExist(err) {
			log.Printf("%s downloading...\n", fpath)
			fd, err = Download(url, fpath)
			if err != nil {
				return fmt.Errorf("error downloading %s: %s", url, err)
			}
		} else {
			log.Printf("%s already downloaded, skipping...\n", fpath)
			fd, err = os.Open(fpath)
			if err != nil {
				return fmt.Errorf("error opening %s: %s", fpath, err)
			}
		}
		fd.Close()
	}
	return nil
}
