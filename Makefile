#!/usr/bin/make -f

up:
	# Building Compass.
	docker build -t local/compass:latest .
	docker compose build php-fpm
	docker compose up

down:
	docker compose down

lint: generate
	revive -set_exit_status ./cli/... ./collector/... ./trace/... ./sidecar/...

test: generate
	go test ./cli/... ./collector/... ./trace/... ./sidecar/...

generate:
	go generate ./collector/...

build: generate
	go build -a -o _output/compass github.com/skpr/compass/cli
	go build -a -o _output/compass-sidecar github.com/skpr/compass/sidecar

.PHONY: *