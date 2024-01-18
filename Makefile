# Copyright 2022 Hal Canary
# Use of this program is governed by the file LICENSE.

all: test build

go_commands = $(notdir $(wildcard cmd/*))
go_binaries = $(addprefix build/,$(go_commands))

${go_binaries}: $(shell find . -name '*.go') go.mod go.sum
	@mkdir -p build
	go get ./...
	go build -o build ./...

${HOME}/bin/%: build/%
	@mkdir -p $(dir $@)
	cp $^ $@

build: ${go_binaries}

install: $(addprefix ${HOME}/bin/,${go_commands})

clean:
	rm -rf build

test:
	go get ./...
	go test ./...

fmt:
	find . -type f -name '*.go' -exec gofmt -w {} \;

update_deps:
	go get -u ./...
	go mod tidy

.PHONY: all build clean fmt install test update_deps
