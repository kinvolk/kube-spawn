# kube-spawn

`kube-spawn` is a tool for creating a multi-node Kubernetes cluster
on a single machine, created mostly for developers __of__ Kubernetes.

It aims to be as similar to the solutions recommended for production
clusters as possible.

## Requirements

* **Host:**
  - `systemd-nspawn` at least version 233

* **Kubernetes** at least version 1.7.0

## Quickstart

`kube-spawn` should run well on Fedroa 26. If you want to test it in a
controlled environment, you can use [Vagrant](doc/vagrant.md).

To setup `kube-spawn` on your machine, make sure you have a working [Go environment](https://golang.org/doc/install):

```
# Get the needed CNI plugins
$ go get -u github.com/containernetworking/plugins/plugins/main/bridge
$ go get -u github.com/containernetworking/plugins/plugins/ipam/host-local

# Get glide
$ go get -u github.com/Masterminds/glide

# Get the source for this project
$ go get -d github.com/kinvolk/kube-spawn
```

`kube-nspawn` will configure the networks it needs in `/etc/cni/net.d`.

```
# Build the tool
$ cd $GOPATH/src/github.com/kinvolk/kube-spawn
$ make vendor all

# Get the recommended container image
$ sudo machinectl pull-raw --verify=no https://alpha.release.core-os.net/amd64-usr/current/coreos_developer_container.bin.bz2 coreos

# Spawn and provision nodes for the cluster
$ sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kube-spawn up --image=coreos --nodes=3

# Setup Kubernetes
$ sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kube-spawn init
```

## Running local Kubernetes builds

One of the main use cases of `kube-spawn` is to be able to easily test patches to
Kubernetes. To do this, some additional steps are required.

To get the Hyperkube image of your local Kubernetes build to deploy, `kube-spawn` sets up
a local insecure Docker registry. Pushing images to it needs to be enabled by adding
the following to the docker daemon configuration file (`/etc/docker/daemon.json`).

```
...
      "insecure-registries": [
          "10.22.0.1:5000"
      ]
...
```

The following steps assume you have a local checkout of the Kubernetes source.

```
# Build Kubernetes
$ cd $GOPATH/src/k8s.io/kubernetes
$ build/run.sh make

# Build a Hyperkube image
$ cd cluster/images/hyperkube
$ make VERSION=latest
```

Assuming you have built `kube-spawn` and pulled the CoreOS image, do:

```
# Spawn and provision nodes for the cluster
$ sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kube-spawn --kubernetes-version=dev up --image=coreos --nodes=3

# Setup Kubernetes
$ sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kube-spawn --kubernetes-version=dev init
```

## Deploying to your local cluster

`kube-spawn` creates `tmp/` inside the directory you run it in.
There you can find the `kubeconfig` for the cluster and a `token` file with
a `kubeadm` token, that can be used to join more nodes.

To verify everything worked you can run:
```
$ kubectl --kubeconfig ./tmp/kubeconfig get nodes


$ kubectl --kubeconfig ./tmp/kubeconfig create -f 'https://github.com/kubernetes/kubernetes/blob/master/examples/guestbook/all-in-one/frontend.yaml'
```

## Command Usage

Run `kube-spawn -h`

## Troubleshooting

see [doc/troubleshooting](doc/troubleshooting.md)
