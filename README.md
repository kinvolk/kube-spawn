# kubeadm-systemd

## Introduction

__kubeadm-systemd__ is a tool for creating a multi-node Kubernetes cluster
on the single machine, created mostly for developers __of__ Kubernetes.
However, it may broaden its focus later.

It aims to be as similar to the solutions recommended for the production
clusters as possible.

## Architecture

![Architecture Diagram](architecture.png?raw=true "Architecture")

kubeadm-systemd uses the following third-party components to
achieve its goal:

### CNI

We chose to use CNI to provide networking for systemd-nspawn as it's what
is used in kubernetes itself and systemd-networkd is currently tricky to use.
We want to get kubeadm-systemd usable for the developers using their
laptops or desktop workstations, where NetworkManager is used instead
of systemd-networkd.

The integration between CNI and systemd-nspawn is made by our binary
called __cnispawn__ which creates a network namespace, executes a CNI
plugin on that, and then runs systemd-nspawn inside that namespace.
By default, systemd-nspawn doesn't create its own network namepsaces,
so the container is successfully running inside the namespace we
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
* [kubernetes-dind-cluster](https://github.com/sttts/kubernetes-dind-cluster) -
  it works great and does a great job in bringing multi-node clusters
  for developers. But still, it uses custom scripts to set up the
  cluster. Also, as Docker is an app container engine, it does not simulate an
  operating system, thus providing a very different environment to what runs
  on production nodes.
* [kubeadm-dind-cluster](https://github.com/Mirantis/kubeadm-dind-cluster) -
  it does a great job of making use of kubeadm. But as it uses Docker,
  it has the same issue as mentioned above; not simulating production nodes.
  Also, we prefer having the complexity of setting this up inside of Go code
  instead of shell scripts.
