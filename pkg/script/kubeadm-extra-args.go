package script

import (
	"bytes"
)

const kubeadmExtraArgsTmpl string = `[Service]
Environment="KUBELET_EXTRA_ARGS=\
--cgroup-driver={{.CgroupDriver}} \
--enforce-node-allocatable= \
{{ printf "--cgroups-per-qos=%t \\" .CgroupsPerQOS }}
{{.FailSwapOnArgs}} \
--authentication-token-webhook \
{{ if .RktRuntime -}}--container-runtime=remote \
--container-runtime-endpoint={{.RuntimeEndpoint}}{{- end}} \
--runtime-request-timeout={{.RequestTimeout}}"
`

// NOTE: --fail-swap-on=false is necessary for k8s 1.8 or newer,
// and the option is not available at all in k8s 1.7 or older.
// With that option, kubelet 1.7 or older will not run at all.

// For rktlet, --container-runtime must be "remote", not "rkt".
// --container-runtime-endpoint needs to point to the unix socket,
// which rktlet listens on.

type KubeadmExtraArgsOpts struct {
	CgroupDriver    string
	CgroupsPerQOS   bool
	FailSwapOnArgs  string
	RktRuntime      bool
	RuntimeEndpoint string
	RequestTimeout  string
}

func GetKubeadmExtraArgs(opts KubeadmExtraArgsOpts) *bytes.Buffer {
	return render(kubeadmExtraArgsTmpl, opts)
}
