BIN = kube-spawn
DOCKERIZED ?= y
DOCKER_TAG ?= latest
PREFIX ?= /usr
BINDIR ?= ${PREFIX}/bin
UID=$(shell id -u)
GOCACHEDIR=$(shell which go >/dev/null 2>&1 && go env GOCACHE || echo "$(HOME)/.cache/go-build")

.PHONY: all clean install

VERSION=$(shell git describe --tags --always --dirty)

ifeq ($(DOCKERIZED),y)
all:
	docker build -t kube-spawn-build:$(DOCKER_TAG) -f Dockerfile.build .
	mkdir -p $(GOCACHEDIR)
	docker run --rm -ti \
		-v `pwd`:/usr/src/kube-spawn:Z \
		-v $(GOCACHEDIR):/tmp/.cache:Z \
		--user $(UID):$(UID) \
		kube-spawn-build
else
all:
	GO111MODULE=on go mod download
	GO111MODULE=on go build -ldflags "-X main.version=$(VERSION)" \
		./cmd/kube-spawn
endif

update-vendor:
	GO111MODULE=on go get -u
	GO111MODULE=on go mod tidy

clean:
	rm -f \
		kube-spawn \

install:
	install kube-spawn "$(BINDIR)"
