FROM ghcr.io/skpr/php-cli:8.4-v2-stable

USER root

RUN apk add alpine-sdk \
            bpftool \
            clang \
            clang-dev \
            git \
            libbpf-dev \
            linux-headers \
            llvm \
            php8.4-dev

ENV MISE_DATA_DIR="/mise"
ENV MISE_CONFIG_DIR="/mise"
ENV MISE_CACHE_DIR="/mise/cache"
ENV MISE_INSTALL_PATH="/usr/local/bin/mise"
ENV PATH="/mise/shims:$PATH"

RUN curl https://mise.run | sh

# Make libclang easy to find for bindgen
ENV LIBCLANG_PATH=/usr/lib/llvm19/lib

ENV RUSTFLAGS="-C target-feature=-crt-static"

ENV RUST_BACKTRACE=full
