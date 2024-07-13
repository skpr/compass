FROM golang:1.22-alpine as build

RUN apk add --no-cache ca-certificates llvm clang libbpf-dev make

# Collector.
ADD . /go/src/github.com/skpr/compass/collector
WORKDIR /go/src/github.com/skpr/compass/collector
RUN go install github.com/cilium/ebpf/cmd/bpf2go@v0.12.3
RUN make fmt vet test build

FROM ubuntu:latest

COPY --from=build /go/src/github.com/skpr/compass/collector/_output/compass-find-lib /usr/local/bin/compass-find-lib

RUN apt-get update && \
    apt-get install -y bpftrace && \
    apt-get clean

ADD docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENV COMPASS_PROCESS_NAME=php-fpm

CMD ["/entrypoint.sh"]
