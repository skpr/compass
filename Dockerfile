ARG PHP_VERSION=8.3
FROM ghcr.io/skpr/php-cli:${PHP_VERSION}-v2-stable as build

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

# Quality tools.
RUN mise run lint
RUN mise run test

RUN mise run build

FROM scratch

# Extension
COPY extension/compass.ini /etc/php/conf.d/00_compass.ini
COPY --from=build /data/target/release/libcompass_extension.so /usr/lib/php/modules/compass.so

# Collector
COPY --from=build /go/src/github.com/skpr/compass/_output/compass /usr/local/bin/compass
COPY --from=build /go/src/github.com/skpr/compass/_output/compass-sidecar /usr/local/bin/compass-sidecar

ENV COLORTERM=truecolor
ENV COMPASS_SIDECAR_PROCESS_NAME=php-fpm
CMD ["compass-sidecar"]
