SCRIPTS_DIR ?= $(HOME)/Development/github.com/rios0rios0/pipelines
-include $(SCRIPTS_DIR)/makefiles/common.mk
-include $(SCRIPTS_DIR)/makefiles/golang.mk

VERSION ?= $(or $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//'),dev)
LDFLAGS := -X main.version=$(VERSION) -X main.repoOwner=rios0rios0 -X main.repoName=devforge -X main.binaryName=dev

.PHONY: debug build build-musl install run

build:
	rm -rf bin
	go build -ldflags "$(LDFLAGS) -s -w" -o bin/dev ./cmd/devforge
	strip -s bin/dev

debug:
	rm -rf bin
	go build -gcflags "-N -l" -ldflags "$(LDFLAGS)" -o bin/dev ./cmd/devforge

build-musl:
	CGO_ENABLED=1 CC=musl-gcc go build \
		-ldflags '$(LDFLAGS) -linkmode external -extldflags="-static"' -o bin/dev ./cmd/devforge
	strip -s bin/dev

run:
	go run ./cmd/devforge

install:
	make build
	mkdir -p ~/.local/bin
	cp -v bin/dev ~/.local/bin/dev
