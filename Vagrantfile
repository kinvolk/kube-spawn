# -*- mode: ruby -*-
# vi: set ft=ruby :

ENV["TERM"] = "xterm-256color"
ENV["LC_ALL"] = "en_US.UTF-8"

Vagrant.configure("2") do |config|
  config.vm.box = "jhcook/fedora26"
  config.vm.provision "shell", inline: "dnf install -y btrfs-progs docker git go kubernetes strace tmux"
  # config.vm.box = "ubuntu/zesty64"
  # config.vm.provision "shell", inline: "DEBIAN_FRONTEND=noninteractive apt-get install -y golang git docker.io systemd-container tmux"

  config.vm.synced_folder ".", "/vagrant", disabled: true
  config.vm.synced_folder ".", "/home/vagrant/go/src/github.com/kinvolk/kube-spawn", create: true, type: "rsync"

  config.vbguest.auto_update = false
  config.vm.provider :virtualbox do |vb|
      vb.check_guest_additions = false
      vb.functional_vboxsf = false
      vb.customize ["modifyvm", :id, "--memory", "4094"]
      vb.customize ["modifyvm", :id, "--cpus", "1"]
  end

  config.vm.provision "shell", env: {"GOPATH" => "/home/vagrant/go"}, privileged: false, inline: <<HERE
echo 'Setting up correct env. variables'
echo "export GOPATH=$GOPATH" >> /home/vagrant/.bash_profile
echo "export PATH=$PATH:$GOPATH/bin:/usr/local/go/bin" >> /home/vagrant/.bash_profile

echo 'Writing build.sh'
source ~/.bash_profile
if [[ ! -f /home/vagrant/build.sh ]]; then
cat >>/home/vagrant/build.sh <<-EOF
#!/bin/bash
set -xe

cd $GOPATH/src/github.com/kinvolk/kube-spawn

go get -u github.com/containernetworking/plugins/plugins/main/bridge
go get -u github.com/containernetworking/plugins/plugins/ipam/host-local

make vendor all

sudo machinectl show-image coreos || sudo machinectl pull-raw --verify=no https://alpha.release.core-os.net/amd64-usr/current/coreos_developer_container.bin.bz2 coreos

sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kube-spawn --kubernetes-version=1.6.6 up --nodes 2 --image coreos
sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kube-spawn --kubernetes-version=1.6.6 init
EOF
fi
HERE

  config.vm.provision "shell", inline: <<HERE
echo 'Modifying environment'
chown -R vagrant:vagrant /home/vagrant
chmod +x /home/vagrant/build.sh
setenforce 0
systemctl stop firewalld
sudo groupadd docker && sudo gpasswd -a ${USER} docker && sudo systemctl restart docker && newgrp docker
usermod -aG docker vagrant
HERE
end
