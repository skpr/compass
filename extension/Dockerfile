ARG PHP_VERSION=8.3
FROM docker.io/skpr/php-cli:${PHP_VERSION}-v2-latest as build

USER root

RUN apk add autoconf \
            gcc \
            make \
            musl-dev \
            php8.3-dev \
            binutils \
            wget \
            g++ \
            python3

ADD . /data

WORKDIR /data

RUN phpize && \
    ./configure --prefix=/usr --with-php-config=/usr/bin/php-config && \
    make && \
    cp modules/observer.so /usr/lib/php/modules/compass.so

FROM docker.io/skpr/php-cli:${PHP_VERSION}-v2-latest
COPY compass.ini /etc/php/conf.d/00_compass.ini
COPY --from=build /usr/lib/php/modules/compass.so /usr/lib/php/modules/compass.so
