.PHONY: help build build-service build-worker run-service run-worker test clean docker-build migrate-up migrate-down

help:
	@echo "Available targets:"
	@echo "  build          - Build both service and worker binaries"
	@echo "  build-service  - Build service binary"
	@echo "  build-worker   - Build worker binary"
	@echo "  run-service    - Run the service locally"
	@echo "  run-worker     - Run the worker locally"
	@echo "  test           - Run tests"
	@echo "  clean          - Remove build artifacts"
	@echo "  docker-build   - Build Docker image"
	@echo "  migrate-up     - Run database migrations up"
	@echo "  migrate-down   - Run database migrations down"

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
	@echo "Running tests..."
	go test -v -race ./...

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
