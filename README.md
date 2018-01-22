![kube-spawn Logo](logos/PNG/kube_spawn-horz_prpblkonwht.png)

# kube-spawn

<img src="https://raw.githubusercontent.com/cncf/artwork/master/kubernetes/certified-kubernetes/versionless/color/certified_kubernetes_color.png" align="right" width="100px">`kube-spawn` is a tool for creating a multi-node Kubernetes (>= 1.7) cluster on a single Linux machine, created mostly for developers __of__ Kubernetes but is also a [Certified Kubernetes Distribution](https://kubernetes.io/partners/#dist) and, therefore, perfect for running and testing deployments locally.

It attempts to mimic production setups by making use of OS containers to set up nodes.

## Demo

[![asciicast](https://asciinema.org/a/132605.png)](https://asciinema.org/a/132605)

## Requirements

* `systemd-nspawn` at least version 233
* `qemu-img`

## Installation

`kube-spawn` should run well on a modern Linux system (for example Fedora 27 or
Debian testing). If you want to test it in a controlled environment, you can
use [Vagrant](doc/vagrant.md).

To setup `kube-spawn` on your machine, make sure you have a working [Go
environment](https://golang.org/doc/install).

kube-spawn uses CNI to setup networking for its containers. For that, you need
to download the CNI plugins (v.0.6.0 or later) from GitHub. Example:

```
cd /tmp
curl -fsSL -O https://github.com/containernetworking/plugins/releases/download/v0.6.0/cni-plugins-amd64-v0.6.0.tgz
sudo mkdir -p /opt/cni/bin
sudo tar -C /opt/cni/bin -xvf cni-plugins-amd64-v0.6.0.tgz
```

Alternatively, you can use `go get` to fetch the plugins into your `GOPATH`:

```
go get -u github.com/containernetworking/plugins/plugins/...
```

(Note: that requires `--cni-plugin-dir=$GOPATH/bin` later.)

To build kube-spawn in a Docker build container, simply run:

```
make
```

Optionally, install kube-spawn under a system directory:

```
sudo make install
```

(`PREFIX` can be set to override the default target `/usr`.)

## Quickstart

Create and start a 3 node cluster with the name "default":

```
sudo ./kube-spawn create --nodes=3
sudo ./kube-spawn start
```

Note: if the CNI plugins cannot be found in `/opt/cni/bin`, you need to
pass `--cni-plugin-dir path/to/plugins` to create.

`create` sets up the cluster environment in `/var/lib/kube-spawn`, and puts all
the neccessary scripts/configs into place.

`start` brings up the nodes and configures the cluster using
[kubeadm](https://github.com/kubernetes/kubeadm).

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

If you don't have `kubectl`, you can get it with:
```
KUBERNETES_VERSION=$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)
sudo curl -Lo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl
sudo chmod +x /usr/local/bin/kubectl
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
$ git describe
v1.8.5-beta.0
$ make -C cluster/images/hyperkube VERSION=v1.8.5-beta.0-myfeature
```

This will create a Docker image with a name `gcr.io/google-containers/hyperkube-amd64:v1.8.5-beta.0-myfeature`.
To check if it is created correctly, do so:

```
$ docker images | grep hyperkube-amd64
gcr.io/google-containers/hyperkube-amd64               v1.8.5-beta.0-myfeature                     8687537eff68        10 minutes ago      530 MB
```

Assuming you have built `kube-spawn` and pulled the CoreOS image, do:

```
# Spawn and provision nodes for the cluster
$ sudo -E ./kube-spawn create --dev -t v1.8.5-beta.0-myfeature -c myfeature
$ sudo -E ./kube-spawn start -c myfeature
```

For a specific example, see [doc/dev-workflow](doc/dev-workflow.md).

### Access a kube-spawn node

All nodes can be seen with `machinectl list`, `machinectl shell` can be used to access a node, for example:

```
sudo machinectl shell root@kubespawn0
```

The password is `root`.


## Command Usage

Run `kube-spawn -h`

## Troubleshooting

see [doc/troubleshooting](doc/troubleshooting.md)
