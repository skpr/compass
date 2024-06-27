#!/usr/bin/make -f

IMAGE=docker.io/skpr/compass-collector:latest

build:
	docker build --no-cache -t $(IMAGE) .

PID=""

run:
	docker run -it -e PID=$(PID) -v /proc:/host/proc -v /run:/host/run --privileged $(IMAGE)

.PHONY: *
