# kubeadm-systemd

## Introduction

__kubeadm-systemd__ is a tool for creating a multi-node Kubernetes cluster
on a single machine, created mostly for developers __of__ Kubernetes.

It aims to be as similar to the solutions recommendend for production
clusters as possible.

## Getting started

### Build proper systemd-nspawn version

Unfortunately, there is [one pending feature](https://github.com/systemd/systemd/pull/4395)
of systemd-nspawn which is merged in master, but not released yet.
You will need to build your own systemd-nspawn binary.

We higly recommend using CoreOS' fork which backported that feature
to the 231 version of systemd (which is the one that Fedora and
the other popular distributions are using in its stable releases).

In order to do that, please use the following commands:

```
git clone git@github.com:coreos/systemd.git
cd systemd
git checkout v231
./autogen.sh
./configure
make
```

You **shoudn't** do `make install` after that! Using the custom
systemd-nspawn binary with the other components of systemd being
in another version is totally fine.

You may try to use master branch from upstream systemd repository, but
it's very risky!

### Get needed Kubernetes repositories

kubeadm-systemd needs the following repos to exist in your GOPATH:

* [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes)
* [kubernetes/release](https://github.com/kubernetes/release)

Also, bulding Kubernetes may rely on having your own fork and the
separate remote called `upstream`. In this HOWTO, we assume that
you have these repositories forked.

You can clone then by the following commands:

```
mkdir -p $GOPATH/src/k8s.io
cd $GOPATH/src/k8s.io
git clone git@github.com:<your_username>/kubernetes.git
git clone git@github.com:<your_username>/release.git
cd kubernetes
git remote add upstream git@github.com:kubernetes/kubernetes.git
cd ../release
git remote add upstream git@github.com:kubernetes/release.git
```

### Build Kubernetes

kubeadm-systemd needs the built Kubernetes binaries and hyperkube
Docker image. You need to build them like that:

```
cd $GOPATH/src/k8s.io/kubernetes
build/run.sh make
build/copy-output.sh
cd cluster/images/hyperkube
make VERSION=latest
```

### Build CNI with plugins

Firstly, you need to get a repo of CNI:

```
mkdir -p $GOPATH/src/github.com/containernetworking
cd $GOPATH/src/github.com/containernetworking
git clone git@github.com:containernetworking/cni.git
```

Then you can build CNI by doing:

```
cd cni
./build.sh
```

And then configure CNI networks needed by kubeadm-systemd:

```
mkdir -p /etc/cni/net.d
cat >/etc/cni/net.d/10-mynet.conf <<EOF
{
    "cniVersion": "0.2.0",
    "name": "mynet",
    "type": "bridge",
    "bridge": "cni0",
    "isGateway": true,
    "ipMasq": true,
    "ipam": {
        "type": "host-local",
        "subnet": "10.22.0.0/16",
        "routes": [
            { "dst": "0.0.0.0/0" }
        ]
    }
}
EOF
cat >/etc/cni/net.d/99-loopback.conf <<EOF
{
    "cniVersion": "0.2.0",
    "type": "loopback"
}
EOF
```

### Build and run kubeadm-systemd

In the directory where you cloned this repository, please do:

```
make
sudo GOPATH=$GOPATH SYSTEMD_NSPAWN_PATH=<path_to_your_nspawn_binary> ./kubeadm-systemd --nodes <number_of_nodes>
```

Sometimes when you Docker doesn't use the newest existing API, you may see
the following error:

```
2017/01/26 16:41:38 Error when pushing image: Error response from daemon: client is newer than server (client API version: 1.26, server API version: 1.24)
```

Then wou will need to include your Docker API version in DOCKER_API_VERSION
environment variable:

```
sudo GOPATH=$GOPATH SYSTEMD_NSPAWN_PATH=<path_to_your_nspawn_binary> DOCKER_API_VERSION=1.24 ./kubeadm-systemd --nodes <number_of_nodes>
```

The process of setting up the containers and running kubeadm
in them will be very long! But finally, the last lines of the
output should look like:

```
2017/01/26 16:13:13 Initializing master
[kubeadm] WARNING: kubeadm is in alpha, please do not use it for production clusters.
[init] Using Kubernetes version: v1.6.0-alpha.0
[init] Using Authorization mode: AlwaysAllow
[init] A token has not been provided, generating one
[preflight] Skipping pre-flight checks
[preflight] Starting the kubelet service
[certificates] Generated CA certificate and key.
[certificates] Generated API server certificate and key.
[certificates] Generated API server kubelet client certificate and key.
[certificates] Valid certificates and keys now exist in "/etc/kubernetes/pki"
[kubeconfig] Wrote KubeConfig file to disk: "/etc/kubernetes/admin.conf"
[kubeconfig] Wrote KubeConfig file to disk: "/etc/kubernetes/kubelet.conf"
[apiclient] Created API client, waiting for the control plane to become ready
[apiclient] All control plane components are healthy after 57.825192 seconds
[apiclient] Waiting for at least one node to register and become ready
[apiclient] First node is ready after 0.507054 seconds
[apiclient] Creating a test deployment
[apiclient] Test deployment succeeded
[token-discovery] Using token: a85f41:7a45e1cac2e6b9a3
[token-discovery] Created the kube-discovery deployment, waiting for it to become ready
[token-discovery] kube-discovery is ready after 30.005429 seconds
[addons] Created essential addon: kube-proxy
[addons] Created essential addon: kube-dns

Your Kubernetes master has initialized successfully!

You should now deploy a pod network to the cluster.
Run "kubectl apply -f [podnetwork].yaml" with one of the options listed at:
    http://kubernetes.io/docs/admin/addons/

You can now join any number of machines by running the following on each node:

kubeadm join --discovery token://a85f41:7a45e1cac2e6b9a3@10.22.0.206:9898
2017/01/26 16:14:49 Joining node
[kubeadm] WARNING: kubeadm is in alpha, please do not use it for production clusters.
[preflight] Skipping pre-flight checks
[preflight] Starting the kubelet service
[discovery] Created cluster info discovery client, requesting info from "http://10.22.0.206:9898/cluster-info/v1/?token-id=a85f41"
[discovery] Cluster info object received, verifying signature using given token
[discovery] Cluster info signature and contents are valid, will use API endpoints [https://10.22.0.206:6443]
[bootstrap] Trying to connect to endpoint https://10.22.0.206:6443
[bootstrap] Detected server version: v1.2.0-alpha.7.18692+28c439dcfb0383-dirty
[bootstrap] Successfully established connection with endpoint "https://10.22.0.206:6443"
[csr] Created API client to obtain unique certificate for this node, generating keys and certificate signing request
[csr] Received signed certificate from the API server[csr] Generating kubelet configuration
[kubeconfig] Wrote KubeConfig file to disk: "/etc/kubernetes/kubelet.conf"

Node join complete:
* Certificate signing request sent to master and response
  received.
* Kubelet informed of new secure connection details.

Run 'kubectl get nodes' on the master to see this machine join.
2017/01/26 16:14:54 Joining node
[kubeadm] WARNING: kubeadm is in alpha, please do not use it for production clusters.
[preflight] Skipping pre-flight checks
[preflight] Starting the kubelet service
[discovery] Created cluster info discovery client, requesting info from "http://10.22.0.206:9898/cluster-info/v1/?token-id=a85f41"
[discovery] Cluster info object received, verifying signature using given token
[discovery] Cluster info signature and contents are valid, will use API endpoints [https://10.22.0.206:6443]
[bootstrap] Trying to connect to endpoint https://10.22.0.206:6443
[bootstrap] Detected server version: v1.2.0-alpha.7.18692+28c439dcfb0383-dirty
[bootstrap] Successfully established connection with endpoint "https://10.22.0.206:6443"
[csr] Created API client to obtain unique certificate for this node, generating keys and certificate signing request
[csr] Received signed certificate from the API server[csr] Generating kubelet configuration
[kubeconfig] Wrote KubeConfig file to disk: "/etc/kubernetes/kubelet.conf"

Node join complete:
* Certificate signing request sent to master and response
  received.
* Kubelet informed of new secure connection details.

Run 'kubectl get nodes' on the master to see this machine join.
```

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

The integration between CNI and systemd-nspawn is made by a binary
called __cnispawn__ which creates a network namespace, executes a CNI
plugin on that, and then runs systemd-nspawn inside that namespace.
By default, systemd-nspawn doesn't create its own network namespaces,
so the container is succesfully running inside the namespace we
created.

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
  for developers. But still, it uses its own way of creating the
  cluster. And also, in our opinion, Docker isn't very good tool
  for simulating the nodes, since it's an application container
  engine, not an operaring system container engine (like
  systemd-nspawn).
* [kubeadm-dind-cluster](https://github.com/Mirantis/kubeadm-dind-cluster) -
  it does a great job with using kubeadm, but still, it uses Docker
  instead of any other container engine which is designed for
  containerizing the whole OS, not an application. Also, we prefer
  to maintain code for doing such complicated things, instead of
  huge shell scripts.
