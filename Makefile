# Copyright 2022 Hal Canary
# Use of this program is governed by the file LICENSE.

all: test booker

booker: $(shell find . -name '*.go')
	go build ./cmd/booker

clean:
	rm -f booker

test: 
	go test ./...

fmt:
	find . -type f -name '*.go' -exec gofmt -w {} \;

define test_build_rule
.PHONY: test.$(1)
test.$(1):
	go test -v ./$(1)
endef

packages := $(shell go list ./... | sed s@^$(shell go list -m)/@@)
$(foreach x,$(packages),$(eval $(call test_build_rule,$x)))

.PHONY: all clean fmt test
