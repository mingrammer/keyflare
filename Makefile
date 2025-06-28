.PHONY: build clean test lint cover docker help

BINARY_NAME = keyflare
VERSION = 0.1.0
BUILD_DATE = $(shell date +%Y-%m-%d-%H:%M:%S)
COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE) -X main.commit=$(COMMIT)"

help:
	@echo "Available commands:"
	@echo "  test        - Run tests"
	@echo "  lint        - Run linters"
	@echo "  cover       - Run tests with coverage"
	@echo "  help        - Show this help message"

build-exporter:
	@echo "Building KeyFlare exporter..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME)-exporter ./cmd/exporter

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Running linters..."
	golangci-lint run

cover:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Install development dependencies
dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
