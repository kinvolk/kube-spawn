#!/bin/sh

set -e

USAGE="""
init.sh - kube-spawn master node init script
! for dev clusters
"""

if [ "$1" = "--help" ]; then
	echo $USAGE
	exit
fi

set -x

KUBEADM_NSPAWN_TMP=/tmp/kube-spawn

kubeadm reset
systemctl start kubelet.service
systemctl start rktlet.service

KUBE_HYPERKUBE_IMAGE="10.22.0.1:5000/hyperkube-amd64" kubeadm init --skip-preflight-checks --config /etc/kubeadm/kubeadm.yml
kubeadm token generate > ${KUBEADM_NSPAWN_TMP}/token
kubeadm token create $(cat ${KUBEADM_NSPAWN_TMP}/token) --description 'kube-spawn bootstrap token' --ttl 0

mkdir /var/lib/weave
kubectl apply -f https://git.io/weave-kube-1.6

install /etc/kubernetes/admin.conf ${KUBEADM_NSPAWN_TMP}/kubeconfig
