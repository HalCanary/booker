# Copyright 2022 Hal Canary
# Use of this program is governed by the file LICENSE.

all: booker

booker: $(wildcard *.go)
	go build .

clean:
	rm -f booker

clean-all:
	rm -rf booker dst

test:
	go test .

.PHONY: clean clean-all test all
