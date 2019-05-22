FROM golang:latest

ENV GOCACHE /tmp/.cache
ENV PATH "/go/bin:${PATH}"

WORKDIR /usr/src/kube-spawn

ENTRYPOINT ["make", "DOCKERIZED=n"]
