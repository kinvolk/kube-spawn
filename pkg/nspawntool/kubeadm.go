package nspawntool

import (
	"path"
	"strings"

	"github.com/kinvolk/kube-spawn/pkg/bootstrap"
	"github.com/kinvolk/kube-spawn/pkg/config"
	"github.com/kinvolk/kube-spawn/pkg/machinetool"
	"github.com/kinvolk/kube-spawn/pkg/utils"
	"github.com/kinvolk/kube-spawn/pkg/utils/fs"
	"github.com/pkg/errors"
)

const weaveNet = "https://github.com/weaveworks/weave/releases/download/v2.0.5/weave-daemonset-k8s-1.7.yaml"

func InitializeMaster(cfg *config.ClusterConfiguration) error {
	// TODO: do we need a switch to turn off printing to stdout?
	var initCmd []string
	var shellOpts string
	if cfg.DevCluster {
		// TODO: remove this or implement config for it
		shellOpts = `--setenv=KUBE_HYPERKUBE_IMAGE="10.22.0.1:5000/hyperkube-amd64"`
	}
	initCmd = append(initCmd, []string{
		"/usr/bin/kubeadm", "init", "--skip-preflight-checks",
		"--config=/etc/kubeadm/kubeadm.yml"}...)

	if err := machinetool.Shell(shellOpts, cfg.Machines[0].Name, initCmd...); err != nil {
		return err
	}
	if err := machinetool.Shell(shellOpts, cfg.Machines[0].Name, "/usr/bin/kubectl", "apply", "-f", weaveNet); err != nil {
		return err
	}

	// generate and register a token for joining
	tok, err := machinetool.Output("shell", cfg.Machines[0].Name, "/usr/bin/kubeadm", "token", "generate")
	if err != nil {
		return errors.Wrap(err, "failed generating token")
	}
	cfg.Token = strings.TrimSpace(string(tok))
	if err := machinetool.Exec(cfg.Machines[0].Name, "/usr/bin/kubeadm", "token", "create", cfg.Token, "--ttl=0", cfg.TokenGroupsOption); err != nil {
		cfg.Token = "" // clear unregistered token before exit
		return errors.Wrap(err, "failed registering token")
	}

	// find IP of master
	ipStr, err := getIP(cfg.Machines[0].Name)
	if err != nil {
		return errors.Wrap(err, "failed to retrieve ip address")
	}
	cfg.Machines[0].IP = ipStr

	kubeConfigSrc := path.Join(cfg.KubeSpawnDir, cfg.Name, cfg.Machines[0].Name, "rootfs/etc/kubernetes/admin.conf")
	kubeConfigDst := path.Join(cfg.KubeSpawnDir, cfg.Name, "kubeconfig")
	if err := fs.Copy(kubeConfigSrc, kubeConfigDst); err != nil {
		return errors.Wrap(err, "failed copying kubeconfig to host")
	}
	return nil
}

func getIP(masterName string) (string, error) {
	nodes, err := bootstrap.GetRunningNodes()
	if err != nil {
		return "", err
	}

	for _, n := range nodes {
		if n.Name == masterName {
			return n.IP, nil
		}
	}
	return "", errors.New("could not find machine")
}

func JoinNode(cfg *config.ClusterConfiguration, mNo int) error {
	if cfg.Token == "" {
		return errors.New("no token found")
	}
	if cfg.Machines[0].IP == "" {
		return errors.New("no master IP found")
	}

	var joinCmd []string
	var shellOpts string
	if cfg.DevCluster {
		// TODO: remove this or implement config for it
		shellOpts = `--setenv=KUBE_HYPERKUBE_IMAGE="10.22.0.1:5000/hyperkube-amd64"`
	}
	joinCmd = append(joinCmd, []string{
		"/usr/bin/kubeadm", "join", "--skip-preflight-checks",
		"--token", cfg.Token}...)

	// --discovery-token-unsafe-skip-ca-verification appeared in Kubernetes 1.8
	// See: https://github.com/kubernetes/kubernetes/pull/49520
	// It is mandatory since Kubernetes 1.9
	// See: https://github.com/kubernetes/kubernetes/pull/55468
	// Test is !<1.8 instead of >=1.8 in order to handle non-semver version 'latest'
	if !utils.CheckVersionConstraint(cfg.KubernetesVersion, "<1.8") {
		joinCmd = append(joinCmd, "--discovery-token-unsafe-skip-ca-verification")
	}

	joinCmd = append(joinCmd, cfg.Machines[0].IP+":6443")

	return machinetool.Shell(shellOpts, cfg.Machines[mNo].Name, joinCmd...)
}
