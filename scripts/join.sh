#!/bin/sh

set -e

MASTER_IP=$1
TOKEN=$(cat /tmp/kube-spawn/token)

set -x

kubeadm reset
systemctl start kubelet.service
systemctl start rktlet.service

KUBE_HYPERKUBE_IMAGE="10.22.0.1:5000/hyperkube-amd64" kubeadm join --skip-preflight-checks --token ${TOKEN} ${MASTER_IP}:6443
