package nspawntool

import "github.com/kinvolk/kubeadm-nspawn/pkg/bootstrap"

func RunBootstrapScript(name string) error {
	return bootstrap.ExecQuiet(name, "/opt/kubeadm-nspawn/bootstrap.sh")
}
