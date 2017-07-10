## Testing `kube-spawn` with [Vagrant](https://www.vagrantup.com/)

The provided Vagrantfile is used to test `kube-spawn` on various Linux distributions.

```
$ vagrant up
$ vagrant ssh
$ ./build.sh    # sets up environment, runs build and up/init command
```

To run them manually:

```
cd $GOPATH/src/github.com/kinvolk/kubeadm-nspawn

go get -u github.com/containernetworking/plugins/plugins/main/bridge
go get -u github.com/containernetworking/plugins/plugins/ipam/host-local

make vendor all

sudo machinectl pull-raw --verify=no https://alpha.release.core-os.net/amd64-usr/current/coreos_developer_container.bin.bz2 coreos

sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kubeadm-nspawn up --nodes 2 --image coreos
sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kubeadm-nspawn init
```
