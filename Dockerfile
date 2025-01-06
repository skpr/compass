ARG PHP_VERSION=8.4
FROM docker.io/skpr/php-cli:${PHP_VERSION}-v2-latest AS extension
USER root
RUN apk add rust rustfmt cargo php${PHP_VERSION}-dev clang-dev
USER skpr
ADD --chown=skpr:skpr extension /data
ENV RUST_BACKTRACE=full
RUN cargo fmt --all -- --check
RUN cargo build --release

FROM golang:1.23-alpine AS collector
# Copy in the extension so we can use it map the probe arguments in our collector.
COPY --from=extension /data/target/release/libcompass_extension.so /usr/lib/php/modules/compass.so
RUN apk add --no-cache ca-certificates llvm clang libbpf-dev make alpine-sdk linux-headers bpftool
ADD . /go/src/github.com/skpr/compass
WORKDIR /go/src/github.com/skpr/compass
RUN go install github.com/cilium/ebpf/cmd/bpf2go@v0.16.0
RUN go install github.com/mgechev/revive@latest
RUN make lint test build

FROM docker.io/skpr/php-cli:${PHP_VERSION}-v2-latest
# Extension
COPY extension/compass.ini /etc/php/conf.d/00_compass.ini
COPY --from=extension /data/target/release/libcompass_extension.so /usr/lib/php/modules/compass.so
# Collector
COPY --from=collector /go/src/github.com/skpr/compass/_output/compass /usr/local/bin/compass
COPY --from=collector /go/src/github.com/skpr/compass/_output/compass-sidecar /usr/local/bin/compass-sidecar
USER root
RUN apk add binutils
USER skpr
ENV COMPASS_PROCESS_NAME=php-fpm
CMD ["compass-sidecar"]
