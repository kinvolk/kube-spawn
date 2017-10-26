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
`

type KubeadmYmlOpts struct {
	KubernetesVersion string
}

func GetKubeadmConfig(opts KubeadmYmlOpts) (*bytes.Buffer, error) {
	return render(kubeadmConfigTmpl, opts)
}
