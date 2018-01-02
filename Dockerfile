FROM golang:1.9.2-alpine3.7 AS builder

MAINTAINER Frank Mai <frank@rancher.com>

ARG BUILD_DATE
ARG VCS_REF
ARG VERSION

LABEL \
    io.github.thxcode.build-date=$BUILD_DATE \
    io.github.thxcode.name="etcd-console" \
    io.github.thxcode.description="A console supports etcd v2 and v3 by Alpine in a docker container." \
    io.github.thxcode.url="https://github.com/thxcode/etcd-console" \
    io.github.thxcode.vcs-type="Git" \
    io.github.thxcode.vcs-ref=$VCS_REF \
    io.github.thxcode.vcs-url="https://github.com/thxcode/etcd-console.git" \
    io.github.thxcode.vendor="Rancher Labs, Inc" \
    io.github.thxcode.version=$VERSION \
    io.github.thxcode.schema-version="1.0" \
    io.github.thxcode.license="MIT" \
    io.github.thxcode.docker.dockerfile="/Dockerfile"

ENV ETCD_ENDPOINTS="http://localhost:2379" \
    GRPC_GO_LOG_SEVERITY_LEVEL="WARNING" \
    GRPC_GO_LOG_VERBOSITY_LEVEL="WARNING" \
    DEP_VERSION="v0.3.2"

RUN apk add --update --no-cache \
    dumb-init bash sudo \
    && apk add --no-cache --virtual=build-dependencies \
	curl git \
    && rm -fr /var/cache/apk/* \
    && curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/${DEP_VERSION}/dep-linux-amd64 \
    && chmod +x /usr/local/bin/dep \
    && mkdir -p $GOPATH/src/github.com/thxcode/etcd-console

ADD . $GOPATH/src/github.com/thxcode/etcd-console/

RUN cd $GOPATH/src/github.com/thxcode/etcd-console \
    && dep ensure -v \
    && cd vendor/github.com/coreos/etcd \
    && dep init -v \
    && cd $GOPATH/src/github.com/thxcode/etcd-console \
    && go build -o ./.build/backend/etcd-console -v ./cmd/main.go \
    && mv ./.build/backend/etcd-console /usr/local/bin/ \
    && apk del --purge build-dependencies \
    && rm -rf \
        $GOPATH/src/* \
        /var/cache/apk/* \
        /root/.cache \
        /tmp/* \
        /usr/local/bin/dep \
        $GOPATH/pkg/dep/*

CMD ["etcd-console"]