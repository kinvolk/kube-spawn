package script

import "bytes"

const kubeadmInitTmpl string = `#!/bin/sh

set -ex

kubeadm reset
systemctl start kubelet.service
{{ if .RuntimeRkt -}}systemctl start rktlet.service{{- end}}

kubeadm init --skip-preflight-checks --config /etc/kubeadm/kubeadm.yml
mkdir -p {{.KubeSpawnDir}}
kubeadm token generate > {{.KubeSpawnDir}}/token
kubeadm token create $(cat {{.KubeSpawnDir}}/token) --description 'kube-spawn bootstrap token' --ttl 0

mkdir -p /var/lib/weave
{{- if .RuntimeRkt}}ln -sfT /etc/cni/net.d /etc/rkt/net.d{{end -}}
kubectl apply -f https://git.io/weave-kube-1.6

install /etc/kubernetes/admin.conf {{.KubeSpawnDir}}/kubeconfig
`

type KubeadmInitOpts struct {
	RuntimeRkt   bool
	KubeSpawnDir string
}

func GetKubeadmInit(opts KubeadmInitOpts) *bytes.Buffer {
	return render(kubeadmInitTmpl, opts)
}
