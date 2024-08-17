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
	sudo cp -v bin/vinit /usr/local/bin/vinit
