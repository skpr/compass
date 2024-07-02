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
	go build -a -o _output/compass-collector github.com/skpr/compass/collector/cmd/compass-collector
	go build -a -buildmode=plugin -o _output/plugin/cache.so github.com/skpr/compass/collector/plugin/cache
	go build -a -buildmode=plugin -o _output/plugin/stdout.so github.com/skpr/compass/collector/plugin/stdout

.PHONY: *
