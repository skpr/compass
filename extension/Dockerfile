ARG PHP_VERSION=8.3
FROM docker.io/skpr/php-cli:${PHP_VERSION}-v2-latest as build

USER root
RUN apk add rust rustfmt cargo php${PHP_VERSION}-dev clang-dev
USER skpr

ADD --chown=skpr:skpr . /data

ENV RUST_BACKTRACE=full
RUN cargo build --release

FROM docker.io/skpr/php-cli:${PHP_VERSION}-v2-latest

USER root
RUN apk add binutils
USER skpr

COPY compass.ini /etc/php/conf.d/00_compass.ini
COPY --from=build /data/target/release/libcompass_extension.so /usr/lib/php/modules/compass.so
