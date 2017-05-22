.PHONY: vendor all reset-cni
.PHONY: clean clean-bins clean-rootfs clean-image clean-ssh-keys 

all:
	go build -o cni-noop ./cmd/cni-noop
	go build -o cnispawn ./cmd/cnispawn
	go build -o kubeadm-nspawn ./cmd/kubeadm-nspawn

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
