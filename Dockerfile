ARG PHP_VERSION=8.3
FROM docker.io/skpr/php-cli:${PHP_VERSION}-v2-latest as extension
USER root
RUN apk add rust rustfmt cargo php${PHP_VERSION}-dev clang-dev
USER skpr
ADD --chown=skpr:skpr extension /data
ENV RUST_BACKTRACE=full
RUN cargo fmt --all -- --check
RUN cargo build --release
# Validate arguments.
RUN chmod +x /data/validate.sh
RUN /data/validate.sh /data/target/release/libcompass_extension.so

FROM golang:1.23-alpine as collector
RUN apk add --no-cache ca-certificates llvm clang libbpf-dev make
ADD collector /go/src/github.com/skpr/compass/collector
WORKDIR /go/src/github.com/skpr/compass/collector
RUN go install github.com/cilium/ebpf/cmd/bpf2go@v0.16.0
RUN go install github.com/mgechev/revive@latest
RUN make lint test build

FROM docker.io/skpr/php-cli:${PHP_VERSION}-v2-latest
# Extension
COPY extension/compass.ini /etc/php/conf.d/00_compass.ini
COPY --from=extension /data/target/release/libcompass_extension.so /usr/lib/php/modules/compass.so
# Collector
COPY --from=collector /go/src/github.com/skpr/compass/collector/_output/compass-collector /usr/local/bin/compass-collector
COPY --from=collector /go/src/github.com/skpr/compass/collector/_output/compass-find-lib /usr/local/bin/compass-find-lib
COPY --from=collector /go/src/github.com/skpr/compass/collector/_output/compass /usr/local/bin/compass
COPY --from=collector /go/src/github.com/skpr/compass/collector/_output/plugin /usr/lib64/compass
ADD collector/docker/entrypoint.sh /usr/local/bin/compass-collector-entrypoint
USER root
RUN chmod +x /usr/local/bin/compass-collector-entrypoint
RUN apk add binutils
USER skpr
ENV COMPASS_ENABLED="false"
ENV COMPASS_MODE=""
ENV COMPASS_HEADER=""
ENV COMPASS_FUNCTION_THRESHOLD="10000"
ENV COMPASS_PROCESS_NAME=php-fpm
CMD ["/usr/local/bin/compass-collector-entrypoint"]
