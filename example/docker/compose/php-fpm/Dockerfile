FROM skpr/php-fpm:8.3-dev-v2-latest

COPY --from=composer:2 /usr/bin/composer /usr/local/bin/

# Copy the Drupal application code in.
COPY --from=drupal:10.2 --chown=skpr:skpr /opt/drupal /data

# Make this Drupal application compatible with our images.
RUN ln -sf /data/web /data/app

# Override settings.php
ADD --chown=skpr:skpr settings.php /data/app/sites/default/settings.php
ADD --chown=skpr:skpr services.yml /data/app/sites/default/services.yml

# Add drush
RUN composer require drush/drush
COPY drush /etc/drush

COPY --from=docker.io/skpr/compass-php-extension:8.3-latest /etc/php/conf.d/00_compass.ini /etc/php/conf.d/00_compass.ini
COPY --from=docker.io/skpr/compass-php-extension:8.3-latest /usr/lib/php/modules/compass.so /usr/lib/php/modules/compass.so
