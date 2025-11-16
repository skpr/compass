FROM ghcr.io/skpr/php-cli:8.4-v2-stable

USER root

RUN apk add alpine-sdk \
            clang \
            clang-dev \
            git \
            php8.4-dev

ENV MISE_DATA_DIR="/mise"
ENV MISE_CONFIG_DIR="/mise"
ENV MISE_CACHE_DIR="/mise/cache"
ENV MISE_INSTALL_PATH="/usr/local/bin/mise"
ENV PATH="/mise/shims:$PATH"

RUN curl https://mise.run | sh

ENV RUSTFLAGS="-C target-feature=-crt-static"

ENV RUST_BACKTRACE=full
