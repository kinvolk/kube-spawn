#!/bin/sh

set -e

MASTER_IP=$1
TOKEN=$(cat /tmp/kubeadm-nspawn/token)

set -x

kubeadm reset
systemctl start kubelet.service

kubeadm join --skip-preflight-checks --token ${TOKEN} ${MASTER_IP}:6443
