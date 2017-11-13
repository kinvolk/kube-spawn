package script

const kubeadmExtraRuntimeTmpl string = `
{{ if .RktRuntime -}}--container-runtime=remote \
--container-runtime-endpoint={{.RuntimeEndpoint}}{{- end}} \
--runtime-request-timeout={{.RequestTimeout}}"
`

// For rktlet, --container-runtime must be "remote", not "rkt".
// --container-runtime-endpoint needs to point to the unix socket,
// which rktlet listens on.

type KubeadmExtraRuntimeOpts struct {
	RktRuntime      bool
	RuntimeEndpoint string
	RequestTimeout  string
}
