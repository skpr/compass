FROM alpine:3.21

USER root

RUN apk add alpine-sdk \
            bash \
            bpftool \
            clang \
            clang-dev \
            curl \
            git \
            libbpf-dev \
            linux-headers \
            llvm

ENV MISE_DATA_DIR="/mise"
ENV MISE_CONFIG_DIR="/mise"
ENV MISE_CACHE_DIR="/mise/cache"
ENV MISE_INSTALL_PATH="/usr/local/bin/mise"
ENV PATH="/mise/shims:$PATH"

RUN curl https://mise.run | sh

# Make libclang easy to find for bindgen
ENV LIBCLANG_PATH=/usr/lib/llvm19/lib
