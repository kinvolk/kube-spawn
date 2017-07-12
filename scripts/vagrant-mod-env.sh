#!/bin/bash

set -euo pipefail

USER=vagrant
HOME=/home/${USER}

echo 'Modifying environment'
chown -R ${USER}:${USER} ${HOME}
chmod +x ${HOME}/build.sh
setenforce 0
systemctl stop firewalld
sudo groupadd docker && sudo gpasswd -a ${USER} docker && sudo systemctl restart docker && newgrp docker
usermod -aG docker ${USER}
