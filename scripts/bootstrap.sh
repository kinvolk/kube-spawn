#!/bin/sh

set -ex

echo "root:k8s" | chpasswd

systemctl enable docker.service
systemctl enable kubelet.service
systemctl enable sshd.service
