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
FROM alpine:3.21 AS sidecar

RUN apk add bash binutils

COPY --from=build /data/_output/compass-sidecar /usr/local/bin/compass-sidecar

ENV COMPASS_SIDECAR_PHP_PROCESS_NAME=php-fpm

CMD ["compass-sidecar"]
