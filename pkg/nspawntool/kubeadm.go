package nspawntool

import (
	"os"

	"github.com/kinvolk/kubeadm-nspawn/pkg/bootstrap"
)

func InitializeMaster(name string) error {
	return bootstrap.Exec(nil, os.Stdout, os.Stderr, name, "/opt/kubeadm-nspawn/init.sh")
}

func JoinNode(name, masterIP string) error {
	return bootstrap.Exec(nil, os.Stdout, os.Stderr, name, "/opt/kubeadm-nspawn/join.sh", masterIP)
}
