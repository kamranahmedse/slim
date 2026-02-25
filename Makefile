BINARY := localname
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build install clean test

build:
	go build -ldflags "-s -w -X github.com/kamrify/localname/cmd.Version=$(VERSION)" -o $(BINARY) .

install: build
	mv $(BINARY) /usr/local/bin/

clean:
	rm -f $(BINARY)

test:
	go test ./...
