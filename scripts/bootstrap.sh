#!/bin/sh

set -ex

echo "root:k8s" | chpasswd
echo "core:core" | chpasswd

systemctl enable docker.service
systemctl enable kubelet.service
systemctl enable rktlet.service
systemctl enable sshd.service

# necessary to prevent docker from being blocked.
systemctl mask systemd-networkd-wait-online.service
