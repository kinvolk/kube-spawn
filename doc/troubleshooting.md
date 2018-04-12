# Troubleshooting

Here are some common issues that we encountered and how we work around or
fix them. If you discover more, please create an issue or submit a PR.

- [SELinux](#selinux)
- [Restarting machines fails without removing machine images](#restarting-machines-fails-without-removing-machine-images)
- [Running on a version of systemd \< 233](#running-on-a-version-of-systemd--233)
- [kubeadm init looks like it is hanging](#kubeadm-init-looks-like-it-is-hanging)
- [Inotify problems with many nodes](#inotify-problems-with-many-nodes)
- [Issues with ISPs hijacking DNS requests](#issues-with-isps-hijacking-dns-requests)

## SELinux

To run `kube-spawn`, it is recommended to turn off SELinux enforcing mode:

```
$ sudo setenforce 0
```

However, it is also true that disabling security framework is not always desirable. So it is planned to handle security policy instead of disabling them. Until then, there's no easy way to get around.

## Restarting machines fails without removing machine images

If the `start` command fails, make sure to remove all created images
(`machinectl remove ...`) before trying again.

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

## kubeadm init looks like it is hanging

Usually it takes 1-3 minutes until `kubeadm init` initialized the
cluster nodes and finished bootstrapping on the master node. While waiting
for it to finish, it shows a message like:

```
[init] This often takes around a minute; or longer if the control plane images have to be pulled.
```

kubeadm does not give many hints to users. Possible reasons are:

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
