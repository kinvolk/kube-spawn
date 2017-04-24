#!/bin/sh

set -e

echo "kubernetes" | passwd --stdin root

chmod +x /root/init.sh

cat >>/etc/sysconfig/docker <<-EOF
INSECURE_REGISTRY='--insecure-registry=10.22.0.1:5000'
EOF

cat >/etc/sysconfig/docker-storage <<-EOF
DOCKER_STORAGE_OPTIONS="--storage-driver overlay"
EOF

cat >/etc/sysconfig/docker-storage-setup <<-EOF
STORAGE_DRIVER="overlay"
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
--cgroup-driver=cgroupfs \
--enforce-node-allocatable= \
--cgroups-per-qos=false \
--authentication-token-webhook"
EOF
