
# ==============================================================================
# az-storage-mock - Makefile
# ==============================================================================

# Variables


APP_NAME        := az-storage-mock

BUILD_DIR       := bin
BINARY_PATH     := $(BUILD_DIR)/$(APP_NAME)
MAIN_FILE       := cmd/main.go

# Go commands
GO              := go
GOTEST          := $(GO) test
GOFMT           := $(GO) fmt
GOVET           := $(GO) vet

.PHONY: all help build run test test-cover fmt vet tidy clean

# Default target
all: help

## help: Display this help message
help:
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "  build           Compile the Go binary to $(BUILD_DIR)"
	@echo "  run             Run the application locally (go run main.go)"
	@echo "  test            Run unit tests"
	@echo "  test-cover      Run unit tests with HTML coverage report"
	@echo "  fmt             Format Go source files"
	@echo "  vet             Run go vet static code analysis"
	@echo "  tidy            Tidy and verify Go module dependencies"
	@echo "  clean           Remove build artifacts and coverage files"


## build: Compile Go binary
build: tidy
	@echo  Building binary $(BINARY_PATH)...
	@$(GO) build -ldflags="-s -w" -o $(BINARY_PATH) $(MAIN_FILE)
	@echo  Build complete: $(BINARY_PATH)

## run: Run application locally
run: tidy
	@echo  Running $(APP_NAME) locally...
	@$(GO) run $(MAIN_FILE)

## test: Run tests
test:
	@echo  Running tests...
	@$(GOTEST) -v ./...

## test-cover: Run tests with coverage output
test-cover:
	@echo  Running tests with coverage...
	@$(GOTEST) -v -coverprofile=coverage.out ./...
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo  Coverage report generated at coverage.html

## fmt: Format source code
fmt:
	@echo  Formatting code...
	@$(GOFMT) ./...

## vet: Run go vet
vet:
	@echo  Running go vet...
	@$(GOVET) ./...

## tidy: Run go mod tidy
tidy:
	@echo  Tidying dependencies...
	@$(GO) mod tidy

## clean: Remove build output and coverage files
clean:

	@echo  Cleaning build artifacts...
	@-rm -rf $(BUILD_DIR) coverage.out coverage.html 2>NUL || del /f /q $(BUILD_DIR) coverage.out coverage.html 2>NUL || true
	@echo  Clean complete.

