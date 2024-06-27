FROM golang:1.22-alpine as build

RUN apk add --no-cache ca-certificates llvm clang libbpf-dev

# Collector.
ADD . /go/src/github.com/skpr/compass/collector
WORKDIR /go/src/github.com/skpr/compass/collector
RUN go install github.com/cilium/ebpf/cmd/bpf2go@v0.12.3
RUN go generate ./...
RUN go build -a -o bin/compass-collector github.com/skpr/compass/collector/cmd/compass-collector

# Plugins
RUN mkdir -p /usr/lib64/compass
RUN go build -a -buildmode=plugin -o /usr/lib64/compass/cache.so github.com/skpr/compass/collector/plugin/cache
RUN go build -a -buildmode=plugin -o /usr/lib64/compass/stdout.so github.com/skpr/compass/collector/plugin/stdout

FROM alpine:3.19
COPY --from=build /go/src/github.com/skpr/compass/collector/bin/compass-collector /usr/local/bin/compass-collector
COPY --from=build /usr/lib64/compass /usr/lib64/compass
ENTRYPOINT [ "/usr/local/bin/compass-collector"]
