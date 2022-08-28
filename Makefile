# Copyright 2022 Hal Canary
# Use of this program is governed by the file LICENSE.

all: test doc booker

booker: $(shell find . -name '*.go')
	go build ./cmd/booker

clean:
	rm -f booker

test:
	go test ./...

doc:
	for x in `go list ./...`; do go doc --all $$x > docs/`basename $$x`.txt; done

fmt:
	find . -type f -name '*.go' -exec gofmt -w {} \;

.PHONY: clean clean-all test all doc fmt
