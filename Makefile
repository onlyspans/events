.PHONY: help build build-events build-service build-worker run run-events run-service run-worker test test-unit test-integration clean docker-build migrate-up migrate-down

help:
	@echo "Available targets:"
	@echo "  build              - Build unified events binary (cmd/events)"
	@echo "  build-events       - Build unified events binary (same as build)"
	@echo "  build-service      - Build legacy service binary (deprecated)"
	@echo "  build-worker       - Build legacy worker binary (deprecated)"
	@echo "  run                - Run the unified events binary locally"
	@echo "  run-events         - Run the unified events binary (same as run)"
	@echo "  run-service        - Run the legacy service locally (deprecated)"
	@echo "  run-worker         - Run the legacy worker locally (deprecated)"
	@echo "  test               - Run all tests (unit + integration)"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-integration   - Run integration tests only (requires Docker)"
	@echo "  clean              - Remove build artifacts"
	@echo "  docker-build       - Build Docker image"
	@echo "  migrate-up         - Run database migrations up"
	@echo "  migrate-down       - Run database migrations down"

# New unified binary (default)
build: build-events

build-events:
	@echo "Building unified events binary..."
	go build -o bin/events ./cmd/events

run: run-events

run-events:
	@echo "Running unified events binary..."
	go run ./cmd/events

# Legacy binaries (kept for backward compatibility)
build-service:
	@echo "Building service (deprecated, use 'make build' instead)..."
	go build -o bin/service ./cmd/service

build-worker:
	@echo "Building worker (deprecated, use 'make build' instead)..."
	go build -o bin/worker ./cmd/worker

run-service:
	@echo "Running service (deprecated, use 'make run' instead)..."
	go run ./cmd/service

run-worker:
	@echo "Running worker (deprecated, use 'make run' instead)..."
	go run ./cmd/worker

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
	docker build -t events-service:latest .

migrate-up:
	@echo "Running migrations up..."
	migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/eventlogs?sslmode=disable" up

migrate-down:
	@echo "Running migrations down..."
	migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/eventlogs?sslmode=disable" down

.DEFAULT_GOAL := help
