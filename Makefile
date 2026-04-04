.PHONY: build test docker-build docker-run lint clean install

BINARY_NAME=miniblue
DOCKER_IMAGE=moabukar/miniblue

build:
	go build -o bin/$(BINARY_NAME) ./cmd/miniblue
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
	go run ./cmd/miniblue

install: build
	cp bin/azlocal /usr/local/bin/azlocal
	cp bin/miniblue /usr/local/bin/miniblue
	@echo "Installed miniblue and azlocal to /usr/local/bin/"
	@echo "Run 'miniblue' to start the server, 'azlocal' to interact with it."

tidy:
	go mod tidy
