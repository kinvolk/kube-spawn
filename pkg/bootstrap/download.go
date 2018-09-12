package bootstrap

import (
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/kinvolk/kube-spawn/pkg/utils"
	"github.com/kinvolk/kube-spawn/pkg/utils/fs"
	"github.com/pkg/errors"
)

const (
	k8sURL         string = "https://dl.k8s.io/$VERSION/bin/linux/amd64/"
	k8sGithubURL   string = "https://raw.githubusercontent.com/kubernetes/kubernetes/$VERSION/build/rpms/"
	staticSocatUrl string = "https://raw.githubusercontent.com/andrew-d/static-binaries/530df977dd38ba3b4197878b34466d49fce69d8e/binaries/linux/x86_64/socat"

	sha1Suffix string = ".sha1"
)

var (
	// note: we are downloading these in parallel (limit number or improve DownloadK8sBins func)
	//
	// key: full URL for a file to be downloaded.
	// value: whether it has to be verified with checksum or not
	k8sBinaryFiles = map[string]bool{
		k8sURL + "kubelet":               true,
		k8sURL + "kubeadm":               true,
		k8sURL + "kubectl":               true,
		k8sGithubURL + "kubelet.service": false,
		k8sGithubURL + "10-kubeadm.conf": false,
	}
)

func Download(url, fpath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.Errorf("server returned [%d] %q", resp.StatusCode, resp.Status)
	}
	return fs.CreateFileFromReader(fpath, resp.Body)
}

func DownloadKubernetesBinaries(k8sVersion, targetDir string) error {
	var err error
	versionPath := path.Join(targetDir, k8sVersion)
	if exists, err := fs.PathExists(versionPath); err != nil {
		return err
	} else if !exists {
		if err := os.MkdirAll(versionPath, 0755); err != nil {
			return err
		}
	}

	var wg sync.WaitGroup
	wg.Add(len(k8sBinaryFiles))
	for url, verifyHash := range k8sBinaryFiles {
		// replace placeholder $VERSION with actual version parameter
		go func(url string, verifyHash bool) {
			defer wg.Done()
			url = strings.Replace(url, "$VERSION", k8sVersion, 1)
			inCachePath := path.Join(versionPath, path.Base(url))
			if exists, err := fs.PathExists(inCachePath); err != nil {
				log.Printf("Error checking if path %q exists: %v\n", inCachePath, err)
				return
			} else if !exists {
				log.Printf("Downloading %s", path.Base(inCachePath))
				if err = Download(url, inCachePath); err != nil {
					err = errors.Wrapf(err, "error downloading %s", url)
					return
				}

				if verifyHash {
					sha1URL := url + sha1Suffix
					sha1CachePath := inCachePath + sha1Suffix
					if err = Download(sha1URL, sha1CachePath); err != nil {
						err = errors.Wrapf(err, "error downloading %s", sha1URL)
						return
					}

					if err = utils.VerifySha1(inCachePath, sha1CachePath); err != nil {
						err = errors.Wrapf(err, "error verifying checksum of %s", sha1URL)
						return
					}
				}
			}
		}(url, verifyHash)
	}
	wg.Wait()

	return err
}

func DownloadSocatBin(targetDir string) error {
	if exists, err := fs.PathExists(targetDir); err != nil {
		return err
	} else if !exists {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}
	}
	inCachePath := path.Join(targetDir, path.Base(staticSocatUrl))

	if exists, err := fs.PathExists(inCachePath); err != nil {
		return err
	} else if !exists {
		log.Printf("downloading %s", path.Base(inCachePath))
		if err := Download(staticSocatUrl, inCachePath); err != nil {
			return errors.Wrapf(err, "error downloading %s", staticSocatUrl)
		}
	}
	return nil
}
