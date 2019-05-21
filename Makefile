BIN = kube-spawn
DOCKERIZED ?= y
DOCKER_TAG ?= latest
PREFIX ?= /usr
BINDIR ?= ${PREFIX}/bin
UID=$(shell id -u)
GOCACHEDIR=$(shell which go >/dev/null 2>&1 && go env GOCACHE || echo "$(HOME)/.cache/go-build")

.PHONY: all clean dep install

VERSION=$(shell git describe --tags --always --dirty)

ifeq ($(DOCKERIZED),y)
all:
	docker build -t kube-spawn-build:$(DOCKER_TAG) -f Dockerfile.build .
	mkdir -p $(GOCACHEDIR)
	docker run --rm -ti \
		-v `pwd`:/go/src/github.com/kinvolk/kube-spawn:Z \
		-v $(GOCACHEDIR):/tmp/.cache:Z \
		--user $(UID):$(UID) \
		kube-spawn-build
else
all:
	go build -o kube-spawn \
		-ldflags "-X main.version=$(VERSION)" \
		./cmd/kube-spawn
endif

update-vendor: | dep
	dep ensure
dep:
	@which dep || go get -u github.com/golang/dep/cmd/dep

clean:
	rm -f \
		kube-spawn \

install:
	install kube-spawn "$(BINDIR)"
