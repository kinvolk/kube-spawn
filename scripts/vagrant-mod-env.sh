#!/bin/bash

set -euo pipefail

if [ ${EUID} -ne 0 ]; then
	echo "This script must be run as root"
	exit 1
fi

USER=vagrant
HOME=/home/${USER}

echo 'Modifying environment'
chmod +x ${HOME}/build.sh

# setenforce always returns 1 when selinux is disabled.
# we should ignore the error and continue.
setenforce 0 || true
systemctl is-active firewalld 1>/dev/null && systemctl stop firewalld
groupadd docker && gpasswd -a ${USER} docker && systemctl restart docker && newgrp docker
usermod -aG docker ${USER}

modprobe overlay
modprobe nf_conntrack

NF_HASHSIZE=/sys/module/nf_conntrack/parameters/hashsize

[ -f ${NF_HASHSIZE} ] && echo "131072" > ${NF_HASHSIZE}
