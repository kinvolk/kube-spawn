package script

const DockerDaemonConfigPath = "/etc/docker/daemon.json"

const DockerDaemonConfig = `{
    "insecure-registries": ["10.22.0.1:5000"],
    "default-runtime": "custom",
    "runtimes": {
        "custom": { "path": "/usr/bin/kube-spawn-runc" }
    },
    "storage-driver": "overlay2"
}
`
