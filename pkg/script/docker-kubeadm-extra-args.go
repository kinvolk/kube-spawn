package script

const DockerKubeadmExtraArgsPath = "/etc/systemd/system/docker.service.d/20-kubeadm-extra-args.conf"

const DockerKubeadmExtraArgs = `[Service]
Environment="DOCKER_OPTS=--exec-opt native.cgroupdriver=cgroupfs"
`
