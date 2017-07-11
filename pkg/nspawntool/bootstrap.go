package nspawntool

import "github.com/kinvolk/kube-spawn/pkg/bootstrap"

func RunBootstrapScript(name string) error {
	return bootstrap.ExecQuiet(name, "/opt/kube-spawn/bootstrap.sh")
}
