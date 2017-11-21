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
EOF
fi
