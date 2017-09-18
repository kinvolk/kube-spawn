package script

import "bytes"

const kubeadmJoinTmpl string = `#!/bin/sh

set -ex

kubeadm reset
systemctl start {{.ContainerRuntime}}.service
systemctl start kubelet.service

mkdir -p /var/lib/weave
KUBE_HYPERKUBE_IMAGE="10.22.0.1:5000/hyperkube-amd64" kubeadm join --skip-preflight-checks --token {{.Token}} {{.MasterIP}}:6443
`

type KubeadmJoinOpts struct {
	ContainerRuntime string
	Token            string
	MasterIP         string
}

func GetKubeadmJoin(opts KubeadmJoinOpts) *bytes.Buffer {
	return render(kubeadmJoinTmpl, opts)
}
