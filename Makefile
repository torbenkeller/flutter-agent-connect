BINARY_NAME=fac
BUILD_DIR=bin
GO=go

.PHONY: build build-linux build-all test lint clean run-server

## Build for current platform
build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) .

## Build for Linux (amd64 + arm64)
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GO) build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .

## Build for all platforms
build-all: build build-linux

## Run tests
test:
	$(GO) test ./... -v

## Run linter
lint:
	$(GO) vet ./...

## Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

## Run server (development)
run-server: build
	./$(BUILD_DIR)/$(BINARY_NAME) serve
