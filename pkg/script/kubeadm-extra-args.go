package script

import (
	"bytes"
)

const KubeadmExtraArgsPath = "/etc/systemd/system/kubelet.service.d/20-kubeadm-extra-args.conf"

// NOTE: --fail-swap-on=false is necessary for k8s 1.8 or newer,
// and the option is not available at all in k8s 1.7 or older.
// With that option, kubelet 1.7 or older will not run at all.

// For rktlet, --container-runtime must be "remote", not "rkt".
// --container-runtime-endpoint needs to point to the unix socket,
// which rktlet listens on.

// --cgroups-per-qos should be set to false, so that we can avoid issues with
// different formats of cgroup paths between k8s and systemd.
// --enforce-node-allocatable= is also necessary.
const kubeadmExtraArgsTmpl string = `[Service]
Environment="KUBELET_CGROUP_ARGS=--cgroup-driver={{ if .UseLegacyCgroupDriver }}cgroupfs{{else}}systemd{{end}}"
Environment="KUBELET_EXTRA_ARGS=\
{{ if ne .ContainerRuntime "docker" -}}--container-runtime=remote \
--container-runtime-endpoint={{.RuntimeEndpoint}} \
--runtime-request-timeout={{.RequestTimeout}} {{- end}} \
--enforce-node-allocatable= \
{{ printf "--cgroups-per-qos=%t" .CgroupsPerQOS }} \
{{ if .FailSwapOn -}}--fail-swap-on {{- end}} \
--authentication-token-webhook"
`

type KubeadmExtraArgsOpts struct {
	ContainerRuntime      string
	UseLegacyCgroupDriver bool
	CgroupsPerQOS         bool
	FailSwapOn            bool
	RuntimeEndpoint       string
	RequestTimeout        string
}

func GetKubeadmExtraArgs(opts KubeadmExtraArgsOpts) (*bytes.Buffer, error) {
	return render(kubeadmExtraArgsTmpl, opts)
}
