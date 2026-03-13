SCRIPTS_DIR ?= $(HOME)/Development/github.com/rios0rios0/pipelines
-include $(SCRIPTS_DIR)/makefiles/common.mk
-include $(SCRIPTS_DIR)/makefiles/golang.mk

.PHONY: debug build build-musl install run

build:
	rm -rf bin
	go build -o bin/vinit ./cmd/versainit
	strip -s bin/vinit

debug:
	rm -rf bin
	go build -gcflags "-N -l" -o bin/vinit ./cmd/versainit

build-musl:
	CGO_ENABLED=1 CC=musl-gcc go build \
		--ldflags 'linkmode external -extldflags="-static"' -o bin/vinit ./cmd/vinit
	strip -s bin/vinit

run:
	go run ./cmd/versainit

install:
	make build
	mkdir -p ~/.local/bin
	cp -v bin/vinit ~/.local/bin/vinit
