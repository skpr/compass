FROM alpine:3.21

USER root

RUN apk add --no-cache curl && \
    curl -sSL https://packages.skpr.io/php-alpine/skpr.rsa.pub -o /etc/apk/keys/skpr.rsa.pub && \
    echo "https://packages.skpr.io/php-alpine/3.21/php8.4" >> /etc/apk/repositories

RUN apk add alpine-sdk \
            clang \
            clang-dev \
            git \
            php8.4 \
            php8.4-dev

ENV MISE_DATA_DIR="/mise"
ENV MISE_CONFIG_DIR="/mise"
ENV MISE_CACHE_DIR="/mise/cache"
ENV MISE_INSTALL_PATH="/usr/local/bin/mise"
ENV PATH="/mise/shims:$PATH"

RUN curl https://mise.run | sh

ENV RUSTFLAGS="-C target-feature=-crt-static"

ENV RUST_BACKTRACE=full

WORKDIR /data

