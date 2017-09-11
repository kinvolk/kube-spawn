package script

import "bytes"

const kubeadmJoinTmpl string = `
#!/bin/sh

set -ex

kubeadm reset
systemctl start kubelet.service
{{ if .RuntimeRkt -}}systemctl start rktlet.service{{- end}}

mkdir -p /var/lib/weave
KUBE_HYPERKUBE_IMAGE="10.22.0.1:5000/hyperkube-amd64" kubeadm join --skip-preflight-checks --token {{.Token}} ${{.MasterIP}}:6443
`

type KubeadmJoinOpts struct {
	RuntimeRkt bool
	Token      string
	MasterIP   string
}

func GetKubeadmInit(opts KubeadmJoinOpts) *bytes.Buffer {
	return render(kubeadmJoinTmpl, opts)
}
