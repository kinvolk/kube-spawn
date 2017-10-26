# Run Kubernetes with rkt container runtime

[`rktlet` repository](https://github.com/kubernetes-incubator/rktlet)

kube-spawn supports spawning a cluster with rkt as the container runtime by
setting `--container-runtime=rkt` to the `create` command.

The necessary binaries are detected from the host systems PATH or can be provided via environment variables:

```
$ sudo -E \
  KUBE_SPAWN_RKT_BIN=/path/to/rkt \
  KUBE_SPAWN_RKT_STAGE1_IMAGE=/path/to/stage1 \
  KUBE_SPAWN_RKTLET_BIN=/path/to/rktlet \
  ./kube-spawn create --nodes=3 --container-runtime=rkt
```

### Notes:

* rkt needs `stage1-coreos.aci`, we default to `/usr/lib/rkt/stage1-images/stage1-coreos.aci`, if this location differs
  please use the environment variable above
