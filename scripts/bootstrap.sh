#!/bin/sh

set -e

echo "kubernetes" | passwd --stdin root

mkdir -p /etc/docker
cat >/etc/docker/daemon.json <<-EOF
{
    "insecure-registries": ["10.22.0.1:5000"],
    "default-runtime": "nspawn",
    "runtimes": {
        "oci": { "path": "/usr/libexec/docker/docker-runc-current" },
        "nspawn": { "path": "/opt/nspawn-runc" }
    },
    "storage-driver": "overlay"
}
EOF

mkdir -p /etc/systemd/system/docker.service.d
cat >/etc/systemd/system/docker.service.d/20-kubeadm-extra-args.conf <<-EOF
[Service]
ExecStart=
ExecStart=/usr/bin/dockerd-current \
          --containerd /run/containerd.sock \
          --exec-opt native.cgroupdriver=systemd \
          --userland-proxy-path=/usr/libexec/docker/docker-proxy-current \
          $OPTIONS \
          $DOCKER_STORAGE_OPTIONS \
          $DOCKER_NETWORK_OPTIONS \
          $ADD_REGISTRY \
          $BLOCK_REGISTRY \
          $INSECURE_REGISTRY
EOF

systemctl daemon-reload || true
systemctl enable docker.service
systemctl enable sshd.service

mkdir -p /etc/kubeadm
cat >/etc/kubeadm/kubeadm.yml <<-EOF
apiVersion: kubeadm.k8s.io/v1alpha1
authorizationMode: AlwaysAllow
kubernetesVersion: latest
apiServerExtraArgs:
  insecure-port: "8080"
controllerManagerExtraArgs:
schedulerExtraArgs:
EOF

mkdir -p /etc/systemd/system/kubelet.service.d
cat >/etc/systemd/system/kubelet.service.d/20-kubeadm-extra-args.conf <<-EOF
[Service]
Environment="KUBELET_EXTRA_ARGS=\
--cgroup-driver=systemd \
--enforce-node-allocatable= \
--cgroups-per-qos=false \
--authentication-token-webhook"
EOF
