# Quick Start Guide

Get the Event Logs microservice running locally in 5 minutes.

## Prerequisites

- Go 1.25+ installed
- Docker and Docker Compose installed
- PostgreSQL client (optional, for manual migrations)

## Quick Start with Docker Compose

The fastest way to get everything running:

```bash
# Start all services (PostgreSQL, Kafka, service, worker)
docker compose up

# In another terminal, run migrations
docker compose exec migrations migrate -path /app/migrations \
  -database "postgresql://postgres:postgres@postgres:5432/eventlogs?sslmode=disable" up
```

That's it! The service is now running on http://localhost:8080

## Local Development

### 1. Start Dependencies

```bash
# Start only PostgreSQL and Kafka
docker compose up postgres kafka -d
```

### 2. Run Migrations

```bash
make migrate-up

# Or manually
migrate -path migrations \
  -database "postgresql://postgres:postgres@localhost:5432/eventlogs?sslmode=disable" up
```

### 3. Run the Service

```bash
# Terminal 1 - API Service
make run-service

# Terminal 2 - Kafka Worker
make run-worker
```

## Quick Test

### 1. Check Health

```bash
curl http://localhost:8080/healthz
# Expected: {"status":"UP"}

curl http://localhost:8080/readyz
# Expected: {"database":"connected","status":"UP"}
```

### 2. Get Settings

```bash
curl http://localhost:8080/settings
# Expected: {"retentionPeriodDays":90,"maxExportSize":10000}
```

### 3. Search Events (Empty)

```bash
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{"page":0,"size":20}'
# Expected: {"events":[],"page":0,"size":20,"total":0,"totalPages":0}
```

### 4. Send Test Event via Kafka

```bash
# Produce a message to Kafka
docker compose exec kafka kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic event-logs << EOF
{"timestamp":"2024-02-01T10:00:00Z","user":"john.doe","category":"user","action":"login","project":"dev-platform","environment":"production","tenant":"acme-corp"}
EOF
```

Wait a moment for the worker to process, then search again:

```bash
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{"page":0,"size":20}'
# Expected: Event appears in results
```

### 5. Export to CSV

```bash
curl -X POST http://localhost:8080/events/export \
  -H "Content-Type: application/json" \
  -d '{}' \
  -o events.csv

# Check the file
cat events.csv
```

### 6. Update Settings

```bash
curl -X PUT http://localhost:8080/settings \
  -H "Content-Type: application/json" \
  -d '{"retentionPeriodDays":30,"maxExportSize":5000}'
# Expected: {"retentionPeriodDays":30,"maxExportSize":5000}
```

### 7. View Metrics

```bash
curl http://localhost:8080/metrics | grep event_logs
# Expected: Prometheus metrics for events
```

## Development Workflow

### Build

```bash
# Build both binaries
make build

# Build individually
make build-service
make build-worker
```

### Test

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

### Clean

```bash
make clean
```

## Environment Variables

Create a `.env` file for local development:

```bash
# Server
SERVER_PORT=8080

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=eventlogs

# Kafka
KAFKA_HOST=localhost
KAFKA_PORT=9092
KAFKA_TOPIC=event-logs
KAFKA_GROUP_ID=event-logs-group

# Optional: Kafka Authentication
# KAFKA_USERNAME=
# KAFKA_PASSWORD=

# Event Logs Config
RETENTION_PERIOD_DAYS=90
MAX_EXPORT_SIZE=10000
RETENTION_CRON=0 2 * * *
```

Then run with:

```bash
source .env
make run-service
```

## Project Structure

```
.
├── cmd/
│   ├── service/     # HTTP API server
│   └── worker/      # Kafka consumer
├── internal/
│   ├── config/      # Configuration
│   ├── consumer/    # Kafka consumer
│   ├── domain/      # Domain entities
│   ├── dto/         # Data transfer objects
│   ├── handler/     # HTTP handlers
│   ├── repository/  # Data access
│   └── service/     # Business logic
├── migrations/      # Database migrations
└── Dockerfile       # Multi-stage build
```

## Common Tasks

### View Logs

```bash
# Service logs
docker compose logs -f service

# Worker logs
docker compose logs -f worker

# All logs
docker compose logs -f
```

### Reset Database

```bash
# Down migrations
make migrate-down

# Up migrations
make migrate-up
```

### Debug Kafka

```bash
# List topics
docker compose exec kafka kafka-topics --list --bootstrap-server localhost:9092

# Consume messages
docker compose exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic event-logs \
  --from-beginning
```

### Connect to Database

```bash
docker compose exec postgres psql -U postgres -d eventlogs

# List events
SELECT COUNT(*) FROM events;

# Check settings
SELECT * FROM settings;
```

## Troubleshooting

### Service won't start

```bash
# Check if port 8080 is available
lsof -i :8080

# Check database connection
docker compose exec postgres pg_isready
```

### Worker not consuming messages

```bash
# Check Kafka is running
docker compose ps kafka

# Check consumer group
docker compose exec kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --describe \
  --group event-logs-group
```

### Migrations fail

```bash
# Check database is accessible
docker compose exec postgres psql -U postgres -l

# Force migration version
migrate -path migrations \
  -database "postgresql://postgres:postgres@localhost:5432/eventlogs?sslmode=disable" \
  force 1
```

## Next Steps

- Review [README_MIGRATION.md](README_MIGRATION.md) for architecture details
- Check [MIGRATION_SUMMARY.md](MIGRATION_SUMMARY.md) for migration notes
- Add integration tests in `internal/*/test.go` files
- Configure CI/CD pipeline
- Set up monitoring and alerting

## Support

For issues or questions:
1. Check logs: `docker compose logs`
2. Verify health: `curl http://localhost:8080/readyz`
3. Review configuration in `compose.yaml`
4. Check database connectivity
5. Validate Kafka consumer group status
