#!/usr/bin/make -f

IMAGE=docker.io/skpr/compass-php-extension

build:
	docker build --no-cache --build-arg=PHP_VERSION=8.3 -t $(IMAGE):8.3-latest .
	docker build --no-cache --build-arg=PHP_VERSION=8.2 -t $(IMAGE):8.2-latest .
	docker build --no-cache --build-arg=PHP_VERSION=8.1 -t $(IMAGE):8.1-latest .

.PHONY: *
