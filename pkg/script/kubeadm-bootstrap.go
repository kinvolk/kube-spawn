package script

import "bytes"

const KubeadmBootstrapPath = "/opt/kube-spawn/bootstrap.sh"

const kubeadmBootstrapTmpl = `#!/bin/sh

set -ex

echo "root:k8s" | chpasswd
echo "core:core" | chpasswd

systemctl enable kubelet.service
systemctl enable sshd.service

{{ if eq .ContainerRuntime "docker" -}}systemctl start --no-block docker.service{{- end}}
{{ if eq .ContainerRuntime "crio" -}}systemctl start --no-block crio.service{{- end}}
{{ if eq .ContainerRuntime "rkt" -}}systemctl start --no-block rktlet.service
ln -sfT /etc/cni/net.d /etc/rkt/net.d{{- end}}

mkdir -p /var/lib/weave

# necessary to prevent docker from being blocked.
systemctl mask systemd-networkd-wait-online.service

kubeadm reset
systemctl start --no-block kubelet.service
`

type KubeadmBootstrapOpts struct {
	ContainerRuntime string
}

func GetKubeadmBootstrap(opts KubeadmBootstrapOpts) (*bytes.Buffer, error) {
	return render(kubeadmBootstrapTmpl, opts)
}
