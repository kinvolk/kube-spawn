package bootstrap

import (
	"os"
	"path"

	"github.com/pkg/errors"

	"github.com/kinvolk/kube-spawn/pkg/config"
	"github.com/kinvolk/kube-spawn/pkg/script"
	"github.com/kinvolk/kube-spawn/pkg/utils/fs"
)

func rootfsPath(cfg *config.ClusterConfiguration) string {
	return path.Join(cfg.KubeSpawnDir, cfg.Name, "rootfs")
}

// GenerateScripts writes in <kube-spawn-dir>/<cluster-name>/rootfs/...
// also create empty machine specific rootfs/ dirs
//
func GenerateScripts(cfg *config.ClusterConfiguration) error {
	if err := os.MkdirAll(rootfsPath(cfg), 0755); err != nil {
		return err
	}

	if err := writeKubeadmBootstrap(cfg); err != nil {
		return err
	}
	if err := writeKubeadmExtraArgs(cfg); err != nil {
		return err
	}
	if err := writeKubeadmConfig(cfg); err != nil {
		return err
	}
	if err := fs.CreateFileFromBytes(path.Join(rootfsPath(cfg), script.DockerDaemonConfigPath), []byte(script.DockerDaemonConfig)); err != nil {
		return err
	}
	if err := fs.CreateFileFromBytes(path.Join(rootfsPath(cfg), script.DockerKubeadmExtraArgsPath), []byte(script.DockerKubeadmExtraArgs)); err != nil {
		return err
	}
	if err := fs.CreateFileFromBytes(path.Join(rootfsPath(cfg), script.KubeletTmpfilesPath), []byte(script.KubeletTmpfiles)); err != nil {
		return err
	}
	if cfg.ContainerRuntime == config.RuntimeRkt {
		if err := fs.CreateFileFromBytes(path.Join(rootfsPath(cfg), script.RktletServicePath), []byte(script.RktletService)); err != nil {
			return err
		}
	}
	if err := fs.CreateFileFromBytes(path.Join(rootfsPath(cfg), script.WeaveNetworkdUnmaskPath), []byte(script.WeaveNetworkdUnmask)); err != nil {
		return err
	}

	// create empty config dirs for all nodes
	for i := 0; i < cfg.Nodes; i++ {
		rootDir := path.Join(cfg.KubeSpawnDir, cfg.Name, config.MachineName(cfg.Name, i), "rootfs")
		if err := os.MkdirAll(path.Join(rootDir, "etc"), 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(path.Join(rootDir, "opt"), 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(path.Join(rootDir, "usr/bin"), 0755); err != nil {
			return err
		}
	}
	return nil
}

func writeKubeadmBootstrap(cfg *config.ClusterConfiguration) error {
	bootstrapScript := path.Join(rootfsPath(cfg), script.KubeadmBootstrapPath)

	buf, err := script.GetKubeadmBootstrap(script.KubeadmBootstrapOpts{
		ContainerRuntime: cfg.ContainerRuntime,
	})
	if err != nil {
		return errors.Wrapf(err, "error generating %q", bootstrapScript)
	}
	return fs.CreateFileFromReader(bootstrapScript, buf)
}

func writeKubeadmExtraArgs(cfg *config.ClusterConfiguration) error {
	extraArgsConf := path.Join(rootfsPath(cfg), script.KubeadmExtraArgsPath)
	buf, err := script.GetKubeadmExtraArgs(script.KubeadmExtraArgsOpts{
		ContainerRuntime:      cfg.ContainerRuntime,
		UseLegacyCgroupDriver: cfg.RuntimeConfiguration.UseLegacyCgroupDriver,
		CgroupsPerQOS:         cfg.RuntimeConfiguration.CgroupPerQos,
		FailSwapOn:            cfg.RuntimeConfiguration.FailSwapOn,
		RuntimeEndpoint:       cfg.RuntimeConfiguration.Endpoint,
		RequestTimeout:        cfg.RuntimeConfiguration.Timeout,
	})
	if err != nil {
		return errors.Wrapf(err, "error generating %q", extraArgsConf)
	}
	return fs.CreateFileFromReader(extraArgsConf, buf)
}

func writeKubeadmConfig(cfg *config.ClusterConfiguration) error {
	kubeadmConf := path.Join(rootfsPath(cfg), script.KubeadmConfigPath)

	buf, err := script.GetKubeadmConfig(script.KubeadmYmlOpts{
		DevCluster:        cfg.DevCluster,
		KubernetesVersion: cfg.KubernetesVersion,
		HyperkubeTag:      cfg.HyperkubeTag,
	})
	if err != nil {
		return errors.Wrapf(err, "error generating %q", kubeadmConf)
	}
	return fs.CreateFileFromReader(kubeadmConf, buf)
}
