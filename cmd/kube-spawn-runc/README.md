# kube-spawn-runc

`kube-spawn-runc` is a wrapper around `runc` to add the `--no-new-keyring` flag on `run` and `create` commands.

To use the wrapper create a custom runtime in `/etc/docker/daemon.json` and activate it as the default-runtime.
For how to do this refer to [this example](https://github.com/kinvolk/kube-spawn/blob/master/etc/daemon.json#L3-L6).

## Debugging

You can set the following environment variables on this wrapper:

- `KUBE_SPAWN_RUNC_BINARY_PATH` : path to runc binary. By default we find `docker-runc` in PATH
- `KUBE_SPAWN_RUNC_LOG_PATH` : path to the log file. By default there are no logs

To run with an environment variable:

`systemctl edit containerd.service`

now add:

```
[Service]
Environment=KUBE_SPAWN_RUNC_LOG_PATH=...
Environment=KUBE_SPAWN_RUNC_BINARY_PATH=...
```
