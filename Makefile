all: install-vendor
	go build -o cni-noop ./cmd/cni-noop
	go build -o cnispawn ./cmd/cnispawn
	go build -o kubeadm-systemd ./cmd/kubeadm-systemd

check-glide-installation:
	@which glide || go get -u github.com/Masterminds/glide

install-vendor: check-glide-installation
	glide install --strip-vendor

update-vendor: check-glide-installation
	glide update --strip-vendor

.PHONY: check-glide-installation install-vendor update-vendor all
