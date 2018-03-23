package cluster

import (
	"bytes"
	"text/template"
)

func ExecuteTemplate(tmplStr string, tmplData interface{}) (bytes.Buffer, error) {
	var out bytes.Buffer
	tmpl, err := template.New("").Parse(tmplStr)
	if err != nil {
		return out, err
	}
	if err := tmpl.Execute(&out, tmplData); err != nil {
		return out, err
	}
	return out, nil
}

const DockerDaemonConfig = `{
    "insecure-registries": ["10.22.0.1:5000"],
    "default-runtime": "custom",
    "runtimes": {
        "custom": { "path": "/usr/bin/kube-spawn-runc" }
    },
    "storage-driver": "overlay2"
}
`

const DockerSystemdDropin = `[Service]
Environment="DOCKER_OPTS=--exec-opt native.cgroupdriver=cgroupfs"
`

const RktletSystemdUnit = `[Unit]
Description=rktlet: The rkt implementation of a Kubernetes Container Runtime
Documentation=https://github.com/kubernetes-incubator/rktlet/tree/master/docs

[Service]
ExecStart=/usr/bin/rktlet --net=weave
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
`

// https://github.com/kinvolk/kube-spawn/issues/99
// https://github.com/weaveworks/weave/issues/2601
const WeaveSystemdNetworkdConfig = `[Match]
Name=weave datapath vethwe*

[Link]
Unmanaged=yes
`

const KubespawnBootstrapScriptTmpl = `#!/bin/bash

set -euxo pipefail

echo "root:root" | chpasswd
echo "core:core" | chpasswd

systemctl enable kubelet.service
systemctl enable sshd.service

{{ if eq .ContainerRuntime "docker" -}}systemctl start --no-block docker.service{{- end}}
{{ if eq .ContainerRuntime "rkt" -}}systemctl start --no-block rktlet.service
mkdir -p /usr/lib/rkt/plugins
ln -s /opt/cni/bin/ /usr/lib/rkt/plugins/net
ln -sfT /etc/cni/net.d /etc/rkt/net.d{{- end}}

mkdir -p /var/lib/weave

# necessary to prevent docker from being blocked
systemctl mask systemd-networkd-wait-online.service

kubeadm reset
systemctl start --no-block kubelet.service
`

// --fail-swap-on=false is necessary for k8s 1.8 or newer.

// For rktlet, --container-runtime must be "remote", not "rkt".
// --container-runtime-endpoint needs to point to the unix socket,
// which rktlet listens on.

// --cgroups-per-qos should be set to false, so that we can avoid issues with
// different formats of cgroup paths between k8s and systemd.
// --enforce-node-allocatable= is also necessary.
const KubeletSystemdDropinTmpl = `[Service]
Environment="KUBELET_CGROUP_ARGS=--cgroup-driver={{ if .UseLegacyCgroupDriver }}cgroupfs{{else}}systemd{{end}}"
Environment="KUBELET_EXTRA_ARGS=\
{{ if ne .ContainerRuntime "docker" -}}--container-runtime=remote \
--container-runtime-endpoint={{.RuntimeEndpoint}} \
--runtime-request-timeout=15m {{- end}} \
--enforce-node-allocatable= \
--cgroups-per-qos=false \
--fail-swap-on=false \
--authentication-token-webhook"
`

const KubeadmConfigTmpl = `apiVersion: kubeadm.k8s.io/v1alpha1
authorizationMode: AlwaysAllow
apiServerExtraArgs:
  insecure-port: "8080"
controllerManagerExtraArgs:
kubernetesVersion: {{.KubernetesVersion}}
schedulerExtraArgs:
{{if .HyperkubeImage -}}
unifiedControlPlaneImage: {{.HyperkubeImage}}
{{- end }}
`

const KubeSpawnRuncWrapperScript = `#!/bin/bash
# TODO: the docker-runc wrapper ensures --no-new-keyring is
# set, otherwise Docker will attempt to use keyring syscalls
# which are not allowed in systemd-nspawn containers. It can
# be removed once we require systemd v235 or later. We then
# will be able to whitelist the required syscalls; see:
# https:#github.com/systemd/systemd/pull/6798
set -euo pipefail
args=()
for arg in "${@}"; do
	args+=("${arg}")
	if [[ "${arg}" == "create" ]] || [[ "${arg}" == "run" ]]; then
		args+=("--no-new-keyring")
	fi
done
exec docker-runc "${args[@]}"
`
