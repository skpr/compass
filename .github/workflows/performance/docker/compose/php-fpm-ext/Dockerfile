FROM localhost/compass:php-fpm-latest

COPY --from=localhost/compass-extension:latest /etc/php/conf.d/00_compass.ini /etc/php/conf.d/00_compass.ini
COPY --from=localhost/compass-extension:latest /usr/lib/php/modules/compass.so /usr/lib/php/modules/compass.so
