package script

import (
	"bytes"
)

const KubeadmConfigPath = "/etc/kubeadm/kubeadm.yml"

const kubeadmConfigTmpl = `apiVersion: kubeadm.k8s.io/v1alpha1
authorizationMode: AlwaysAllow
apiServerExtraArgs:
  insecure-port: "8080"
controllerManagerExtraArgs:
kubernetesVersion: {{.KubernetesVersion}}
schedulerExtraArgs:
{{if .DevCluster -}}
unifiedControlPlaneImage: 10.22.0.1:5000/hyperkube-amd64:{{.HyperkubeTag}}
{{- end }}
`

type KubeadmYmlOpts struct {
	DevCluster        bool
	KubernetesVersion string
	HyperkubeTag      string
}

func GetKubeadmConfig(opts KubeadmYmlOpts) (*bytes.Buffer, error) {
	return render(kubeadmConfigTmpl, opts)
}
