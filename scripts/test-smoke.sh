#!/bin/bash

# Script to test kube-spawn

set -eux
set -o pipefail

CDIR=$(cd "$(dirname "$0")" && pwd)
pushd "$CDIR"
trap 'popd' EXIT

if ! vagrant version > /dev/null 2>&1; then
	echo "Please install vagrant first"
	exit 1
fi

MSTATUS="$(vagrant status fedora |grep fedora|awk -F' ' '{print $2}')"
if [[ "${MSTATUS}" == "running" ]]; then
	vagrant halt fedora
fi

vagrant up fedora --provider=virtualbox

vagrant ssh fedora -c " \
	sudo setenforce 0; \
	go get -u github.com/containernetworking/plugins/plugins/... && \
	cd ~/go/src/github.com/kinvolk/kube-spawn && \
	DOCKERIZED=n make all && \
	sudo -E go test -v --tags integration ./tests \
	"
RESCODE=$?
if [[ "${RESCODE}" -eq 0 ]]; then
	RES="SUCCESS"
else
	RES="FAILURE"
fi

echo "Test result: ${RES}"

trap 'vagrant halt fedora' EXIT
