.PHONY: build clean test install help run

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Binary name
BINARY := codegate

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY) $(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY) .
	@echo "✓ Build complete: ./$(BINARY)"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY)
	rm -f coverage.out
	@echo "✓ Clean complete"

## test: Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests complete"

## test-coverage: Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out

## install: Install the binary to $GOPATH/bin
install:
	@echo "Installing $(BINARY) to $$GOPATH/bin..."
	go install $(LDFLAGS) .
	@echo "✓ Installed: $$GOPATH/bin/$(BINARY)"

## run: Build and run the binary
run: build
	./$(BINARY) $(ARGS)

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Format complete"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "✓ Vet complete"

## lint: Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. See: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run
	@echo "✓ Lint complete"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies updated"

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' Makefile | column -t -s ':' | sed -e 's/^/ /'

# Examples
.PHONY: example-review example-json example-preview
## example-review: Run review on staged changes (table format)
example-review: build
	./$(BINARY) review

## example-json: Run review with JSON output
example-json: build
	./$(BINARY) review --format json

## example-preview: Run review with browser preview
example-preview: build
	./$(BINARY) review --preview
