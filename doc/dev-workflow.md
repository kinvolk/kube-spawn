# Kubernetes development workflow example

This article describes a step-by-step workflow that a Kubernetes developer might follow when testing a Kubernetes patch with kube-spawn.

For the purpose of the article, we will write a new [admission controller](https://kubernetes.io/docs/admin/admission-controllers/) named `DenyAttach` that inconditionally denies all attaching to a container. The end result will be:

```bash
$ kubectl attach mypod-74c9fd65cb-n5hsg
If you don't see a command prompt, try pressing enter.
Error from server (Forbidden): pods "mypod-74c9fd65cb-n5hsg" is forbidden: cannot attach to a container, rejected by admission controller
```

The implementation of `DenyAttach` will be reusing code from the existing admission controller [DenyEscalatingExec](https://kubernetes.io/docs/admin/admission-controllers/#denyescalatingexec).

## Compiling locally

We will first fetch the [patch](https://github.com/kinvolk/kubernetes/commit/c117bd71672b2da7c7777cddf0287b07d29b90e5).

```bash
$ cd $GOPATH/src/k8s.io/kubernetes

# Add git kinvolk remote if not already done
$ git remote |grep -q kinvolk || git remote add kinvolk https://github.com/kinvolk/kubernetes

# Fetch the branch
$ git pull kinvolk alban/v1.8.5-beta.0-denyattach
$ git checkout kinvolk/alban/v1.8.5-beta.0-denyattach

# Build Kubernetes
$ build/run.sh make

# Build a Hyperkube Docker image with a tag
$ make -C cluster/images/hyperkube VERSION=v1.8.5-beta.0-denyattach

$ docker images | grep hyperkube-amd64
```

## Pushing the new hyperkube image to a registry of your choice

In this example, we will spawn a local registry:

```
docker run -d -p 5000:5000 --name registry registry:2
```

Tag the hyperkube image and push it, for example:

```
docker tag e0d598144aa3 127.0.0.1:5000/me/hyperkube-amd64:v1.8.5-beta.0-denyattach
docker push 127.0.0.1:5000/me/hyperkube-amd64:v1.8.5-beta.0-denyattach
```

## Deploying your build on kube-spawn

```
$ sudo ./kube-spawn create --kubernetes-source-dir $GOPATH/src/k8s.io/kubernetes --hyperkube-image 10.22.0.1:5000/me/hyperkube-amd64:v1.8.5-beta.0-denyattach -c denyattach
$ sudo ./kube-spawn start -c denyattach
```

Note that the registry IP address must be `10.22.0.1` here, which is the
address of the host `cni0` interface by kube-spawn.

Since the hyperkube image contains the API server, controller manager and
scheduler but not e.g. kubeadm, we also pass `--kubernetes-source-dir`
to point kube-spawn to the location from where to copy the necessary
binaries. If not given, kube-spawn would use the vanilla upstream version
(`--kubernetes-version` default).

Let's test if it works:

```
$ export KUBECONFIG=/var/lib/kube-spawn/clusters/denyattach/admin.kubeconfig
$ kubectl run mypod --image=busybox --command -- /bin/sh -c 'while true ; do sleep 1 ; date ; done'
...
$ kubectl get pods
NAME                     READY     STATUS    RESTARTS   AGE
mypod-74c9fd65cb-n9rfs   1/1       Running   0          11s
$ kubectl attach mypod-74c9fd65cb-n9rfs
If you don't see a command prompt, try pressing enter.
Error from server (Forbidden): pods "mypod-74c9fd65cb-n9rfs" is forbidden: cannot attach to a container, rejected by admission controller
```

## Testing a different DenyAttach admission controller

Someone might not like the error message, saying "rejected by admission controller":
Kubernetes has plenty of admission controllers and it does not say which one rejected the request.

Luckily, a colleague fixed that already. Let's test her patch:

```
$ sudo ./kube-spawn create --kubernetes-source-dir $GOPATH/src/k8s.io/kubernetes --hyperkube-image docker.io/kinvolk/hyperkube-amd64:v1.8.5-beta.0-denyattachfix -c denyattachfix
$ sudo ./kube-spawn start -c denyattachfix
```

```
$ export KUBECONFIG=/var/lib/kube-spawn/clusters/denyattachfix/admin.kubeconfig
$ kubectl run mypod --image=busybox --command -- /bin/sh -c 'while true ; do sleep 1 ; date ; done'
...
$ kubectl get pods
NAME                     READY     STATUS    RESTARTS   AGE
mypod-74c9fd65cb-gbrd9   1/1       Running   0          11s
$ kubectl attach mypod-74c9fd65cb-gbrd9
If you don't see a command prompt, try pressing enter.
Error from server (Forbidden): pods "mypod-74c9fd65cb-gbrd9" is forbidden: cannot attach to a container, rejected by DenyAttach
```
