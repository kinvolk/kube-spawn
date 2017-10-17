package bootstrap

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/kinvolk/kube-spawn/pkg/utils"
)

const (
	k8sURL         string = "https://dl.k8s.io/v$VERSION/bin/linux/amd64/"
	k8sGithubURL   string = "https://raw.githubusercontent.com/kubernetes/release/706c64874faa5653cf963fb30de390969e25f175/rpm/"
	staticSocatUrl string = "https://raw.githubusercontent.com/andrew-d/static-binaries/530df977dd38ba3b4197878b34466d49fce69d8e/binaries/linux/x86_64/socat"
)

var (
	k8sfiles = []string{
		k8sURL + "kubelet",
		k8sURL + "kubeadm",
		k8sURL + "kubectl",
		k8sGithubURL + "kubelet.service",
		k8sGithubURL + "10-kubeadm-pre-1.8.conf",
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

func DownloadSocatBin(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, os.ModePerm)
	}

	var fd *os.File
	fpath := filepath.Join(dir, path.Base(staticSocatUrl))

	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		log.Printf("%s downloading...\n", fpath)
		fd, err = Download(staticSocatUrl, fpath)
		if err != nil {
			return fmt.Errorf("error downloading %s: %s", staticSocatUrl, err)
		}
	} else {
		log.Printf("%s already downloaded, skipping...\n", fpath)
		fd, err = os.Open(fpath)
		if err != nil {
			return fmt.Errorf("error opening %s: %s", fpath, err)
		}
	}
	fd.Close()
	return nil
}

func CheckForK8sBinVersions(expectedVer, k8sdir string) error {
	// if /var/lib/kube-spawn/k8s does not exist, there's nothing to do.
	if _, err := os.Stat(k8sdir); os.IsNotExist(err) {
		return nil
	}

	if err := checkKubectlVersion(expectedVer, k8sdir); err != nil {
		return err
	}

	if err := checkKubeletVersion(expectedVer, k8sdir); err != nil {
		return err
	}

	if err := checkKubeadmVersion(expectedVer, k8sdir); err != nil {
		return err
	}

	return nil
}

// checkKubectlVersion checks for versions of a pre-existing binary
// by running "kubectl version --client=true --short=true".
func checkKubectlVersion(expectedVer, k8sdir string) error {
	kubectlPath := filepath.Join(k8sdir, "kubectl")

	if err := utils.CheckValidFile(kubectlPath); err != nil {
		/// nothing to do. skip version checking
		return nil
	}

	args := []string{
		kubectlPath,
		"version",
		"--client=true",
		"--short=true",
	}

	cmd := exec.Cmd{
		Path: kubectlPath,
		Args: args,
		Env:  os.Environ(),
	}

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cannot run kubectl version command: %v\n", err)
	}

	binVer := ""

	s := bufio.NewScanner(strings.NewReader(string(out)))
	for s.Scan() {
		// example output:
		//  Client version: v1.7.5
		line := s.Text()

		verFields := strings.Split(line, ":")
		if len(verFields) <= 1 {
			continue
		}

		// strip out a prefix "v" and go to the next step
		ver := strings.TrimSpace(verFields[1])
		if ver != "" {
			binVer = strings.TrimPrefix(ver, "v")
			break
		}
	}

	if binVer == "" {
		return errors.New("unable to get the kubectl's version")
	}

	if !utils.IsSemVer(binVer, expectedVer) {
		return fmt.Errorf("pre-existing kubectl binary's version = %s, expected = %s.", binVer, expectedVer)
	}

	return nil
}

// checkKubeletVersion checks for versions of a pre-existing binary
// by running "kubelet --version".
func checkKubeletVersion(expectedVer, k8sdir string) error {
	kubeletPath := filepath.Join(k8sdir, "kubelet")

	if err := utils.CheckValidFile(kubeletPath); err != nil {
		/// nothing to do. skip version checking
		return nil
	}

	args := []string{
		kubeletPath,
		"--version",
	}

	cmd := exec.Cmd{
		Path: kubeletPath,
		Args: args,
		Env:  os.Environ(),
	}

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cannot run kubelet version command: %v\n", err)
	}

	binVer := ""

	s := bufio.NewScanner(strings.NewReader(string(out)))
	for s.Scan() {
		// example output:
		//   Kubernetes v1.7.5
		line := strings.Fields(s.Text())
		if len(line) <= 1 {
			continue
		}

		ver := strings.TrimSpace(line[1])
		if ver == "" {
			continue
		}

		// strip out a prefix "v" and go to the next step
		ver = strings.TrimPrefix(ver, "v")
		if ver != "" {
			binVer = ver
			break
		}
	}

	if binVer == "" {
		return errors.New("unable to get the kubelet's version")
	}

	if !utils.IsSemVer(binVer, expectedVer) {
		return fmt.Errorf("pre-existing kubelet binary's version = %s, expected = %s.", binVer, expectedVer)
	}

	return nil
}

// checkKubeadmVersion checks for versions of a pre-existing binary
// by running "kubeadm --version".
func checkKubeadmVersion(expectedVer, k8sdir string) error {
	kubeadmPath := filepath.Join(k8sdir, "kubeadm")

	if err := utils.CheckValidFile(kubeadmPath); err != nil {
		/// nothing to do. skip version checking
		return nil
	}

	args := []string{
		kubeadmPath,
		"version",
		"--output=short",
	}

	cmd := exec.Cmd{
		Path: kubeadmPath,
		Args: args,
		Env:  os.Environ(),
	}

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cannot run kubeadm version command: %v\n", err)
	}

	binVer := ""

	s := bufio.NewScanner(strings.NewReader(string(out)))
	for s.Scan() {
		// example output:
		//   v1.7.5
		line := strings.Fields(s.Text())

		ver := strings.TrimSpace(line[0])
		if ver == "" {
			continue
		}

		// strip out a prefix "v" and go to the next step
		ver = strings.TrimPrefix(ver, "v")
		if ver != "" {
			binVer = ver
			break
		}
	}

	if binVer == "" {
		return errors.New("unable to get the kubeadm's version")
	}

	if !utils.IsSemVer(binVer, expectedVer) {
		return fmt.Errorf("pre-existing kubeadm binary's version = %s, expected = %s.", binVer, expectedVer)
	}

	return nil
}
