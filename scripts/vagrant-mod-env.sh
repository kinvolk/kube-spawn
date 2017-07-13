#!/bin/bash

set -euo pipefail

USER=vagrant
HOME=/home/${USER}

echo 'Modifying environment'
chown -R ${USER}:${USER} ${HOME}/go/src/github.com/kinvolk/kube-spawn/k8s
chmod +x ${HOME}/build.sh

# setenforce always returns 1 when selinux is disabled.
# we should ignore the error and continue.
setenforce 0 || true
systemctl stop firewalld
sudo groupadd docker && sudo gpasswd -a ${USER} docker && sudo systemctl restart docker && newgrp docker
usermod -aG docker ${USER}
sudo modprobe overlay

NF_HASHSIZE=/sys/module/nf_conntrack/parameters/hashsize

[ -f ${NF_HASHSIZE} ] && echo "131072" | sudo tee ${NF_HASHSIZE}
