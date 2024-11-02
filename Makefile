#!/usr/bin/make -f

lint: generate
	revive -set_exit_status ./cli/... ./collector/... ./profile/... ./sidecar/...

test: generate
	go test ./cli/... ./collector/... ./profile/... ./sidecar/...

generate:
	go generate ./collector/...

build: generate
	go build -a -o _output/compass github.com/skpr/compass/cli
	go build -a -o _output/compass-sidecar github.com/skpr/compass/sidecar

.PHONY: *