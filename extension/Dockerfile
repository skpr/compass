ARG PHP_VERSION=8.3
FROM alpine:3.21 AS build
ARG PHP_VERSION=8.3

USER root

RUN apk add --no-cache curl && \
    curl -sSL https://packages.skpr.io/php-alpine/skpr.rsa.pub -o /etc/apk/keys/skpr.rsa.pub && \
    echo "https://packages.skpr.io/php-alpine/3.21/php${PHP_VERSION}" >> /etc/apk/repositories

RUN apk add alpine-sdk clang clang-dev php${PHP_VERSION} php${PHP_VERSION}-dev

ENV MISE_DATA_DIR="/mise"
ENV MISE_CONFIG_DIR="/mise"
ENV MISE_CACHE_DIR="/mise/cache"
ENV MISE_INSTALL_PATH="/usr/local/bin/mise"
ENV PATH="/mise/shims:$PATH"

RUN curl https://mise.run | sh

ENV RUSTFLAGS="-C target-feature=-crt-static"
ENV RUST_BACKTRACE=full

WORKDIR /data

ADD --chown=skpr:skpr . /data
RUN mise trust .

RUN mise run lint
RUN mise run build
RUN mise run validate

FROM scratch

COPY compass.ini /etc/php/conf.d/00_compass.ini
COPY --from=build /data/target/release/libcompass_extension.so /usr/lib/php/modules/compass.so
