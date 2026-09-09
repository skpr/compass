FROM alpine:3.21 AS build

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

ENV GOFLAGS=-buildvcs=false

WORKDIR /data
ADD . /data

RUN mise trust .

# Build both binaries.
RUN mise run build

# Lint and test, used by CI with "docker build --target=test".
FROM build AS test

RUN mise run lint
RUN mise run test
RUN mise run test:race

# Compass CLI.
FROM alpine:3.21 AS cli

RUN apk add bash binutils

COPY --from=build /data/_output/compass /usr/local/bin/compass

CMD ["compass"]

# Compass sidecar, the default target.
#
# scratch: the sidecar is a statically linked (CGO_ENABLED=0) server that never
# dials out, loads eBPF via the host's /sys/kernel/btf, and parses ELF in pure
# Go, so it needs nothing from userspace. Shipping just the binary keeps this
# privileged, host-PID container's attack surface to a minimum. There is no
# shell, so debug with an ephemeral container rather than `exec sh`.
FROM scratch AS sidecar

COPY --from=build /data/_output/compass-sidecar /usr/local/bin/compass-sidecar

ENV COMPASS_SIDECAR_PHP_PROCESS_NAME=php-fpm

# Absolute path: scratch has no shell or PATH to resolve a bare command against.
ENTRYPOINT ["/usr/local/bin/compass-sidecar"]

# Compass daemon, a per-node DaemonSet collector.
#
# scratch for the same reasons as the sidecar: a static, server-only binary in a
# privileged, host-PID container.
FROM scratch AS daemon

COPY --from=build /data/_output/compass-daemon /usr/local/bin/compass-daemon

ENTRYPOINT ["/usr/local/bin/compass-daemon"]
