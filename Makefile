.PHONY: help build run test test-unit test-integration test-coverage clean \
        docker-build compose-up compose-down compose-up-kafka lint fmt deps \
        run-race install-tools version

# Variables
BINARY_NAME=events
BUILD_DIR=bin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X github.com/onlyspans/events/pkg/version.Version=${VERSION} \
                  -X github.com/onlyspans/events/pkg/version.GitCommit=${COMMIT} \
                  -X github.com/onlyspans/events/pkg/version.BuildTime=${BUILD_TIME}"

help:
	@echo "Available targets:"
	@echo "  build              - Build server binary with version info"
	@echo "  run                - Run the server locally"
	@echo "  run-race           - Run with race detector"
	@echo "  test               - Run all tests (unit + integration)"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-integration   - Run integration tests only (requires Docker)"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  lint               - Run golangci-lint"
	@echo "  fmt                - Format code with gofmt and goimports"
	@echo "  deps               - Download and verify dependencies"
	@echo "  clean              - Remove build artifacts"
	@echo "  docker-build       - Build Docker image"
	@echo "  compose-up         - Start minimal stack (postgres + events)"
	@echo "  compose-down       - Stop compose stack"
	@echo "  compose-up-kafka   - Start with Kafka enabled"
	@echo "  install-tools      - Install development tools"
	@echo "  version            - Display version information"

# Build with version information
build:
	@echo "Building ${BINARY_NAME} ${VERSION}..."
	@mkdir -p ${BUILD_DIR}
	CGO_ENABLED=0 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/server

# Run server
run:
	@echo "Running server..."
	go run ./cmd/server

# Run with race detector
run-race:
	@echo "Running with race detector..."
	go run -race ./cmd/server

# Run all tests
test:
	@echo "Running all tests..."
	go test -v -race ./...

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v -race ./internal/service/...

# Run integration tests
test-integration:
	@echo "Running integration tests (requires Docker)..."
	go test -v -race ./internal/repository/...

# Test with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Linting
lint:
	@echo "Running linters..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Run 'make install-tools'"; exit 1; }
	golangci-lint run ./...

# Code formatting
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@command -v goimports >/dev/null 2>&1 && goimports -w . || echo "goimports not found, skipping"

# Dependency management
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	go mod verify

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf ${BUILD_DIR}
	rm -f coverage.out coverage.html
	go clean

# Docker operations
docker-build:
	@echo "Building Docker image..."
	docker build -f deployments/docker/Dockerfile -t ${BINARY_NAME}:${VERSION} -t ${BINARY_NAME}:latest .

# Compose operations
compose-up:
	@echo "Starting minimal stack..."
	docker compose -f deployments/compose/compose.yaml up -d

compose-down:
	@echo "Stopping compose stack..."
	docker compose -f deployments/compose/compose.yaml down

compose-up-kafka:
	@echo "Starting with Kafka..."
	docker compose -f deployments/compose/compose.yaml -f deployments/compose/compose.kafka.yaml up -d

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Version information
version:
	@echo "Version:    ${VERSION}"
	@echo "Commit:     ${COMMIT}"
	@echo "Build Time: ${BUILD_TIME}"

.DEFAULT_GOAL := help
