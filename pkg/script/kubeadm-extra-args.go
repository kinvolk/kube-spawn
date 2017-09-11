package script

import (
	"bytes"
)

const kubeletExtraArgsTmpl string = `
[Service],
Environment="KUBELET_EXTRA_ARGS=\
--cgroup-driver=cgroupfs \
--enforce-node-allocatable= \
{{ printf "--cgroups-per-qos=%t \\" .CgrouptsPerQOS }}
{{ printf "--fail-wrap-on=%t \\" .FailSwapOn }}
--authentication-token-webhook"
`

type KubeletExtraArgsOpts struct {
	CgrouptsPerQOS bool
	FailSwapOn     bool
}

func GetKubeletExtraArgs(opts KubeadmExtraArgsOpts) bytes.Buffer {
	return render(kubeletExtraArgsTmpl, opts)
}
