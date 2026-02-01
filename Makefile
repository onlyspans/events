.PHONY: help build build-service build-worker run-service run-worker test test-unit test-integration clean docker-build migrate-up migrate-down

help:
	@echo "Available targets:"
	@echo "  build              - Build both service and worker binaries"
	@echo "  build-service      - Build service binary"
	@echo "  build-worker       - Build worker binary"
	@echo "  run-service        - Run the service locally"
	@echo "  run-worker         - Run the worker locally"
	@echo "  test               - Run all tests (unit + integration)"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-integration   - Run integration tests only (requires Docker)"
	@echo "  clean              - Remove build artifacts"
	@echo "  docker-build       - Build Docker image"
	@echo "  migrate-up         - Run database migrations up"
	@echo "  migrate-down       - Run database migrations down"

build: build-service build-worker

build-service:
	@echo "Building service..."
	go build -o bin/service ./cmd/service

build-worker:
	@echo "Building worker..."
	go build -o bin/worker ./cmd/worker

run-service:
	@echo "Running service..."
	go run ./cmd/service

run-worker:
	@echo "Running worker..."
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
