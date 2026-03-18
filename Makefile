BINARY_NAME=fac
BUILD_DIR=bin
GO=go
GOLANGCI_LINT_VERSION=v1.64.8
GOLANGCI_LINT=$(shell which golangci-lint 2>/dev/null || echo $(shell $(GO) env GOPATH)/bin/golangci-lint)

.PHONY: build build-linux build-all test lint lint-fix clean run-server

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
	$(GO) test ./...

## Run linter (installs golangci-lint if missing)
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run ./...

## Run linter and auto-fix where possible
lint-fix: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run --fix ./...

$(GOLANGCI_LINT):
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

## Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

## Run server (development)
run-server: build
	./$(BUILD_DIR)/$(BINARY_NAME) serve
