# Troubleshooting

Here are some common issues that we encountered and how we work around or
fix them. If you discover more, please create an issue or submit a PR.

- [Missing GOPATH environment variable](#missing-gopath-environment-variable)
- [SELinux](#selinux)
- [Restarting machines fails without removing machine images](#restarting-machines-fails-without-removing-machine-images)
- [Running on a version of systemd \< 233](#running-on-a-version-of-systemd--233)
- [Docker: Error when pushing image](#docker-error-when-pushing-image)
- [kubeadm init looks like it is hanging](#kubeadm-init-looks-like-it-is-hanging)
- [Running with Kubernetes 1.7.3 or newer](#running-with-kubernetes-173-or-newer)
- [Getting the Kubernetes repositories](#getting-the-kubernetes-repositories)
- [Setting up an insecure registry](#setting-up-an-insecure-registry)
- [Inotify problems with many nodes](#inotify-problems-with-many-nodes)
- [Issues with ISPs hijacking DNS requests](#issues-with-isps-hijacking-dns-requests)

## SELinux

To run `kube-spawn`, it is recommended to turn off SELinux enforcing mode:

```
$ sudo setenforce 0
```

However, it is also true that disabling security framework is not always desirable. So it is planned to handle security policy instead of disabling them. Until then, there's no easy way to get around.

## Restarting machines fails without removing machine images

If your `start` command fails upon restarting machines without any reason, please try to removing existing images like:

```
$ for i in $(seq 0 2); do sudo machinectl remove kubespawn$i; done
```

That could make the setup process do the job again. Ideally the remaining images should be handled automatically. For that it is planned to implement storing node infos persistently. (See https://github.com/kinvolk/kube-spawn/issues/37)

## Running on a version of systemd < 233

You can build `systemd-nspawn` yourself and include these patches:

* `SYSTEMD_NSPAWN_USE_CGNS` https://github.com/systemd/systemd/pull/3809
* `SYSTEMD_NSPAWN_MOUNT_RW` and `SYSTEMD_NSPAWN_USE_NETNS` https://github.com/systemd/systemd/pull/4395

We highly recommend using CoreOS' fork which backported that feature
to the 231 version of systemd (which is the one that Fedora and
the other popular distributions are using in its stable releases).

In order to do that, please use the following commands:

```
$ git clone git@github.com:coreos/systemd.git
$ cd systemd
$ git checkout v231
$ ./autogen.sh
$ ./configure
$ make
```

You **shouldn't** do `make install` after that! Using the custom
`systemd-nspawn` binary with the other components of systemd being
in another version is totally fine.

You may try to use master branch from upstream systemd repository, but we
don't encourage it.

You can pass `kube-spawn` an alternative `systemd-nspawn` binary by setting the
environment variable `SYSTEMD_NSPAWN_PATH` to where you have built your own.


## Docker: Error when pushing image

You may see the following error:
```
2017/01/26 16:41:38 Error when pushing image: Error response from daemon: client is newer than server (client API version: 1.26, server API version: 1.24)
```

Then you will need to include your Docker API version in an
environment variable: `DOCKER_API_VERSION=1.24 `

## kubeadm init looks like it is hanging

Usually it takes several minutes until `kubeadm init` initialized
cluster nodes and finished bootstrapping on the master node. While waiting
for it to finish, it shows the following message (in case of k8s 1.7):

```
[apiclient] Created API client, waiting for the control plane to become ready
```

Even when in these phase something fundamental is broken inside nspawn
containers, kubeadm does not give many hints to users. Possible reasons are:

* container runtime (docker or rktlet) is not running or running incorrectly
* kubelet is not running or running incorrectly
* any other fundamental errors like filesystem being full

So in that case, users should find out the underlying reasons by doing e.g.:

```
$ sudo machinectl shell kubespawn0
```

Then inside kubespawn0, do debugging like:

```
# systemctl status docker
# systemctl status kubelet
# journalctl -u docker -xe --no-pager | less
# journalctl -u kubelet -xe --no-pager | less
```

## Running with Kubernetes 1.7.3 or newer

When running kube-spawn with an Kubernetes release, by specifying a
command-line option `--kubernetes-version=`, you should specify
version 1.7.3 or newer. That's because of the following issues:

* token creation issue during cluster init, with k8s version 1.7.0 or older:
  - (https://github.com/kubernetes/kubeadm/issues/335)

* issue of mixing options `--skip-preflight-checks` and `--config=`, when running k8s version 1.7.2 or older:
  - (https://github.com/kubernetes/kubernetes/pull/49498)

## Getting the Kubernetes repositories

kube-spawn needs the following repos to exist in your GOPATH:

* [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes)
* [kubernetes/release](https://github.com/kubernetes/release)

Also, building Kubernetes may rely on having your own fork and the
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

## Setting up an insecure registry

To be able to run a self-built Kubernetes `dev` cluster, we need to be able to
start an insecure registry on localhost, before launching nodes with
`kube-spawn start`. In general, the IP address & port number of the registry
server need to be specified in `/etc/docker/daemon.json`, like:

```
    "insecure-registries": [
        "10.22.0.1:5000"
    ]
```

On some distros, however, the approach above might not work. For example on
Fedora, the insecure registry needs to be configured in
`/etc/sysconfig/docker`:

```
INSECURE_REGISTRY='--insecure-registry=10.22.0.1:5000'
```

Even when docker runs with the configs above, kube-spawn could fail with the
following message:

```
Error pushing hyperkube image: Error response from daemon: no such id: gcr.io/google-containers/hyperkube-amd64
```

In that case, try to rebuild a hyperkube image, as described in (https://github.com/kinvolk/kube-spawn/blob/master/README.md#run-local-kubernetes-builds).

If you see the error even after rebuilding the image, then the image ID might
be wrong, probably because the public registry `gcr.io` has changed its naming
scheme. As a workaround, you can override the name like this:

```
sudo docker tag gcr.io/google-containers/hyperkube-amd64 SOME_CORRECT_NAME
```

You could also see the message below.

```
Error pushing hyperkube image: Get http://10.22.0.1:5000/v2/: net/http: request canceled while waiting for connection (Client.Timeout exceeded while awaiting headers)
```

In that case, there are several approaches you could try out.

* Check if the network interface `cni0` is available, by running `ip link | grep cni0`.
* Check if the port 5000 is open, by running `ss | grep 5000`.

## Inotify problems with many nodes

Running a big amount of nodes (many-node clusters or many clusters) can cause inotify limits to be reached, making new nodes fail to start.

The symptom is a message like this in the kubelet logs on nodes with the `NotReady` state:

```
Failed to start cAdvisor inotify_add_watch /sys/fs/cgroup/blkio/machine.slice/machine-kubespawndefault0.scope/system.slice/var-lib-docker-overlay2-0646d006ef5cf6c4d61c1ad51f958d0891d184ba70a2816d30462175a80beeaa-merged.mount: no space left on device
```

To increase inotify limits you can use the sysctl tool on the host:

```
# sysctl fs.inotify.max_user_watches=524288
# sysctl fs.inotify.max_user_instances=8192
```

## Issues with ISPs hijacking DNS requests

Some ISPs use [DNS
hijacking](https://en.wikipedia.org/wiki/DNS_hijacking#Manipulation_by_ISPs),
violating the DNS protocol. Please check if your DNS server correctly returns the `NXDOMAIN` error on non-existent domains:

```
$ host non-existent-domain-name-7932432687432.com
Host non-existent-domain-name-7932432687432.com not found: 3(NXDOMAIN)
```

If it's not the case, the kube-dns pod might not start correctly or might be very slow:

```
$ kubectl get pods --all-namespaces
NAMESPACE     NAME                                        READY     STATUS              RESTARTS   AGE
kube-system   kube-dns-2425271678-t7mrw                   0/3       ContainerCreating   0          5m
```

To fix this issue, please specify valid DNS servers on the host. Example:
```
$ cat /etc/resolv.conf
nameserver 8.8.8.8
```
