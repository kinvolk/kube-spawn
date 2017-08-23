package nspawntool

import (
	"os"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/utils"
)

func InitializeMaster(k8srelease, name string) error {
	cmd := []string{
		"/opt/kube-spawn/init.sh",
	}
	if !utils.IsK8sDev(k8srelease) {
		cmd = []string{"/opt/kube-spawn/init-release.sh", "--version", k8srelease}
	}
	return bootstrap.Exec(nil, os.Stdout, os.Stderr, name, cmd...)
}

func JoinNode(k8srelease, name, masterIP string) error {
	cmd := []string{
		"/opt/kube-spawn/join.sh",
	}
	if !utils.IsK8sDev(k8srelease) {
		cmd = []string{"/opt/kube-spawn/join-release.sh"}
	}
	cmd = append(cmd, masterIP)
	return bootstrap.Exec(nil, os.Stdout, os.Stderr, name, cmd...)
}
