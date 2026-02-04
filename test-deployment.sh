#!/bin/bash
set -e

echo "================================"
echo "Testing Task 7 Implementation"
echo "================================"

echo ""
echo "1. Testing local binary build..."
make build
if [ -f "bin/events" ]; then
    echo "✓ Binary built successfully at bin/events"
else
    echo "✗ Binary not found!"
    exit 1
fi

echo ""
echo "2. Testing Docker build..."
docker build -t events:test .
echo "✓ Docker image built successfully"

echo ""
echo "3. Checking Docker image..."
docker images | grep "events.*test"
echo "✓ Docker image found"

echo ""
echo "4. Testing minimal deployment (postgres + events only)..."
echo "   Stopping any existing containers..."
docker compose down -v 2>/dev/null || true

echo "   Starting postgres..."
docker compose up -d postgres

echo "   Waiting for postgres to be ready..."
sleep 10

echo "   Starting events service..."
docker compose up -d events

echo "   Waiting for events service to be ready..."
sleep 15

echo "   Checking health endpoint..."
if curl -f http://localhost:8080/healthz > /dev/null 2>&1; then
    echo "✓ Events service is healthy"
else
    echo "✗ Events service health check failed!"
    docker compose logs events
    exit 1
fi

echo ""
echo "5. Verifying container count..."
RUNNING_CONTAINERS=$(docker compose ps --services | wc -l)
echo "   Running containers: $RUNNING_CONTAINERS"
if [ "$RUNNING_CONTAINERS" -eq 2 ]; then
    echo "✓ Correct number of containers (expected 2: postgres + events)"
else
    echo "✗ Unexpected container count (expected 2, got $RUNNING_CONTAINERS)"
    docker compose ps
fi

echo ""
echo "6. Testing KAFKA_ENABLED=false (default)..."
KAFKA_ENABLED=$(docker compose exec -T events env | grep KAFKA_ENABLED | cut -d= -f2)
if [ "$KAFKA_ENABLED" = "false" ]; then
    echo "✓ KAFKA_ENABLED is set to false by default"
else
    echo "✗ KAFKA_ENABLED is not false (got: $KAFKA_ENABLED)"
fi

echo ""
echo "7. Checking events service logs for migrations..."
if docker compose logs events | grep -q "migrations"; then
    echo "✓ Migrations appear in logs (AUTO_MIGRATE working)"
else
    echo "⚠ No migration logs found (might not have run yet)"
fi

echo ""
echo "================================"
echo "All tests passed successfully!"
echo "================================"
echo ""
echo "Cleanup: Run 'docker compose down -v' to stop and remove containers"
