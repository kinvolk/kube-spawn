#!/bin/sh

kubeadm reset
KUBE_HYPERKUBE_IMAGE=10.22.0.1:5000/hyperkube-amd64 kubeadm init --skip-preflight-checks --config /etc/kubeadm/kubeadm.yml
kubectl -n kube-system get ds -l 'component=kube-proxy' -o json | jq '.items[0].spec.template.spec.containers[0].command |= .+ ["--conntrack-max-per-core=0"]' | kubectl apply -f - && kubectl -n kube-system delete pods -l 'component=kube-proxy'
kubectl apply -f weave-daemonset.yaml
