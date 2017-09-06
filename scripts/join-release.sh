#!/bin/sh

set -e

MASTER_IP=$1
TOKEN=$(cat /tmp/kube-spawn/token)

set -x

kubeadm reset
systemctl start kubelet.service
systemctl start rktlet.service

kubeadm join --skip-preflight-checks --token ${TOKEN} ${MASTER_IP}:6443
