# -*- mode: ruby -*-
# vi: set ft=ruby sw=2 ts=2 :

ENV["TERM"] = "xterm-256color"
ENV["LC_ALL"] = "en_US.UTF-8"

Vagrant.configure("2") do |config|
  config.vm.box = "fedora/28-cloud-base" # defaults to fedora

  # common parts
  if Vagrant.has_plugin?("vagrant-vbguest")
    config.vbguest.auto_update = false
  end
  config.vm.provider :libvirt do |libvirt|
    libvirt.cpus = 2
    libvirt.memory = 4096
  end
  config.vm.provider :virtualbox do |vb|
    vb.check_guest_additions = false
    vb.functional_vboxsf = false
    vb.customize ["modifyvm", :id, "--memory", "4096"]
    vb.customize ["modifyvm", :id, "--cpus", "2"]
  end

  # Fedora 28
  config.vm.define "fedora", primary: true do |fedora|
    config.vm.provision "shell", inline: "dnf install -y btrfs-progs git go iptables libselinux-utils make polkit qemu-img rinetd systemd-container"

    config.vm.synced_folder ".", "/vagrant", disabled: true
    config.vm.synced_folder ".", "/home/vagrant/go/src/github.com/kinvolk/kube-spawn",
      create: true,
      owner: "vagrant",
      group: "vagrant",
      type: "rsync",
      rsync__exclude: ".kube-spawn/"

    # NOTE: chown is explicitly needed, even when synced_folder is configured
    # with correct owner/group. Maybe a vagrant issue?
    config.vm.provision "shell", inline: "mkdir -p /home/vagrant/go ; chown -R vagrant:vagrant /home/vagrant/go"

    config.vm.provision "shell", env: {"GOPATH" => "/home/vagrant/go", "KUBESPAWN_REDIRECT_TRAFFIC" => ENV["KUBESPAWN_REDIRECT_TRAFFIC"]}, privileged: false, path: "scripts/vagrant-setup-env.sh"
    config.vm.provision "shell", env: {"VUSER" => "vagrant"}, path: "scripts/vagrant-mod-env.sh"
    if ENV["KUBESPAWN_AUTOBUILD"] <=> "true"
      config.vm.provision "shell", env: {"GOPATH" => "/home/vagrant/go", "KUBESPAWN_REDIRECT_TRAFFIC" => ENV["KUBESPAWN_REDIRECT_TRAFFIC"]}, inline: "bash /home/vagrant/build.sh"
    end
  end

  # Ubuntu 18.04 (Artful)
  config.vm.define "ubuntu", autostart: false do |ubuntu|
    config.vm.box = "generic/ubuntu1804"
    config.vm.provision "shell", inline: "apt-get update; DEBIAN_FRONTEND=noninteractive apt-get install -y btrfs-progs git golang iptables make policykit-1 qemu-utils rinetd selinux-utils systemd-container"

    config.vm.synced_folder ".", "/vagrant", disabled: true
    config.vm.synced_folder ".", "/home/vagrant/go/src/github.com/kinvolk/kube-spawn",
      create: true,
      owner: "vagrant",
      group: "vagrant",
      type: "rsync",
      rsync__exclude: ".kube-spawn/"

    config.vm.provision "shell", inline: "mkdir -p /home/vagrant/go ; chown -R vagrant:vagrant /home/vagrant/go"
    config.vm.provision "shell", env: {"GOPATH" => "/home/vagrant/go", "KUBESPAWN_REDIRECT_TRAFFIC" => ENV["KUBESPAWN_REDIRECT_TRAFFIC"]}, privileged: false, path: "scripts/vagrant-setup-env.sh"
    config.vm.provision "shell", env: {"VUSER" => "vagrant"}, path: "scripts/vagrant-mod-env.sh"
    if ENV["KUBESPAWN_AUTOBUILD"] <=> "true"
      config.vm.provision "shell", env: {"GOPATH" => "/home/vagrant/go", "KUBESPAWN_REDIRECT_TRAFFIC" => ENV["KUBESPAWN_REDIRECT_TRAFFIC"]}, inline: "bash /home/vagrant/build.sh"
    end
  end

  # Debian testing
  config.vm.define "debian", autostart: false do |debian|
    config.vm.box = "debian/testing64"
    config.vm.provision "shell", inline: "echo deb http://httpredir.debian.org/debian unstable main >> /etc/apt/sources.list; apt-get update; DEBIAN_FRONTEND=noninteractive apt-get install -y btrfs-progs git golang iptables make policykit-1 qemu-utils rinetd selinux-utils systemd-container"

    config.vm.synced_folder ".", "/vagrant", disabled: true
    config.vm.synced_folder ".", "/home/vagrant/go/src/github.com/kinvolk/kube-spawn",
      create: true,
      owner: "vagrant",
      group: "vagrant",
      type: "rsync",
      rsync__exclude: ".kube-spawn/"

    config.vm.provision "shell", inline: "mkdir -p /home/vagrant/go ; chown -R vagrant:vagrant /home/vagrant/go"
    config.vm.provision "shell", env: {"GOPATH" => "/home/vagrant/go", "KUBESPAWN_REDIRECT_TRAFFIC" => ENV["KUBESPAWN_REDIRECT_TRAFFIC"]}, privileged: false, path: "scripts/vagrant-setup-env.sh"
    config.vm.provision "shell", env: {"VUSER" => "vagrant"}, path: "scripts/vagrant-mod-env.sh"
    if ENV["KUBESPAWN_AUTOBUILD"] <=> "true"
      config.vm.provision "shell", env: {"GOPATH" => "/home/vagrant/go", "KUBESPAWN_REDIRECT_TRAFFIC" => ENV["KUBESPAWN_REDIRECT_TRAFFIC"]}, inline: "bash /home/vagrant/build.sh"
    end
  end

  config.vm.network "forwarded_port", guest: 6443, host: 6443
end
