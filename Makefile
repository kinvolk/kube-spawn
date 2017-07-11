.PHONY: vendor all reset-cni
.PHONY: clean clean-bins clean-rootfs clean-image clean-ssh-keys 

VERSION=$(shell git describe --tags --always --dirty)

all:
	go build -o cni-noop ./cmd/cni-noop
	go build -o cnispawn ./cmd/cnispawn
	go build -o nspawn-runc ./cmd/nspawn-runc
	go build -o kubeadm-nspawn \
		-ldflags "-X main.version=$(VERSION)" \
		./cmd/kubeadm-nspawn

cnibridge:
	go get -u github.com/containernetworking/plugins/plugins/main/bridge

vendor: glide.lock | glide
	glide --quiet install --strip-vendor
glide.lock: | glide
	glide update --strip-vendor
glide:
	@which glide || go get -u github.com/Masterminds/glide

clean: clean-bins clean-rootfs clean-image clean-ssh-keys
clean-ssh-keys:
	rm -rf ./id_rsa*
clean-bins:
	rm -rf ./{cni-noop,cnispawn,kubeadm-nspawn}
clean-rootfs:
	sudo rm -rf kubeadm-nspawn-*
clean-image:
	rm -rf rootfs.tar.xz

reset-cni:
	sudo rm -rf /var/lib/cni/networks/mynet
