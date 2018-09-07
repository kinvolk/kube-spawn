![kube-spawn Logo](logos/PNG/kube_spawn-horz_prpblkonwht.png)

# kube-spawn

<img src="https://github.com/cncf/artwork/raw/8760b54868864a24459716cd0e9ba9986de882f8/kubernetes/certified-kubernetes/versionless/color/certified-kubernetes-color.png" align="right" width="100px"> `kube-spawn` is a tool for creating a multi-node Kubernetes (>= 1.8) cluster on a single Linux machine, created mostly for developers __of__ Kubernetes but is also a [Certified Kubernetes Distribution](https://kubernetes.io/partners/#dist) and, therefore, perfect for running and testing deployments locally.

It attempts to mimic production setups by making use of OS containers to set up nodes.

## Demo

[![asciicast](https://asciinema.org/a/132605.png)](https://asciinema.org/a/132605)

## Requirements

* `systemd-nspawn` in at least version 233
* Large enough `/var/lib/machines` partition.

  If /var/lib/machines is not its own filesystem, systemd-nspawn
  will create /var/lib/machines.raw and loopback mount it as a
  btrfs filesystem. You may wish to increase the default size:

  `machinectl set-limit 20G`

  We recommend you create a partition of sufficient size, format
  it as btrfs, and mount it on /var/lib/machines, rather than
  letting the loopback mechanism take hold.

  In the event there is a loopback file mounted on /var/lib/machines,
  kube-spawn will attempt to enlarge the underlying image `/var/lib/machines.raw`
  on cluster start, but this can only succeed when the image is not in use by
  another cluster or machine. Not enough disk space is a common source
  of error. See [doc/troubleshooting](doc/troubleshooting.md#varlibmachines-partition-too-small) for
  instructions on how to increase the size manually.
* `qemu-img`

## Installation

`kube-spawn` should run well on a modern Linux system (for example Fedora 27 or
Debian testing). If you want to test it in a controlled environment, you can
use [Vagrant](doc/vagrant.md).

To install `kube-spawn` on your machine, download a single binary release
or [build from source](#building).

kube-spawn uses CNI to setup networking for its containers. For that, you need
to download the CNI plugins (v.0.6.0 or later) from GitHub.

Example:

```
cd /tmp
curl -fsSL -O https://github.com/containernetworking/plugins/releases/download/v0.6.0/cni-plugins-amd64-v0.6.0.tgz
sudo mkdir -p /opt/cni/bin
sudo tar -C /opt/cni/bin -xvf cni-plugins-amd64-v0.6.0.tgz
```

By default, kube-spawn expects the plugins in `/opt/cni/bin`. The location
can be configured with `--cni-plugin-dir=` from the command line or
by setting `cni-plugin-dir: ...` in the configuration file.

Alternatively, you can use `go get` to fetch the plugins into your `GOPATH`:

```
go get -u github.com/containernetworking/plugins/plugins/...
```

## Quickstart

Create and start a 3 node cluster with the name "default":

```
sudo ./kube-spawn create
sudo ./kube-spawn start [--nodes 3]
```

Reminder: if the CNI plugins can't be found in `/opt/cni/bin`, you need
to pass `--cni-plugin-dir path/to/plugins`.

`create` prepares the cluster environment in `/var/lib/kube-spawn/clusters`.

`start` brings up the nodes and configures the cluster using
[kubeadm](https://github.com/kubernetes/kubeadm).

Shortly after, the cluster should be initialized:

```
[...]

Cluster "default" initialized
Export $KUBECONFIG as follows for kubectl:

        export KUBECONFIG=/var/lib/kube-spawn/clusters/default/admin.kubeconfig
```

After another 1-2 minutes the nodes should be ready:

```
export KUBECONFIG=/var/lib/kube-spawn/clusters/default/admin.kubeconfig
kubectl get nodes
NAME                          STATUS    ROLES     AGE       VERSION
kube-spawn-c1-master-q9fd4y   Ready     master    5m        v1.9.6
kube-spawn-c1-worker-dj7xou   Ready     <none>    4m        v1.9.6
kube-spawn-c1-worker-etbxnu   Ready     <none>    4m        v1.9.6
```

## Configuration

kube-spawn can be configured by command line flags, configuration file
(default `/etc/kube-spawn/config.yaml` or `--config path/to/config.yaml`),
environment variables or a mix thereof.

Example:

```
# /etc/kube-spawn/config.yaml
cni-plugin-dir: /home/user/code/go/bin
cluster-name: cluster1
container-runtime: rkt
rktlet-binary-path: /home/user/code/go/src/github.com/kubernetes-incubator/rktlet/bin/rktlet
```

## CNI plugins

kube-spawn supports weave, flannel, calico. It defaults to weave.

To configure with flannel:
```
kube-spawn create --pod-network-cidr 10.244.0.0/16 --cni-plugin flannel --kubernetes-version=v1.10.5
kube-spawn start --cni-plugin flannel --nodes 5
```

To configure with calico:
```
kube-spawn create --pod-network-cidr 192.168.0.0/16 --cni-plugin calico --kubernetes-version=v1.10.5
kube-spawn start --cni-plugin calico --nodes 5
```

To configure with canal:
```
kube-spawn create --pod-network-cidr 10.244.0.0/16 --cni-plugin canal --kubernetes-version=v1.10.5
kube-spawn start --cni-plugin canal --nodes 5
```

## Accessing kube-spawn nodes

All nodes can be seen with `machinectl list`. `machinectl shell` can be
used to access a node, for example:

```
sudo machinectl shell kube-spawn-c1-master-fubo3j
```

The password is `root`.

## Documentation

See [doc/](doc/)

## Building

To build kube-spawn in a Docker build container, simply run:

```
make
```

Optionally, install kube-spawn under a system directory:

```
sudo make install
```

`PREFIX` can be set to override the default target `/usr`.

## Troubleshooting

See [doc/troubleshooting](doc/troubleshooting.md)

## Community

Discuss the project on [Slack](https://kubernetes.slack.com/messages/C9ZMJH2NL/).
