#!/bin/bash

set -eo pipefail

export KUBESPAWN_AUTOBUILD="true"
export KUBESPAWN_DISTRO=${KUBESPAWN_DISTRO:-fedora}
export KUBESPAWN_REDIRECT_TRAFFIC="true"

KUBESPAWN_PROVIDER=${KUBESPAWN_PROVIDER:-virtualbox}

vagrant up $KUBESPAWN_DISTRO --provider=$KUBESPAWN_PROVIDER

./vagrant-fetch-kubeconfig.sh

export KUBECONFIG=$(pwd)/kubeconfig
