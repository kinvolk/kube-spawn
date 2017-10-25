# kube-spawn

`kube-spawn` is a tool for creating a multi-node Kubernetes cluster on a single Linux machine, created mostly for developers __of__ Kubernetes but should also be useful for just trying things out.

It attempts to mimic production setups by making use of OS containers to set up nodes.

[![asciicast](https://asciinema.org/a/132605.png)](https://asciinema.org/a/132605)

## Requirements

* **Host:**
  - `systemd-nspawn` at least version 233
  - `qemu-img`

* **Kubernetes** at least version 1.7.0

## Quickstart

`kube-spawn` should run well on Fedora 26. If you want to test it in a
controlled environment, you can use [Vagrant](doc/vagrant.md).

To setup `kube-spawn` on your machine, make sure you have a working [Go environment](https://golang.org/doc/install):

```
# Get CNI plugins
$ go get -u github.com/containernetworking/plugins/plugins/...

# Get the source for this project
$ go get -d github.com/kinvolk/kube-spawn
```

`kube-spawn` will configure the networks it needs in `/etc/cni/net.d`.

```
# Build the tool
$ cd $GOPATH/src/github.com/kinvolk/kube-spawn
$ make all

$ sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kube-spawn up --image=coreos --nodes=3
```

The `up` subcommand pulls the image, sets up the nodes and then configures the cluster using [kubeadm](https://github.com/kubernetes/kubeadm).

Now that you're up and running, you can start using it.

## How to..

### Deploy to your local cluster

`kube-spawn` creates the config at `/var/lib/kube-spawn/default`. There you can find the `kubeconfig` for the cluster and a `token` file with a `kubeadm` token, that can be used to join more nodes.

To verify everything worked you can run:
```
$ export KUBECONFIG=/var/lib/kube-spawn/default/kubeconfig
$ kubectl get nodes
$ kubectl get pods --all-namespaces
$ kubectl create -f 'https://raw.githubusercontent.com/kubernetes/kubernetes/master/examples/guestbook/all-in-one/frontend.yaml'
```

It is possible to run `rktlet` on `kube-spawn`. See [doc/rktlet](doc/rktlet.md).

### Run local Kubernetes builds

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

# Build a Hyperkube Docker image
$ cd cluster/images/hyperkube
$ make VERSION=latest
```

This will create a Docker image with a name `gcr.io/google-containers/hyperkube-amd64`.
To check if it is created correctly, do so:

```
$ docker images | grep hyperkube-amd64
gcr.io/google-containers/hyperkube-amd64               latest                        179a38ef4d78        45 hours ago        506 MB
```

Assuming you have built `kube-spawn` and pulled the CoreOS image, do:

```
# Spawn and provision nodes for the cluster
$ sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kube-spawn --kubernetes-version=dev setup --image=coreos --nodes=3

# Setup Kubernetes
$ sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kube-spawn --kubernetes-version=dev init
```

### Access a kube-spawn node

All nodes can be seen with `machinectl list`, `machinectl shell` can be used to access a node, for example:

```
sudo machinectl shell root@kubespawn0
```

The password is `k8s`.


## Command Usage

Run `kube-spawn -h`

## Troubleshooting

see [doc/troubleshooting](doc/troubleshooting.md)
