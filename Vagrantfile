# -*- mode: ruby -*-
# vi: set ft=ruby sw=2 ts=2 :

ENV["TERM"] = "xterm-256color"
ENV["LC_ALL"] = "en_US.UTF-8"

Vagrant.configure("2") do |config|
  config.vm.box = "jhcook/fedora26" # defaults to fedora

  # common parts
  if Vagrant.has_plugin?("vagrant-vbguest")
    config.vbguest.auto_update = false
  end
  config.vm.provider :virtualbox do |vb|
    vb.check_guest_additions = false
    vb.functional_vboxsf = false
    vb.customize ["modifyvm", :id, "--memory", "4096"]
    vb.customize ["modifyvm", :id, "--cpus", "2"]
  end

  # Fedora 26
  config.vm.define "fedora", primary: true do |fedora|
    config.vm.provision "shell", inline: "dnf install -y btrfs-progs docker git go kubernetes qemu-img strace tmux"

    config.vm.synced_folder ".", "/vagrant", disabled: true
    config.vm.synced_folder ".", "/home/vagrant/go/src/github.com/kinvolk/kube-spawn",
      create: true,
      owner: "vagrant",
      group: "vagrant",
      type: "rsync"

    # NOTE: chown is explicitly needed, even when synced_folder is configured
    # with correct owner/group. Maybe a vagrant issue?
    config.vm.provision "shell", inline: "mkdir -p /home/vagrant/go ; chown -R vagrant:vagrant /home/vagrant/go"

    config.vm.provision "shell", env: {"GOPATH" => "/home/vagrant/go"}, privileged: false, path: "scripts/vagrant-setup-env.sh"
    config.vm.provision "shell", env: {"VUSER" => "vagrant"}, path: "scripts/vagrant-mod-env.sh"
  end

  # Ubuntu 17.04 (Zesty)
  config.vm.define "ubuntu", autostart: false do |ubuntu|
    config.vm.box = "ubuntu/zesty64"
    config.vm.provision "shell", inline: "curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -; echo \"deb http://apt.kubernetes.io/ kubernetes-xenial main\" > /etc/apt/sources.list.d/kubernetes.list; apt-get update; DEBIAN_FRONTEND=noninteractive apt-get install -y docker.io golang git qemu-utils selinux-utils systemd-container kubectl tmux"

    config.vm.synced_folder ".", "/vagrant", disabled: true
    config.vm.synced_folder ".", "/home/ubuntu/go/src/github.com/kinvolk/kube-spawn",
      create: true,
      owner: "ubuntu",
      group: "ubuntu",
      type: "rsync"

    config.vm.provision "shell", inline: "mkdir -p /home/ubuntu/go ; chown -R ubuntu:ubuntu /home/ubuntu/go"
    config.vm.provision "shell", env: {"GOPATH" => "/home/ubuntu/go"}, privileged: false, path: "scripts/vagrant-setup-env.sh"
    config.vm.provision "shell", env: {"VUSER" => "ubuntu"}, path: "scripts/vagrant-mod-env.sh"
  end

  # Debian testing
  config.vm.define "debian", autostart: false do |debian|
    config.vm.box = "debian/testing64"
    config.vm.provision "shell", inline: "echo deb http://apt.dockerproject.org/repo debian-stretch main > /etc/apt/sources.list.d/docker.list; apt-get update; DEBIAN_FRONTEND=noninteractive apt-get install -y --allow-unauthenticated docker-engine; DEBIAN_FRONTEND=noninteractive apt-get install -y golang git kubernetes-client qemu-utils selinux-utils systemd-container tmux"

    config.vm.synced_folder ".", "/vagrant", disabled: true
    config.vm.synced_folder ".", "/home/vagrant/go/src/github.com/kinvolk/kube-spawn",
      create: true,
      owner: "vagrant",
      group: "vagrant",
      type: "rsync"

    config.vm.provision "shell", inline: "mkdir -p /home/vagrant/go ; chown -R vagrant:vagrant /home/vagrant/go"
    config.vm.provision "shell", env: {"GOPATH" => "/home/vagrant/go"}, privileged: false, path: "scripts/vagrant-setup-env.sh"
    config.vm.provision "shell", env: {"VUSER" => "vagrant"}, path: "scripts/vagrant-mod-env.sh"
  end
end
