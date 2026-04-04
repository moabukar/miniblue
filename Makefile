.PHONY: build test docker-build docker-run lint clean install

BINARY_NAME=local-azure
DOCKER_IMAGE=moabukar/local-azure

build:
	go build -o bin/$(BINARY_NAME) ./cmd/local-azure
	go build -o bin/azlocal ./cmd/azlocal

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
	go run ./cmd/local-azure

install: build
	cp bin/azlocal /usr/local/bin/azlocal
	@echo "azlocal installed to /usr/local/bin/azlocal"

tidy:
	go mod tidy
