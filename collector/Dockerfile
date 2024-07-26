FROM golang:1.22-alpine as build

RUN apk add --no-cache ca-certificates llvm clang libbpf-dev make

# Collector.
ADD . /go/src/github.com/skpr/compass/collector
WORKDIR /go/src/github.com/skpr/compass/collector
RUN go install github.com/cilium/ebpf/cmd/bpf2go@v0.12.3
RUN make fmt vet test build

FROM alpine:3.19

COPY --from=build /go/src/github.com/skpr/compass/collector/_output/compass-collector /usr/local/bin/compass-collector
COPY --from=build /go/src/github.com/skpr/compass/collector/_output/compass-find-lib /usr/local/bin/compass-find-lib
COPY --from=build /go/src/github.com/skpr/compass/collector/_output/plugin /usr/lib64/compass

ADD docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV COMPASS_PROCESS_NAME=php-fpm

CMD ["/entrypoint.sh"]
