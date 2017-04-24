all: vendor
	go build -o cni-noop ./cmd/cni-noop
	go build -o cnispawn ./cmd/cnispawn
	go build -o kubeadm-nspawn ./cmd/kubeadm-nspawn

glide:
	@which glide || go get -u github.com/Masterminds/glide

glide.lock: glide.yaml | glide
	glide update --strip-vendor

vendor: glide.lock | glide
	glide install --strip-vendor

.PHONY: glide vendor all

clean:
	rm -rf ./{kubeadm-nspawn,cni-noop,cnispawn}

clean-rootfs:
	sudo rm -rf kubeadm-nspawn-*

clean-cni:
	sudo rm -rf /var/lib/cni/networks/mynet/*
	sudo iptables-restore < ~/iptables

clean-image:
	rm -rf rootfs.tar.xz
