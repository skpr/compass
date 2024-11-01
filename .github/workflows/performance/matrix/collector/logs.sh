#!/bin/bash

OUTPUT_DIR=$1

docker compose logs nginx > ${OUTPUT_DIR}/nginx.log
docker compose logs php-fpm > ${OUTPUT_DIR}/php-fpm.log
docker compose logs mysql-default > ${OUTPUT_DIR}/mysql-default.log
docker compose logs collector > ${OUTPUT_DIR}/collector.log
