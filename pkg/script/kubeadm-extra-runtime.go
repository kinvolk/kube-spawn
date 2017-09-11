package script

import (
	"bytes"
)

const kubeadmExtraRuntimeTmpl string = `
Environment="--container-runtime=remote \
--runtime-request-timeout={{.RequestTimeout}} \
--container-runtime-endpoint={{.RuntimeEndpoint}}"
`

type KubeadmExtraRuntimeOpts struct {
	RuntimeEndpoint string
	RequestTimeout  string
}

func GetKubeadmExtraRuntime(opts KubeadmExtraRuntimeOpts) *bytes.Buffer {
	return render(kubeadmExtraRuntimeTmpl, opts)
}
