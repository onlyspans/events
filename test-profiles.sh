#!/bin/bash
# Test script for Docker Compose profiles
set -e

echo "=========================================="
echo "Testing Docker Compose Profiles"
echo "=========================================="

# Clean up any existing deployment
echo ""
echo "Cleaning up existing deployment..."
docker compose down -v 2>/dev/null || true
docker compose --profile kafka down -v 2>/dev/null || true

# Test 1: Default profile (minimal deployment)
echo ""
echo "=========================================="
echo "Test 1: Default Profile (Minimal)"
echo "=========================================="
echo ""
echo "Starting default profile (postgres + events)..."
docker compose up -d

echo ""
echo "Waiting for services to be healthy..."
sleep 10

echo ""
echo "Checking running containers..."
CONTAINER_COUNT=$(docker compose ps --format json | jq -s 'length')
echo "Running containers: $CONTAINER_COUNT"

if [ "$CONTAINER_COUNT" -eq 2 ]; then
    echo "✅ PASS: Expected 2 containers (postgres, events)"
else
    echo "❌ FAIL: Expected 2 containers, got $CONTAINER_COUNT"
    docker compose ps
    exit 1
fi

echo ""
echo "Verifying Kafka is NOT running..."
if docker compose ps zookeeper 2>/dev/null | grep -q "Up"; then
    echo "❌ FAIL: Zookeeper should not be running in default profile"
    exit 1
fi

if docker compose ps kafka 2>/dev/null | grep -q "Up"; then
    echo "❌ FAIL: Kafka should not be running in default profile"
    exit 1
fi
echo "✅ PASS: Kafka services not running"

echo ""
echo "Checking KAFKA_ENABLED environment variable..."
KAFKA_ENABLED=$(docker compose exec -T events env | grep KAFKA_ENABLED || echo "KAFKA_ENABLED=false")
echo "Environment: $KAFKA_ENABLED"

if echo "$KAFKA_ENABLED" | grep -q "false"; then
    echo "✅ PASS: KAFKA_ENABLED=false in default profile"
else
    echo "❌ FAIL: KAFKA_ENABLED should be false, got: $KAFKA_ENABLED"
    exit 1
fi

echo ""
echo "Testing health endpoints..."
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/healthz)
if [ "$HEALTH_STATUS" -eq 200 ]; then
    echo "✅ PASS: Health endpoint returns 200"
else
    echo "❌ FAIL: Health endpoint returned $HEALTH_STATUS"
    exit 1
fi

# Clean up before next test
echo ""
echo "Stopping default profile..."
docker compose down

# Test 2: Kafka profile with compose override (full deployment)
echo ""
echo "=========================================="
echo "Test 2: Kafka Profile with Override (Full)"
echo "=========================================="
echo ""
echo "Starting kafka profile using compose override..."
docker compose -f compose.yaml -f compose.kafka.yaml up -d

echo ""
echo "Waiting for services to be healthy (Kafka takes longer)..."
sleep 30

echo ""
echo "Checking running containers..."
CONTAINER_COUNT=$(docker compose -f compose.yaml -f compose.kafka.yaml ps --format json | jq -s 'length')
echo "Running containers: $CONTAINER_COUNT"

if [ "$CONTAINER_COUNT" -eq 4 ]; then
    echo "✅ PASS: Expected 4 containers (postgres, events, zookeeper, kafka)"
else
    echo "❌ FAIL: Expected 4 containers, got $CONTAINER_COUNT"
    docker compose -f compose.yaml -f compose.kafka.yaml ps
    exit 1
fi

echo ""
echo "Verifying all Kafka services are running..."
for service in zookeeper kafka; do
    if docker compose -f compose.yaml -f compose.kafka.yaml ps $service | grep -q "Up"; then
        echo "✅ PASS: $service is running"
    else
        echo "❌ FAIL: $service is not running"
        docker compose -f compose.yaml -f compose.kafka.yaml ps $service
        exit 1
    fi
done

echo ""
echo "Checking KAFKA_ENABLED environment variable with compose.kafka.yaml override..."
# When using compose.kafka.yaml override, KAFKA_ENABLED should be true
KAFKA_ENABLED=$(docker compose -f compose.yaml -f compose.kafka.yaml exec -T events env | grep KAFKA_ENABLED || echo "KAFKA_ENABLED=not_found")
echo "Environment with override: $KAFKA_ENABLED"

if echo "$KAFKA_ENABLED" | grep -q "true"; then
    echo "✅ PASS: KAFKA_ENABLED=true with compose override"
else
    echo "❌ FAIL: KAFKA_ENABLED should be true with override, got: $KAFKA_ENABLED"
    exit 1
fi

echo ""
echo "Testing health endpoints..."
HEALTH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/healthz)
if [ "$HEALTH_STATUS" -eq 200 ]; then
    echo "✅ PASS: Health endpoint returns 200"
else
    echo "❌ FAIL: Health endpoint returned $HEALTH_STATUS"
    exit 1
fi

# Clean up
echo ""
echo "Cleaning up kafka profile deployment..."
docker compose -f compose.yaml -f compose.kafka.yaml down

echo ""
echo "=========================================="
echo "All Profile Tests Passed! ✅"
echo "=========================================="
echo ""
echo "Summary:"
echo "  - Default profile: 2 containers (postgres, events) with KAFKA_ENABLED=false"
echo "  - Kafka profile: 4 containers (postgres, events, zookeeper, kafka)"
echo "  - Health endpoints working in both profiles"
