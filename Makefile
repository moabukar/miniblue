.PHONY: build test docker-build docker-run lint clean

BINARY_NAME=local-azure
DOCKER_IMAGE=moabukar/local-azure

build:
	go build -o bin/$(BINARY_NAME) ./cmd/local-azure

test:
	go test -v ./...

lint:
	golangci-lint run

docker-build:
	docker build -t $(DOCKER_IMAGE):latest .

docker-run:
	docker run -p 4566:4566 $(DOCKER_IMAGE):latest

clean:
	rm -rf bin/
	go clean

run:
	go run ./cmd/local-azure

tidy:
	go mod tidy
