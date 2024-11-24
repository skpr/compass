#!/usr/bin/make -f

SYSTEM_ARCH := $(shell uname -m)

ifeq ($(SYSTEM_ARCH),x86_64)
    ARCH = amd64
else ifeq ($(SYSTEM_ARCH),aarch64)
    ARCH = arm64
else
    ARCH = unknown
endif

up:
	# Building Compass.
	docker build -t local/compass:latest .
	docker compose build php-fpm
	docker compose up

down:
	docker compose down

lint: generate
	revive -set_exit_status ./cli/... ./collector/... ./profile/... ./sidecar/...

test: generate
	go test ./cli/... ./collector/... ./profile/... ./sidecar/...

generate:
	cd collector
	bpftool btf dump file /sys/kernel/btf/vmlinux format c > ./includes/vmlinux.h
	readelf -n /usr/lib/php/modules/compass.so | go run ./scripts/bpftmpl -arch=${ARCH} -template=./template.bpf.c > ./program.bpf.c
	GOPACKAGE=collector bpf2go -target ${ARCH} -type event bpf program.bpf.c -- -I./includes

build: generate
	go build -a -o _output/compass github.com/skpr/compass/cli
	go build -a -o _output/compass-sidecar github.com/skpr/compass/sidecar

# https://www.gnu.org/software/make/manual/html_node/One-Shell.html
.ONESHELL: # Applies to every targets in the file!

.PHONY: *