.PHONY: help build run test test-unit test-integration clean docker-build

help:
	@echo "Available targets:"
	@echo "  build              - Build unified events binary (cmd/events)"
	@echo "  run                - Run the unified events binary locally"
	@echo "  test               - Run all tests (unit + integration)"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-integration   - Run integration tests only (requires Docker)"
	@echo "  clean              - Remove build artifacts"
	@echo "  docker-build       - Build Docker image"

# Unified binary
build:
	@echo "Building unified events binary..."
	go build -o bin/events ./cmd/events

run:
	@echo "Running unified events binary..."
	go run ./cmd/events

test:
	@echo "Running all tests..."
	go test -v -race ./...

test-unit:
	@echo "Running unit tests..."
	go test -v -race ./internal/service/...

test-integration:
	@echo "Running integration tests (requires Docker)..."
	go test -v -race ./internal/repository/...

clean:
	@echo "Cleaning..."
	rm -rf bin/
	go clean

docker-build:
	@echo "Building Docker image..."
	docker build -t events:latest .

.DEFAULT_GOAL := help
