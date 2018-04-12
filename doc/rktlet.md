# Run Kubernetes with rkt as container runtime

kube-spawn supports rktlet, the rkt implementation of the Kubernetes Container
Runtime Interface: https://github.com/kubernetes-incubator/rktlet)

To use rkt as the container runtime, set `--container-runtime=rkt` with the
`create` command. rkt and rktlet must be available on the host.

Example:

```
sudo ./kube-spawn create --container-runtime rkt --rktlet-binary-path ~/code/go/src/github.com/kubernetes-incubator/rktlet/bin/rktlet -c rktcluster
sudo ./kube-spawn start -c rktcluster -n 5
...
export KUBECONFIG=/var/lib/kube-spawn/clusters/rktcluster/admin.kubeconfig
kubectl get nodes -o wide
NAME                                  STATUS    ROLES     AGE       VERSION   EXTERNAL-IP   OS-IMAGE                                       KERNEL-VERSION   CONTAINER-RUNTIME
kube-spawn-rktcluster-master-yomfri   Ready     master    1m        v1.9.6    <none>        Flatcar Linux by Kinvolk 1729.0.0 (Rhyolite)   4.15.0-2-amd64   rkt://0.1.0
kube-spawn-rktcluster-worker-4u9fsu   Ready     <none>    41s       v1.9.6    <none>        Flatcar Linux by Kinvolk 1729.0.0 (Rhyolite)   4.15.0-2-amd64   rkt://0.1.0
kube-spawn-rktcluster-worker-mysslr   Ready     <none>    41s       v1.9.6    <none>        Flatcar Linux by Kinvolk 1729.0.0 (Rhyolite)   4.15.0-2-amd64   rkt://0.1.0
kube-spawn-rktcluster-worker-ogrm8l   Ready     <none>    40s       v1.9.6    <none>        Flatcar Linux by Kinvolk 1729.0.0 (Rhyolite)   4.15.0-2-amd64   rkt://0.1.0
kube-spawn-rktcluster-worker-yxspu2   Ready     <none>    41s       v1.9.6    <none>        Flatcar Linux by Kinvolk 1729.0.0 (Rhyolite)   4.15.0-2-amd64   rkt://0.1.0
```

`--rkt-binary-path` and `--rkt-stage1-image-path` can be used to specify
non-default location for the rkt / stage1 binaries.

NB: rktlet doesn't support Kubernetes 1.10 at the time of writing, see
https://github.com/kubernetes-incubator/rktlet/issues/183
