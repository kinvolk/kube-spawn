# -*- mode: ruby -*-
# vi: set ft=ruby :

ENV["TERM"] = "xterm-256color"
ENV["GOPATH"] = "/home/ubuntu/go"
ENV["LC_ALL"] = "en_US.UTF-8"

Vagrant.configure("2") do |config|
  # config.vm.box = "fedora/25-cloud-base"
  # config.vm.provision "shell", inline: "dnf install -y go git docker"
  config.vm.box = "ubuntu/zesty64"
  config.vm.provision "shell", inline: "DEBIAN_FRONTEND=noninteractive apt-get install -y golang git docker.io systemd-container tmux"
  config.vm.provision "shell", inline: "usermod -aG docker ubuntu"

  # config.vbguest.auto_update = true

  config.vm.synced_folder ".", "/vagrant", disabled: true
  config.vm.synced_folder ".", "/home/ubuntu/go/src/github.com/kinvolk/kubeadm-nspawn", create: true
  # config.vm.provider :virtualbox do |vbox|
  #     vbox.check_guest_additions = false
  #     vbox.functional_vboxsf = false
  # end

  config.vm.provider :virtualbox do |vb|
      vb.customize ["modifyvm", :id, "--memory", "4094"]
      vb.customize ["modifyvm", :id, "--cpus", "1"]
  end

  config.vm.provision "shell", privileged: false, inline: <<HERE
echo 'Setting up correct env. variables'
echo "export GOPATH=$GOPATH" >> /home/ubuntu/.bash_profile
echo "export PATH=$PATH:$GOPATH/bin:/usr/local/go/bin" >> /home/ubuntu/.bash_profile
HERE

  config.vm.provision "shell", inline: "chown -R ubuntu:ubuntu /home/ubuntu"
end
