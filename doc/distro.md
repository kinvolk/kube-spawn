## How to set up kube-spawn on various Linux distros

### Common

First of all, to be able to run kube-spawn, you need to make sure that the
following things are done on your system, no matter which distro you run on.

* make sure that systemd v233 or newer is installed
* set SELinux mode to either `Permissive` or `Disabled`, e.g.: `sudo setenforce 0`
* set env variables correctly
  * set `GOPATH` correctly, e.g. `export GOPATH=$HOME/go`
  * set `KUBECONFIG`: `export KUBECONFIG=/var/lib/kube-spawn/clusters/default/admin.kubeconfig`

### Fedora

Fedora 26 or newer is needed, mainly for systemd.
kube-spawn works fine on Fedora as long as the following dependencies are installed.

#### install required packages

```
sudo dnf install -y btrfs-progs git go iptables libselinux-utils polkit qemu-img systemd-container
```

### Ubuntu

Ubuntu 17.10 (Artful) or newer is needed, mainly for systemd.

#### install required packages

```
sudo apt-get install -y btrfs-progs git golang iptables policykit-1 qemu-utils selinux-utils systemd-container
```

#### systemd-resolved

On Ubuntu 17.10, systemd-resolved is enabled by default, as well as its stub
listener, which listens on 127.0.0.53:53 for the local DNS resolution.
Unfortunately, systemd v234, the default version on Ubuntu 17.10, has bugs
regarding DNS resolution. Thus we need to disable systemd-resolved on the host,
which will make systemd-resolved disabled inside nspawn containers as well.
So it's recommended to run the following commands on the host, before starting
kube-spawn clusters.

```
sudo sed -i -e 's/^#*.*DNSStubListener=.*$/DNSStubListener=no/' /etc/systemd/resolved.conf
sudo sed -i -e 's/nameserver 127.0.0.53/nameserver 8.8.8.8/' /etc/resolv.conf
systemctl is-active systemd-resolved >& /dev/null && sudo systemctl stop systemd-resolved
systemctl is-enabled systemd-resolved >& /dev/null && sudo systemctl disable systemd-resolved
```

### Debian

Normally it should be similar to Ubuntu.

### openSUSE Kubic

All versions of openSUSE Kubic should work. 

#### install required packages

```
transactional-update pkg install kubernetes-client systemd-container cni-plugins
systemctl reboot
```

#### CNI plugins

openSUSE has a cni-plugin RPM. If this should be used, CNI_PATH
has to be set:

```
export CNI_PATH=/usr/lib/cni
```

