#!/usr/bin/make -f

lint: generate
	revive -set_exit_status -exclude=./vendor/... ./...

test: generate
	go test ./...

generate:
	go generate ./...

build: generate
	go build -a -o _output/compass-collector github.com/skpr/compass/collector/cmd/compass-collector
	go build -a -o _output/compass-find-lib github.com/skpr/compass/collector/cmd/compass-find-lib
	go build -a -buildmode=plugin -o _output/plugin/cli.so github.com/skpr/compass/collector/plugin/cli
	go build -a -buildmode=plugin -o _output/plugin/stdout.so github.com/skpr/compass/collector/plugin/stdout
	go build -a -buildmode=plugin -o _output/plugin/noop.so github.com/skpr/compass/collector/plugin/noop

.PHONY: *
