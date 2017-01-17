# kubeadm-systemd

## Introduction

__kubeadm-systemd__ is a tool for creating a multi-node Kubernetes cluster
on the single machine, created mostly for developers __of__ Kubernetes.

It aims to be as similar to the solutions recommendend for the production
clusters as possible.

## Architecture

![Architecture Diagram](architecture.png?raw=true "Architecture")

kubeadm-systemd uses the following third-party components to
achieve its goal:

### CNI

Raw systemd-nspawn needs systemd-networkd for automanaging the networking
for containers, which is bad for developers trying to use nspawn on their
laptops - networkd doesn't provide necessary features for desktop systems
where Network Manager is usually used. That's why we are using CNI as a
tool for providing networking for nspawn.

The integration between CNI and systemd-nspawn is made by our binary
called __cnispawn__ which creates a network namespace, executes a CNI
plugin on that, and then runs systemd-nspawn inside that namespace.
By default, systemd-nspawn doesn't create its own network namepsaces,
so the container is succesfully running inside the namespace we
created.

### Ansible

Ansible is used for running kubeadm among multiple nodes and coordinating
this process.

## Motivation

There are many other ways for setting up the development environment
of Kubernetes, however, we see some issues in them.

* `hack/local-up-cluster.sh` - it only supports single-node clusters
  and doesn't share _any_ similarity with the tools that operators
  are using for setting up k8s clusters. That brings a very huge
  risk that a developer _of_ k8s may be unable to reproduce some
  bugs or issues which happen in clusters used in production.
* [kubernetes-dind-cluster]{https://github.com/sttts/kubernetes-dind-cluster}
  - it works great and does a great job in bringing multi-node clusters
  for developers. But still, it uses its own way of creating the
  cluster. And also, in our opinion, Docker isn't very good tool
  for simulating the nodes, since it's an application container
  engine, not an operaring system container engine (like
  systemd-nspawn).
* [kubeadm-dind-cluster]{https://github.com/Mirantis/kubeadm-dind-cluster}
  - it does a great job with using kubeadm, but still, it uses Docker
  instead of any other container engine which is designed for
  containerizing the whole OS, not an application. Also, we prefer
  to maintain a code for doing such complicated things, instead of
  huge shell scripts.
