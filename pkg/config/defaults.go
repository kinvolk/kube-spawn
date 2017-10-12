/*
Copyright 2017 Kinvolk GmbH

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"os"
	"os/exec"
	"path"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/kinvolk/kube-spawn/pkg/utils"
	"github.com/kinvolk/kube-spawn/pkg/utils/fs"
)

const (
	DefaultKubeSpawnDir      = "/var/lib/kube-spawn"
	DefaultClusterName       = "default"
	DefaultContainerRuntime  = RuntimeDocker
	DefaultKubernetesVersion = "v1.7.5"
	DefaultBaseImage         = "coreos"

	DefaultDockerRuntimeEndpoint = "" // "unix:///var/run/docker.sock"
	DefaultRktRuntimeEndpoint    = "unix:///var/run/rktlet.sock"
	DefaultCrioRuntimeEndpoint   = "unix:///var/run/crio.sock"
	DefaultRuntimeTimeout        = "15m"
	DefaultRktStage1ImagePath    = "/usr/lib/rkt/stage1-images/stage1-coreos.aci"

	CacheDir = ".cache"
)

func SetDefaults_Viper(v *viper.Viper) {
	v.SetDefault("dir", DefaultKubeSpawnDir)
	v.SetDefault("cluster-name", DefaultClusterName)
	v.SetDefault("container-runtime", DefaultContainerRuntime)
	v.SetDefault("kubernetes-version", DefaultKubernetesVersion)
	v.SetDefault("dev", false)
	v.SetDefault("nodes", 2)
	v.SetDefault("image", DefaultBaseImage)
}

func SetDefaults_Kubernetes(cfg *ClusterConfiguration) error {
	var (
		kubeletPath, kubeadmPath, kubectlPath string
		kubeletServicePath, kubeadmDropinPath string
	)

	cacheDir := path.Join(cfg.KubeSpawnDir, CacheDir)

	if cfg.DevCluster {
		cfg.KubernetesVersion = "latest"
		// self-compiled k8s development tree
		k8sOutputDir, err := utils.GetK8sBuildOutputDir()
		if err != nil {
			return errors.Wrap(err, "error getting k8s build output directory")
		}
		kubeletPath = path.Join(k8sOutputDir, "kubelet")
		kubeadmPath = path.Join(k8sOutputDir, "kubeadm")
		kubectlPath = path.Join(k8sOutputDir, "kubectl")

		k8sBuildAssetDir, err := utils.GetK8sBuildAssetDir()
		if err != nil {
			return errors.Wrap(err, "error getting k8s build asset directory")
		}
		kubeletServicePath = path.Join(k8sBuildAssetDir, "debs/kubelet.service")
		kubeadmDropinPath = path.Join(k8sBuildAssetDir, "rpms/10-kubeadm-pre-1.8.conf")
	} else {
		// from download cache
		kubeletPath = path.Join(cacheDir, cfg.KubernetesVersion, "kubelet")
		kubeadmPath = path.Join(cacheDir, cfg.KubernetesVersion, "kubeadm")
		kubectlPath = path.Join(cacheDir, cfg.KubernetesVersion, "kubectl")
		kubeletServicePath = path.Join(cacheDir, cfg.KubernetesVersion, "kubelet.service")
		kubeadmDropinPath = path.Join(cacheDir, cfg.KubernetesVersion, "10-kubeadm-pre-1.8.conf")
	}

	cfg.Copymap = map[string]string{
		"/usr/bin/kubelet":                                              kubeletPath,
		"/usr/bin/kubeadm":                                              kubeadmPath,
		"/usr/bin/kubectl":                                              kubectlPath,
		"/etc/systemd/system/kubelet.service":                           kubeletServicePath,
		"/etc/systemd/system/kubelet.service.d/10-kubeadm-pre-1.8.conf": kubeadmDropinPath,
	}

	// NOTE: workaround for making kubelet work with port-forward
	cfg.Copymap["/usr/bin/socat"] = path.Join(cacheDir, "socat")
	return nil
}

func SetDefaults_RuntimeConfiguration(cfg *ClusterConfiguration) error {
	if cfg.RuntimeConfiguration.Timeout == "" {
		cfg.RuntimeConfiguration.Timeout = DefaultRuntimeTimeout
	}

	// K8s 1.8 or newer fails to run by default when swap is enabled.
	// So we should disable the feature with an option "--fail-swap-on=false".
	if cfg.DevCluster || utils.CheckVersionConstraint(cfg.KubernetesVersion, ">=1.8.0") {
		cfg.RuntimeConfiguration.FailSwapOn = true
	}

	var err error
	switch cfg.ContainerRuntime {
	case RuntimeDocker:
		err = SetDefaults_DockerRuntime(cfg)
	case RuntimeRkt:
		err = SetDefaults_RktRuntime(cfg)
	case RuntimeCrio:
		err = SetDefaults_CrioRuntime(cfg)
	}
	if err != nil {
		return err
	}

	// note: using docker/rkt in our nodes we run out of space quick
	// TODO: can this be moved to the runtime functions below?
	cfg.Machines = make([]MachineConfiguration, cfg.Nodes)
	for i := 0; i < cfg.Nodes; i++ {
		cfg.Machines[i].Name = MachineName(i)
		mountPath := path.Join(cfg.KubeSpawnDir, cfg.Name, cfg.Machines[i].Name, "mount")
		switch cfg.ContainerRuntime {
		case RuntimeDocker:
			cfg.Machines[i].Bindmount.ReadWrite = map[string]string{
				"/var/lib/docker": mountPath,
			}
		case RuntimeRkt:
			cfg.Machines[i].Bindmount.ReadWrite = map[string]string{
				"/var/lib/rktlet": mountPath,
			}
		case RuntimeCrio:
			cfg.Machines[i].Bindmount.ReadWrite = map[string]string{
				"/var/lib/containers": mountPath,
			}
		}

		// create dirs from above if they don't exist already
		// TODO: should we create the dirs from here or move this to the check pkg
		for _, dir := range cfg.Machines[i].Bindmount.ReadWrite {
			if !fs.Exists(dir) {
				if err := fs.CreateDir(dir); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func SetDefaults_DockerRuntime(cfg *ClusterConfiguration) error {
	var err error
	cfg.RuntimeConfiguration.Endpoint = DefaultDockerRuntimeEndpoint
	// note: cgroup driver defaults to systemd on most systems, but there's
	// an issue of runc <=1.0.0-rc2 that conflicts with --cgroup-driver=systemd,
	// so we should use legacy driver "cgroupfs".
	cfg.RuntimeConfiguration.UseLegacyCgroupDriver = true
	return err
}

func SetDefaults_RktRuntime(cfg *ClusterConfiguration) error {
	var err error
	cfg.RuntimeConfiguration.Endpoint = DefaultRktRuntimeEndpoint

	if cfg.RuntimeConfiguration.Rkt.RktBin == "" {
		cfg.RuntimeConfiguration.Rkt.RktBin, err = exec.LookPath("rkt")
		if err != nil {
			return err
		}
		cfg.Bindmount.ReadOnly["/usr/bin/rkt"] = cfg.RuntimeConfiguration.Rkt.RktBin
	}
	if cfg.RuntimeConfiguration.Rkt.RktletBin == "" {
		cfg.RuntimeConfiguration.Rkt.RktletBin, err = exec.LookPath("rktlet")
		if err != nil {
			return err
		}
		cfg.Bindmount.ReadOnly["/usr/bin/rktlet"] = cfg.RuntimeConfiguration.Rkt.RktletBin
	}
	if cfg.RuntimeConfiguration.Rkt.Stage1Image == "" {
		cfg.RuntimeConfiguration.Rkt.Stage1Image = DefaultRktStage1ImagePath
		cfg.Bindmount.ReadOnly["/usr/bin/stage1-coreos.aci"] = cfg.RuntimeConfiguration.Rkt.Stage1Image
	}

	cfg.Bindmount.ReadOnly["/usr/lib/rkt/plugins/net"] = os.Getenv("CNI_PATH")
	return err
}

func SetDefaults_CrioRuntime(cfg *ClusterConfiguration) error {
	var err error
	cfg.RuntimeConfiguration.Endpoint = DefaultCrioRuntimeEndpoint
	// cgroup driver defaults to systemd on most systems, but there's
	// an issue of runc <=1.0.0-rc2 that conflicts with --cgroup-driver=systemd,
	// so we should use legacy driver "cgroupfs".
	cfg.RuntimeConfiguration.UseLegacyCgroupDriver = true

	if cfg.RuntimeConfiguration.Crio.CrioBin == "" {
		cfg.RuntimeConfiguration.Crio.CrioBin, err = exec.LookPath("crio")
		if err != nil {
			return err
		}
	}
	if cfg.RuntimeConfiguration.Crio.RuncBin == "" {
		cfg.RuntimeConfiguration.Crio.RuncBin, err = exec.LookPath("runc")
		if err != nil {
			return err
		}
	}
	if cfg.RuntimeConfiguration.Crio.ConmonBin == "" {
		cfg.RuntimeConfiguration.Crio.ConmonBin, err = exec.LookPath("conmon")
		if err != nil {
			return err
		}
	}

	return err
}

func SetDefaults_BindmountConfiguration(cfg *ClusterConfiguration) error {
	cniPath := os.Getenv("CNI_PATH")
	if cniPath == "" {
		return errors.New("CNI_PATH was not set")
	}

	cfg.Bindmount.ReadOnly = map[string]string{
		"/opt/cni/bin": cniPath,
	}
	cfg.Bindmount.ReadWrite = map[string]string{}
	return nil
}
