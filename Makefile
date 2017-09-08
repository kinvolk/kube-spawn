.PHONY: vendor all clean dep

VERSION=$(shell git describe --tags --always --dirty)

all:
	go build -o cni-noop ./cmd/cni-noop
	go build -o cnispawn ./cmd/cnispawn
	go build -o kube-spawn-runc ./cmd/kube-spawn-runc
	go build -o kube-spawn \
		-ldflags "-X main.version=$(VERSION)" \
		./cmd/kube-spawn

vendor: | dep
	dep ensure
dep:
	@which dep || go get -u github.com/golang/dep/cmd/dep

clean:
	rm -rf \
		cni-noop \
		cnispawn \
		kube-spawn \
		kube-spawn-runc
