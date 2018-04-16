## Testing `kube-spawn` with [Vagrant](https://www.vagrantup.com/)

### With script

There is a script called `vagrant-all.sh` which does the following things:

- sets up the Vagrant VM (`vagrant up`)
- automatically builds the project during provisioning
- runs the Kubernetes cluster
- redirects traffic from 6443 port on VM to the container with k8s API
- copies the `kubeconfig` to host

If you are behind a proxy,
```
$ vagrant plugin install vagrant-proxyconf
```

If you are using libvirt instead of virtualbox
```
export KUBESPAWN_PROVIDER=libvirt
```

and then

```
$ ./vagrant-all.sh
```

Then you can use the Kubernetes cluster both from inside the VM:

```
$ vagrant ssh
$ kubectl get nodes
NAME                STATUS    AGE       VERSION
kubespawndefault0   Ready     12m       v1.7.5
kubespawndefault1   Ready     11m       v1.7.5
```

or from host, if you have `kubectl` installed on it:

```
$ kubectl get nodes
NAME                STATUS    AGE       VERSION
kubespawndefault0   Ready     12m       v1.7.5
kubespawndefault1   Ready     11m       v1.7.5
```

### Manually

The provided Vagrantfile is used to test `kube-spawn` on various Linux distributions.

```
$ vagrant up
$ vagrant ssh
$ ./build.sh    # sets up environment, runs build and setup/init command
```

You can set the following environment variables:

- `KUBESPAWN_AUTOBUILD` - runs `build.sh` script (which builds kube-spawn) during machine
  provisioning
- `KUBESPAWN_REDIRECT_TRAFFIC` - redirects traffic from 6443 port on VM to the container
  with k8s API
