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

# Run iptables to allow CNI traffic by default.
iptables -C FORWARD -i cni0 -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -i cni0 -j ACCEPT

# Note that especially on Debian systems, it's not sufficient to add
# a single iptables rule, because the FORWARD chain's policy is still DROP.
iptables -P FORWARD ACCEPT
sysctl -w net.ipv4.ip_forward=1

modprobe overlay
modprobe nf_conntrack

NF_HASHSIZE=/sys/module/nf_conntrack/parameters/hashsize

[ -f ${NF_HASHSIZE} ] && echo "131072" > ${NF_HASHSIZE}

# systemd-nspawn containers are not able to resolve DNS, if systemd-resolved
# is running on the host.
# As workaround, we need to disable stub listener of systemd-resolved for now.
# We also need to explicitly set nameserver to an external one, as
# /etc/resolv.conf is a symlink that points to
# /run/systemd/resolve/stub-resolv.conf created by systemd-resolved.
# This is hacky, but it's at least necessary for systemd v234, the default
# version on Ubuntu 17.10.
sudo sed -i -e 's/^#*.*DNSStubListener=.*$/DNSStubListener=no/' /etc/systemd/resolved.conf
sudo sed -i -e 's/nameserver 127.0.0.53/nameserver 8.8.8.8/' /etc/resolv.conf
systemctl is-active systemd-resolved >& /dev/null && sudo systemctl stop systemd-resolved
systemctl is-enabled systemd-resolved >& /dev/null && sudo systemctl disable systemd-resolved
