build:
		go build -o bin/vinit ./cmd/versainit
		strip -s bin/vinit

run:
		go run ./cmd/versainit

install:
		cp -v bin/vinit /usr/local/bin/vinit
