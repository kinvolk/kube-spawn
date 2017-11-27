#!/bin/bash

set -eo pipefail

KUBESPAWN_DISTRO=${KUBESPAWN_DISTRO:-fedora}

# The following command sometimes exists with status 1 when using vagrant-libvirt,
# despite the fact that ssh config was generated successfully.
vagrant ssh-config $KUBESPAWN_DISTRO > $(pwd)/.ssh_config || true
scp -F $(pwd)/.ssh_config $(vagrant status | awk '/running/{print $1;}'):/home/vagrant/kubeconfig .
