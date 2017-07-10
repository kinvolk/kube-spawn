package nspawntool

import (
	"os"

	"github.com/kinvolk/kubeadm-nspawn/pkg/bootstrap"
)

func InitializeMaster(k8srelease, name string) error {
	cmd := []string{
		"/opt/kubeadm-nspawn/init.sh",
	}
	if k8srelease != "" {
		cmd = []string{"/opt/kubeadm-nspawn/init-release.sh", "--version", k8srelease}
	}
	return bootstrap.Exec(nil, os.Stdout, os.Stderr, name, cmd...)
}

func JoinNode(k8srelease, name, masterIP string) error {
	cmd := []string{
		"/opt/kubeadm-nspawn/join.sh",
	}
	if k8srelease != "" {
		cmd = []string{"/opt/kubeadm-nspawn/join-release.sh"}
	}
	cmd = append(cmd, masterIP)
	return bootstrap.Exec(nil, os.Stdout, os.Stderr, name, cmd...)
}
