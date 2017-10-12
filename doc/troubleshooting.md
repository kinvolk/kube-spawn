Here are some common issues that we encountered and how we work around or
fix the. If you discover more, please create an issue or submit a PR.

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

You can build systemd-nspawn yourself and include these patches:

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

## Debugging `kube-spawn-runc`

see [here](../cmd/kube-spawn-runc/README.md)
