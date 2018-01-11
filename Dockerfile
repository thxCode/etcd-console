#############
# phase one #
#############
FROM golang:1.9.2-alpine3.7 AS builder

RUN apk add --no-cache \
	    curl \
        git \
        nodejs \
        yarn \
        python \
        python-dev \
        make \
        gcc \
        g++ \
        linux-headers \
        binutils-gold \
        gnupg \
        libstdc++ \
    ; \
    curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64; \
    chmod +x /usr/local/bin/dep; \
    mkdir -p $GOPATH/src/github.com/thxcode/etcd-console; \
    mkdir -p /build/backend; \
    mkdir -p /build/frontend

ADD . $GOPATH/src/github.com/thxcode/etcd-console/

## build backend
RUN cd $GOPATH/src/github.com/thxcode/etcd-console; \
    dep ensure -v; \
    cd vendor/github.com/coreos/etcd; \
    dep init -v; \
    cd $GOPATH/src/github.com/thxcode/etcd-console; \
    go build -o ./.build/backend/etcd-console -v ./cmd/main.go; \
    cp -f ./.build/backend/etcd-console /build/backend/

## build frontend
RUN cd $GOPATH/src/github.com/thxcode/etcd-console; \
    npm install -g @angular/cli --unsafe; \
    yarn install --prod; \
    ng build --prod; \
    cp -rf ./.build/frontend /build/frontend/etcd-console

## build entrypoint
RUN cd $GOPATH/src/github.com/thxcode/etcd-console; \
    chmod +x entrypoint.sh; \
    cp -r entrypoint.sh /build/

#############
# phase two #
#############
FROM alpine:3.7

MAINTAINER Frank Mai <frank@rancher.com>

ARG BUILD_DATE
ARG VCS_REF
ARG VERSION

LABEL \
    io.github.thxcode.build-date=$BUILD_DATE \
    io.github.thxcode.name="etcd-console" \
    io.github.thxcode.description="A console supports etcd v3 by Alpine in a docker container." \
    io.github.thxcode.url="https://github.com/thxcode/etcd-console" \
    io.github.thxcode.vcs-type="Git" \
    io.github.thxcode.vcs-ref=$VCS_REF \
    io.github.thxcode.vcs-url="https://github.com/thxcode/etcd-console.git" \
    io.github.thxcode.vendor="Rancher Labs, Inc" \
    io.github.thxcode.version=$VERSION \
    io.github.thxcode.schema-version="1.0" \
    io.github.thxcode.license="MIT" \
    io.github.thxcode.docker.dockerfile="/Dockerfile"

ENV GRPC_GO_LOG_SEVERITY_LEVEL="WARNING" \
    GRPC_GO_LOG_VERBOSITY_LEVEL="WARNING"

RUN apk add --update --no-cache \
        dumb-init \
        bash \
        sudo \
        nginx \
    ; \
    mkdir -p /run/nginx; \
    mkdir -p /run/cache

COPY default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /build/backend/etcd-console /usr/sbin/etcd-console
COPY --from=builder /build/frontend/etcd-console /var/www/etcd-console
COPY --from=builder /build/entrypoint.sh /usr/sbin/entrypoint.sh

ENTRYPOINT ["/usr/sbin/entrypoint.sh"]
