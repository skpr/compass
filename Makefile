#!/usr/bin/make -f

EXTENSION_INSTALLED=true
EXTENSION_ENABLED=true

IMAGE_NGINX=localhost/compass:nginx-latest
IMAGE_FPM=localhost/compass:php-fpm-latest
IMAGE_FPM_WITH_EXTENSION=localhost/compass:php-fpm-ext-latest

IMAGE_EXTENSION=localhost/compass:extension-latest
IMAGE_COLLECTOR=localhost/compass:collector-latest

DOCKER_COMPOSE_PROFILE=all

build:
	# Building base stack.
	docker build -t $(IMAGE_NGINX) docker/compose/nginx
	docker build -t $(IMAGE_FPM) docker/compose/php-fpm
	# Building extension.
	docker build --no-cache --build-arg=PHP_VERSION=8.3 -t $(IMAGE_EXTENSION) extension
	docker build -t $(IMAGE_FPM_WITH_EXTENSION) docker/compose/php-fpm-ext
	# Building collector.
	docker build --no-cache --build-arg=PHP_VERSION=8.3 -t $(IMAGE_COLLECTOR) collector

up:
	ifeq ($(EXTENSION_INSTALLED),true)
		IMAGE_PHP_FPM=$(IMAGE_FPM_WITH_EXTENSION) COMPASS_ENABLED=$(EXTENSION_ENABLED) docker compose --profile $(DOCKER_COMPOSE_PROFILE) up -d --wait
	else
		IMAGE_PHP_FPM=$(IMAGE_FPM) COMPASS_ENABLED=$(EXTENSION_ENABLED) docker compose --profile $(DOCKER_COMPOSE_PROFILE) up -d --wait
	endif

install:
	docker compose exec --user=root php-fpm chown skpr:skpr /data/app/sites/default/files
	docker compose exec --user=root php-fpm chown skpr:skpr /mnt/private
	docker compose exec --user=root php-fpm chown skpr:skpr /mnt/temporary
	docker compose exec -e PHP_MEMORY_LIMIT=512M php-fpm vendor/bin/drush si demo_umami

stop:
	docker compose stop

.PHONY: *
