ARG PHP_VERSION=8.3
ARG ALPINE_VERSION=3.21

# This is the image used to build the extension.
# We need the latest version of Alpine so we get a later version of Rust for PHP 8.4 support.
FROM alpine:3.21 AS extension

    ARG PHP_VERSION=8.3
    ARG ALPINE_VERSION=3.20

    # Install PHP.
    RUN apk add --no-cache curl && \
        curl -sSL https://packages.skpr.io/php-alpine/skpr.rsa.pub -o /etc/apk/keys/skpr.rsa.pub && \
        echo "https://packages.skpr.io/php-alpine/${ALPINE_VERSION}/php${PHP_VERSION}" >> /etc/apk/repositories

    RUN apk --update --no-cache add rust rustfmt cargo php${PHP_VERSION}-dev clang-dev

    # Build the project.
    ADD --chown=skpr:skpr extension /data
    WORKDIR /data

    ENV RUST_BACKTRACE=full
    RUN cargo fmt --all -- --check
    RUN cargo build --release

FROM docker.io/skpr/php-cli:${PHP_VERSION}-v2-latest

    USER root
    RUN apk add binutils
    USER skpr

    # Extension
    COPY extension/compass.ini /etc/php/conf.d/00_compass.ini
    COPY --from=extension /data/target/release/libcompass_extension.so /usr/lib/php/modules/compass.so

    ENV COLORTERM=truecolor
    ENV COMPASS_PROCESS_NAME=php-fpm
    CMD ["compass-sidecar"]
