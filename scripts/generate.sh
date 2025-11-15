#!/usr/bin/env bash
set -euo pipefail

arch="$(uname -m)"

case "$arch" in
    x86_64)
        ARCH="amd64"
        ;;
    arm64 | aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "Unknown architecture: $arch"
        ARCH="unknown"
        ;;
esac

cd collector

bpftool btf dump file /sys/kernel/btf/vmlinux format c > ./includes/vmlinux.h
readelf -n /usr/lib/php/modules/compass.so | go run ./scripts/bpftmpl -arch="${ARCH}" -template=./template.bpf.c > ./program.bpf.c
GOPACKAGE=collector bpf2go -target "${ARCH}" -cflags "-O2 -g -Wall -Werror" -type event bpf program.bpf.c -- -I./includes
