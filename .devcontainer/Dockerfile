FROM golang:1.24-alpine3.21

USER root

RUN apk add --no-cache curl && \
    curl -sSL https://packages.skpr.io/php-alpine/skpr.rsa.pub -o /etc/apk/keys/skpr.rsa.pub && \
    echo "https://packages.skpr.io/php-alpine/3.20/php8.3" >> /etc/apk/repositories

RUN apk add clang-dev \
            go \
            git \
            php8.3-dev \ 
            ca-certificates \
            llvm \
            clang \
            libbpf-dev \
            make \
            alpine-sdk \
            linux-headers \
            bpftool

RUN apk add cargo rust rustfmt --repository=http://dl-cdn.alpinelinux.org/alpine/edge/main

ENV RUST_BACKTRACE=full
