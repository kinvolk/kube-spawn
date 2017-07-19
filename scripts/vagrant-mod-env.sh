#!/bin/bash

# Run it with env variable $VUSER set to a customized user, other than the
# default user "vagrant". For example on Ubuntu VM on Vagrant:
#
#  $ sudo VUSER=ubuntu ./vagrant-mod-env.sh

set -eo pipefail

if [ ${EUID} -ne 0 ]; then
	echo "This script must be run as root"
	exit 1
fi

if [ "${VUSER}" == "" ]; then
	VUSER=vagrant
fi

set -u

HOME=/home/${VUSER}

echo 'Modifying environment'
chmod +x ${HOME}/build.sh

# setenforce always returns 1 when selinux is disabled.
# we should ignore the error and continue.
/usr/sbin/setenforce 0 || true

# Stop firewalld to allow traffic by default.
# Note that especially on Debian systems, it's not sufficient to stop
# firewalld, because the FORWARD chain's policy is still DROP.
systemctl is-active firewalld 1>/dev/null && systemctl stop firewalld
iptables -P FORWARD ACCEPT
sysctl -w net.ipv4.ip_forward=1

groupadd docker && gpasswd -a ${VUSER} docker && systemctl restart docker && newgrp docker
usermod -aG docker ${VUSER}

modprobe overlay
modprobe nf_conntrack

NF_HASHSIZE=/sys/module/nf_conntrack/parameters/hashsize

[ -f ${NF_HASHSIZE} ] && echo "131072" > ${NF_HASHSIZE}
