ARG PHP_VERSION=8.3
ARG ALPINE_VERSION=3.20

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

# This stage builds the collector component which will attach to the extension and collect telemetry.
FROM golang:1.24-alpine AS collector

    RUN apk add --no-cache ca-certificates llvm clang libbpf-dev make alpine-sdk linux-headers bpftool

    # Copy in the extension so we can use it map the probe arguments in our collector.
    COPY --from=extension /data/target/release/libcompass_extension.so /usr/lib/php/modules/compass.so
    
    ADD . /go/src/github.com/skpr/compass
    WORKDIR /go/src/github.com/skpr/compass

    RUN go install github.com/cilium/ebpf/cmd/bpf2go@v0.16.0
    RUN go install github.com/mgechev/revive@latest
    RUN make lint test build

FROM docker.io/skpr/php-cli:${PHP_VERSION}-v2-latest

    USER root
    RUN apk add binutils
    USER skpr

    # Extension
    COPY extension/compass.ini /etc/php/conf.d/00_compass.ini
    COPY --from=extension /data/target/release/libcompass_extension.so /usr/lib/php/modules/compass.so

    # Collector
    COPY --from=collector /go/src/github.com/skpr/compass/_output/compass /usr/local/bin/compass
    COPY --from=collector /go/src/github.com/skpr/compass/_output/compass-sidecar /usr/local/bin/compass-sidecar

    ENV COMPASS_PROCESS_NAME=php-fpm
    CMD ["compass-sidecar"]
