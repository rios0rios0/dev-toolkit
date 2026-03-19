SCRIPTS_DIR ?= $(HOME)/Development/github.com/rios0rios0/pipelines
-include $(SCRIPTS_DIR)/makefiles/common.mk
-include $(SCRIPTS_DIR)/makefiles/golang.mk

VERSION ?= $(or $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'),dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: debug build build-musl install run

build:
	rm -rf bin
	go build -ldflags "$(LDFLAGS) -s -w" -o bin/vinit ./cmd/versainit
	strip -s bin/vinit

debug:
	rm -rf bin
	go build -gcflags "-N -l" -ldflags "$(LDFLAGS)" -o bin/vinit ./cmd/versainit

build-musl:
	CGO_ENABLED=1 CC=musl-gcc go build \
		-ldflags '$(LDFLAGS) -linkmode external -extldflags="-static"' -o bin/vinit ./cmd/versainit
	strip -s bin/vinit

run:
	go run ./cmd/versainit

install:
	make build
	mkdir -p ~/.local/bin
	cp -v bin/vinit ~/.local/bin/vinit
