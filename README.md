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

# (optional) Install binaries under a system directory.
# The install prefix defaults to /usr, which you can override with an env
# variable $PREFIX, like "make PREFIX=/usr/local install".
$ sudo make install

$ export CNI_PATH=$GOPATH/bin

# This generated a default 3 nodes cluster configuration
$ sudo -E ./kube-spawn create --nodes=3
$ sudo -E ./kube-spawn start
```

The `create` subcommand sets up a cluster environment in `/var/lib/kube-spawn`, and puts all the neccessary
scripts/configs into place for running the cluster.
Via `start` you bring up the nodes and `kube-spawn` configures the cluster using [kubeadm](https://github.com/kubernetes/kubeadm).
Now that you're up and running, you can start using it.

After stopping the cluster with `stop` you don't have to run `create` again, unless you want to change the cluster config in
`/var/lib/kube-spawn/CLUSTER_NAME/kspawn.toml`.

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

The password is `k8s`.


## Command Usage

Run `kube-spawn -h`

## Troubleshooting

see [doc/troubleshooting](doc/troubleshooting.md)
