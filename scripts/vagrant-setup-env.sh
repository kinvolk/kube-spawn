#!/bin/bash

set -eo pipefail

echo 'Setting up correct env. variables'
echo "export GOPATH=$GOPATH" >> "$HOME/.bash_profile"
echo "export PATH=$PATH:$GOPATH/bin:/usr/local/go/bin" >> "$HOME/.bash_profile"
echo "export KUBECONFIG=/var/lib/kube-spawn/default/kubeconfig" >> "$HOME/.bash_profile"

# shellcheck disable=SC1090
source ~/.bash_profile

# -u must be set after "source ~/.bash_profile" to avoid errors like
# "PS1: unbound variable"
set -u

echo 'Writing build.sh'

if [[ ! -f $HOME/build.sh ]]; then
	cat >>"$HOME/build.sh" <<-EOF
#!/bin/bash
set -xe

cd $GOPATH/src/github.com/kinvolk/kube-spawn

go get -u github.com/containernetworking/plugins/plugins/...

DOCKERIZED=n make all

sudo machinectl show-image coreos || sudo machinectl pull-raw --verify=no https://alpha.release.core-os.net/amd64-usr/current/coreos_developer_container.bin.bz2 coreos

sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kube-spawn create --nodes=2
sudo GOPATH=$GOPATH CNI_PATH=$GOPATH/bin ./kube-spawn start

if [ "\$KUBESPAWN_REDIRECT_TRAFFIC" == "true" ]; then
	# Redirect traffic from the VM to kube-apiserver inside container
	APISERVER_IP_PORT=\$(grep server /var/lib/kube-spawn/default/kubeconfig | awk '{print \$2;}' | perl -pe 's/(https|http):\/\///g')
	APISERVER_IP=\$(echo \$APISERVER_IP_PORT | perl -pe 's/:\d*$//g')
	APISERVER_PORT=\$(echo \$APISERVER_IP_PORT | perl -pe 's/^[\d.]+://g')
	echo "0.0.0.0 \$APISERVER_PORT \$APISERVER_IP \$APISERVER_PORT" | sudo tee /etc/rinetd.conf > /dev/null
	sudo systemctl enable rinetd
	sudo systemctl start rinetd

	# Generate kubeconfig
	cd /home/vagrant
	VAGRANT_IP=\$(ip addr show eth0 | grep "inet\\b" | awk '{print \$2}' | cut -d/ -f1)
	cp /var/lib/kube-spawn/default/kubeconfig .
	perl -pi.back -e "s/\$APISERVER_IP/\$VAGRANT_IP/g;" kubeconfig
	perl -pi.back -e "s/certificate-authority-data.*/insecure-skip-tls-verify: true/g;" kubeconfig
fi
EOF
fi

KUBERNETES_VERSION=$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)
sudo curl -Lo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBERNETES_VERSION}/bin/linux/amd64/kubectl
sudo chmod +x /usr/local/bin/kubectl
