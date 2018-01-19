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
	"fmt"
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
	DefaultKubernetesVersion = "v1.8.5"
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
		kubeadmDropinPath = path.Join(k8sBuildAssetDir, "rpms/10-kubeadm.conf")
	} else {
		// from download cache
		kubeletPath = path.Join(cacheDir, cfg.KubernetesVersion, "kubelet")
		kubeadmPath = path.Join(cacheDir, cfg.KubernetesVersion, "kubeadm")
		kubectlPath = path.Join(cacheDir, cfg.KubernetesVersion, "kubectl")
		kubeletServicePath = path.Join(cacheDir, cfg.KubernetesVersion, "kubelet.service")
		kubeadmDropinPath = path.Join(cacheDir, cfg.KubernetesVersion, "10-kubeadm.conf")
	}

	cfg.Copymap = []Pathmap{
		{Dst: "/usr/bin/kubelet", Src: kubeletPath},
		{Dst: "/usr/bin/kubeadm", Src: kubeadmPath},
		{Dst: "/usr/bin/kubectl", Src: kubectlPath},
		{Dst: "/etc/systemd/system/kubelet.service", Src: kubeletServicePath},
		{Dst: "/etc/systemd/system/kubelet.service.d/10-kubeadm.conf", Src: kubeadmDropinPath},
		// NOTE: workaround for making kubelet work with port-forward
		{Dst: "/usr/bin/socat", Src: path.Join(cacheDir, "socat")},
	}

	if cfg.DevCluster || utils.CheckVersionConstraint(cfg.KubernetesVersion, ">=1.8.0") {
		cfg.TokenGroupsOption = "--groups=system:bootstrappers:kubeadm:default-node-token"
	}

	return nil
}

func SetDefaults_RuntimeConfiguration(cfg *ClusterConfiguration) error {
	if cfg.RuntimeConfiguration.Timeout == "" {
		cfg.RuntimeConfiguration.Timeout = DefaultRuntimeTimeout
	}

	// NOTE: K8s 1.8 or newer fails to run by default when swap is enabled.
	// So we should disable the feature with an option "--fail-swap-on=false".
	if !cfg.DevCluster && utils.CheckVersionConstraint(cfg.KubernetesVersion, "<1.8.0") {
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
	default:
		return fmt.Errorf("runtime %q not supported", cfg.ContainerRuntime)
	}
	if err != nil {
		return err
	}

	// NOTE: using docker/rkt in our nodes we run out of space quick
	// TODO: can this be moved to the runtime functions below?
	cfg.Machines = make([]MachineConfiguration, cfg.Nodes)
	for i := 0; i < cfg.Nodes; i++ {
		cfg.Machines[i].Name = MachineName(cfg.Name, i)
		mountPath := path.Join(cfg.KubeSpawnDir, cfg.Name, cfg.Machines[i].Name, "mount")

		var pm Pathmap
		switch cfg.ContainerRuntime {
		case RuntimeDocker:
			pm = Pathmap{Dst: "/var/lib/docker", Src: mountPath}
		case RuntimeRkt:
			pm = Pathmap{Dst: "/var/lib/rktlet", Src: mountPath}
		case RuntimeCrio:
			pm = Pathmap{Dst: "/var/lib/containers", Src: mountPath}
		}
		cfg.Machines[i].Bindmount.ReadWrite = append(cfg.Machines[i].Bindmount.ReadWrite, pm)

		// create dirs from above if they don't exist already
		// TODO: should we create the dirs from here or move this to the check pkg
		for _, pm := range cfg.Machines[i].Bindmount.ReadWrite {
			if exists, err := fs.PathExists(pm.Src); err != nil {
				return errors.Wrap(err, "cannot determine if path exists")
			} else if !exists {
				if err := os.MkdirAll(pm.Src, 0755); err != nil {
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
	}
	if cfg.RuntimeConfiguration.Rkt.RktletBin == "" {
		cfg.RuntimeConfiguration.Rkt.RktletBin, err = exec.LookPath("rktlet")
		if err != nil {
			return err
		}
	}
	if cfg.RuntimeConfiguration.Rkt.Stage1Image == "" {
		cfg.RuntimeConfiguration.Rkt.Stage1Image = DefaultRktStage1ImagePath
	}

	pms := []Pathmap{
		{Dst: "/usr/bin/rkt", Src: cfg.RuntimeConfiguration.Rkt.RktBin},
		{Dst: "/usr/bin/rktlet", Src: cfg.RuntimeConfiguration.Rkt.RktletBin},
		{Dst: path.Join("/usr/bin/", path.Base(cfg.RuntimeConfiguration.Rkt.Stage1Image)), Src: cfg.RuntimeConfiguration.Rkt.Stage1Image},
		{Dst: "/usr/lib/rkt/plugins/net", Src: cfg.CNIPluginDir},
	}
	cfg.Bindmount.ReadOnly = append(cfg.Bindmount.ReadOnly, pms...)
	return err
}

func SetDefaults_CrioRuntime(cfg *ClusterConfiguration) error {
	// note: This lays the groundwork for supporting cri-o
	// in the future.
	// As of now it is not expected to work.
	// https://github.com/kubernetes-incubator/cri-o/blob/master/kubernetes.md
	//
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

	pms := []Pathmap{
		{Dst: "/usr/bin/crio", Src: cfg.RuntimeConfiguration.Crio.CrioBin},
		{Dst: "/usr/bin/runc", Src: cfg.RuntimeConfiguration.Crio.RuncBin},
		{Dst: "/usr/bin/conmon", Src: cfg.RuntimeConfiguration.Crio.ConmonBin},
	}
	cfg.Bindmount.ReadOnly = append(cfg.Bindmount.ReadOnly, pms...)
	return err
}

func SetDefaults_BindmountConfiguration(cfg *ClusterConfiguration) error {
	cfg.Bindmount.ReadWrite = append(cfg.Bindmount.ReadWrite, Pathmap{Dst: "/opt/cni/bin", Src: cfg.CNIPluginDir})
	return nil
}
