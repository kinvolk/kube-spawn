package script

import "bytes"

const kubeadmBootstrapTmpl string = `#!/bin/sh

set -ex

echo "root:k8s" | chpasswd
echo "core:core" | chpasswd

systemctl enable {{.ContainerRuntime}}.service
systemctl enable kubelet.service
systemctl enable sshd.service

# necessary to prevent docker from being blocked.
systemctl mask systemd-networkd-wait-online.service

`

type KubeadmBootstrapOpts struct {
	ContainerRuntime string
}

func GetKubeadmBootstrap(opts KubeadmBootstrapOpts) *bytes.Buffer {
	return render(kubeadmBootstrapTmpl, opts)
}
