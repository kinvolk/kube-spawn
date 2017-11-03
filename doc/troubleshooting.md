Here are some common issues that we encountered and how we work around or
fix them. If you discover more, please create an issue or submit a PR.

## Missing GOPATH environment variable

`kube-spawn` is able to check for environment variables, such as `$GOPATH` and `$CNI_PATH`, which are necessary for launching nspawn containers as well as creating network namespaces for CNI. If `$GOPATH` is unavailable, it tries to fall back to `$HOME/go`. If `$CNI_PATH` is unavailable, it tries to fall back to `$GOPATH/bin`. Doing so, `kube-spawn` should be able to figure out most probable paths.

If any path-related problem still occurs, please try the following approaches:

* try to run with `sudo -E kube-spawn ...` to pass normal-user's env variables
* try to run with pre-defined `GOPATH` or `CNI_PATH`, for example `sudo GOPATH=/home/myuser/go CNI_PATH=/home/myuser/go/bin kube-spawn ...`

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

## Debugging `kube-spawn-runc`

see [here](../cmd/kube-spawn-runc/README.md)
