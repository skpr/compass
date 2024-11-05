FROM golang:1.23-alpine3.20

USER root

RUN apk add --no-cache curl && \
    curl -sSL https://packages.skpr.io/php-alpine/skpr.rsa.pub -o /etc/apk/keys/skpr.rsa.pub && \
    echo "https://packages.skpr.io/php-alpine/3.20/php8.3" >> /etc/apk/repositories

RUN apk add cargo \
            clang-dev \
            go \
            git \
            rust \
            rustfmt \
            php8.3-dev

ENV RUST_BACKTRACE=full
