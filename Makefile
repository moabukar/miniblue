.PHONY: build test docker-build docker-run lint clean install

BINARY_NAME=miniblue
DOCKER_IMAGE=moabukar/miniblue
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS=-ldflags="-s -w -X github.com/moabukar/miniblue/internal/server.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/miniblue
	go build -ldflags="-s -w" -o bin/azlocal ./cmd/azlocal

test:
	go test -v ./...

lint:
	golangci-lint run

docker-build:
	docker build -t $(DOCKER_IMAGE):latest .

docker-run:
	docker run -p 4566:4566 -p 4567:4567 $(DOCKER_IMAGE):latest

clean:
	rm -rf bin/
	go clean

run:
	go run ./cmd/miniblue

install: build
	cp bin/azlocal /usr/local/bin/azlocal
	cp bin/miniblue /usr/local/bin/miniblue
	@echo "Installed miniblue and azlocal to /usr/local/bin/"

tidy:
	go mod tidy
