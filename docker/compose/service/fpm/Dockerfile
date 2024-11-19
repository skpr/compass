FROM local/compass:latest as compass

FROM skpr/php-fpm:8.3-dev-v2-latest
COPY --from=compass /etc/php/conf.d/00_compass.ini /etc/php/conf.d/00_compass.ini
COPY --from=compass /usr/lib/php/modules/compass.so /usr/lib/php/modules/compass.so
