**Spin up the Stack**

```bash
# Up the stack.
docker compose up -d

# Fix permissions.
docker compose exec --user=root php-fpm chown skpr:skpr /data/app/sites/default/files
docker compose exec --user=root php-fpm chown skpr:skpr /mnt/private
docker compose exec --user=root php-fpm chown skpr:skpr /mnt/temporary

# Install Drupal
docker compose exec php-fpm PHP_MEMORY_LIMIT=512M vendor/bin/drush si demo_umami
```
