package nspawntool

import (
	"os"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
)

func InitializeMaster(k8srelease, name string) error {
	cmd := []string{
		"/opt/kube-spawn/init.sh",
	}
	return bootstrap.Exec(nil, os.Stdout, os.Stderr, name, cmd...)
}

func JoinNode(k8srelease, name, masterIP string) error {
	cmd := []string{
		"/opt/kube-spawn/join.sh",
	}
	cmd = append(cmd, masterIP)
	return bootstrap.Exec(nil, os.Stdout, os.Stderr, name, cmd...)
}
