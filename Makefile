# Copyright 2022 Hal Canary
# Use of this program is governed by the file LICENSE.

all: test build

build: $(shell find . -name '*.go') go.mod go.sum
	mkdir -p build
	go get ./...
	go build -o build ./...

clean:
	rm -f booker

test:
	go get ./...
	go test ./...

fmt:
	find . -type f -name '*.go' -exec gofmt -w {} \;

update_deps:
	go get -u ./...
	go mod tidy

install: build
	cp build/booker ~/bin/

.PHONY: all clean fmt test update_deps build install
