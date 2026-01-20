FROM from_image AS build

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

  # Make libclang easy to find for bindgen
  ENV LIBCLANG_PATH=/usr/lib/llvm19/lib

  ENV GOFLAGS=-buildvcs=false

  WORKDIR /data
  ADD --chown=skpr:skpr . /data

  # Check and build.
  RUN mise trust .
  RUN mise run build:sidecar

FROM scratch

  COPY --from=build /data/_output/compass-sidecar /usr/local/bin/compass-sidecar
  ENV COMPASS_SIDECAR_PROCESS_NAME=php-fpm
  CMD ["compass-sidecar"]
