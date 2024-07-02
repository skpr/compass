#!/usr/bin/make -f

fmt: generate
	go fmt ./...

vet: generate
	go vet ./...

test: generate
	go test ./...

generate:
	go generate ./...

build: generate
	go build -a -o bin/compass-collector github.com/skpr/compass/collector/cmd/compass-collector

.PHONY: *
