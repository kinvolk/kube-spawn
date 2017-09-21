package nspawntool

import "github.com/kinvolk/kube-spawn/pkg/bootstrap"

func (n *Node) Bootstrap() error {
	return bootstrap.ExecQuiet(n.Name, "/opt/kube-spawn/bootstrap.sh")
}
