# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go microservice that manages domain events for dev-platform. It consumes events from Kafka, stores them in PostgreSQL, and provides HTTP endpoints for searching and exporting events.

**Key Technology Stack:**
- Go 1.25
- PostgreSQL with JSONB support
- Kafka (via IBM Sarama)
- golang-migrate for database migrations
- Prometheus metrics
- Docker Compose for local development

## Architecture

### Two-Binary System

The service consists of two separate binaries:

1. **HTTP Service** (`cmd/service/main.go`): REST API server that handles event searches, exports, settings management, health checks, and metrics. Runs on port 8080 by default.

2. **Kafka Worker** (`cmd/worker/main.go`): Background consumer that reads events from Kafka and persists them to PostgreSQL. Uses batch processing (100 events or 500ms timeout) for efficiency.

### Layer Structure

```
cmd/          - Entry points (service and worker binaries)
internal/
  ├── config/      - Configuration loading from environment variables/.env
  ├── consumer/    - Kafka consumer with SASL/SCRAM authentication support
  ├── domain/      - Core entities (Event, Settings) with JSONB mapping
  ├── dto/         - Data transfer objects for API requests/responses
  ├── handler/     - HTTP handlers (events, settings, health)
  ├── repository/  - Database access layer (PostgreSQL)
  └── service/     - Business logic including retention service with cron
migrations/   - Database schema migrations (golang-migrate format)
```

### Key Design Patterns

- **Manual Offset Management**: The Kafka consumer manually commits offsets after successful batch processing to ensure at-least-once delivery semantics.
- **Batch Processing**: Events are processed in batches of 100 or every 500ms to optimize database writes and Kafka commits.
- **JSONB Details**: The `details` field uses PostgreSQL JSONB with a GIN index for flexible querying of nested data.
- **Graceful Shutdown**: Both binaries handle SIGINT/SIGTERM with context cancellation and 30-second timeouts.
- **Connection Pooling**: Database connections configured with max 25 open, 5 idle, 5-minute lifetime.

## Development Commands

### Building

```bash
make build              # Build both service and worker
make build-service      # Build service only
make build-worker       # Build worker only
```

### Running Locally (Requires PostgreSQL and Kafka Running)

```bash
# Start dependencies first
docker compose up postgres kafka -d

# Run migrations
make migrate-up

# Run binaries (in separate terminals)
make run-service        # Starts HTTP API on :8080
make run-worker         # Starts Kafka consumer
```

### Testing

```bash
make test               # Run all tests with race detector
```

### Database Migrations

```bash
make migrate-up         # Apply all migrations
make migrate-down       # Rollback all migrations
```

### Docker

```bash
docker compose up       # Start all services (postgres, kafka, migrations, service, worker)
make docker-build       # Build Docker image locally
```

## Configuration

All configuration is via environment variables with defaults in `internal/config/config.go`. A `.env` file is auto-loaded if present.

**Critical Configuration:**
- Database connection: `POSTGRES_*` variables
- Kafka broker: `KAFKA_HOST`, `KAFKA_PORT`, `KAFKA_TOPIC`
- Kafka authentication: `KAFKA_USERNAME`, `KAFKA_PASSWORD` (optional, enables SASL/SCRAM-SHA-512)
- Event retention: `RETENTION_PERIOD_DAYS` (default: 90), `RETENTION_CRON` (default: "0 2 * * *")
- Export limits: `MAX_EXPORT_SIZE` (default: 10000)

## Database Schema

### Events Table

Primary table for storing all domain events with extensive indexing:
- UUID primary key with `gen_random_uuid()` default
- Indexed fields: timestamp, user_name, category, action, document_name, project, environment, tenant, correlation_id, trace_id
- JSONB `details` field with GIN index for nested data queries
- All timestamps use `TIMESTAMP WITH TIME ZONE`

### Settings Table

Single-row table for runtime configuration:
- `retention_period_days`: How long to keep events (updated via API)
- `max_export_size`: Maximum events per CSV export (updated via API)

## API Endpoints

- `POST /events` - Search events with pagination and filters
- `POST /events/export` - Export events to CSV (streams response)
- `GET /settings` - Get current retention and export settings
- `PUT /settings` - Update retention and export settings
- `GET /healthz` - Liveness probe (always returns 200)
- `GET /readyz` - Readiness probe (checks database connectivity)
- `GET /metrics` - Prometheus metrics

## Kafka Consumer Behavior

- Consumes from topic specified in `KAFKA_TOPIC` (default: "event-logs")
- Consumer group: `KAFKA_GROUP_ID` (default: "event-logs-group")
- Batches up to 100 messages or 500ms timeout
- Manual offset commits only after successful batch persistence
- Failed messages are logged and skipped (offset still committed to avoid blocking)
- Prometheus metrics track received, processed, and failed event counts

## Retention Service

The retention service runs as a cron job within the service binary:
- Default schedule: Daily at 2 AM (`0 2 * * *`)
- Deletes events older than `retention_period_days` setting
- Uses `context.Background()` with no timeout for large deletions
- Logs deleted count on each run

## Common Patterns

### Adding a New Event Field

1. Update `domain.Event` struct in `internal/domain/event.go`
2. Update `dto.EventDTO` in `internal/dto/event.go` if exposed in API
3. Create migration in `migrations/` to add column and index if needed
4. Update repository queries in `internal/repository/event_repository.go`

### Modifying Search Filters

Search logic is in `repository.BuildSearchQuery()` at `internal/repository/event_repository.go`. Filters are built dynamically based on non-empty `dto.EventSearchRequest` fields.

### Changing Batch Size

Kafka batch size is hardcoded to 100 in `consumer.go:158`. Increasing it improves throughput but increases memory usage and commit latency.

## Testing Strategy

The project uses a two-tier testing approach:

### Unit Tests (`internal/service/*_test.go`)
- Use mocked repositories for fast, isolated tests
- Test business logic and DTO conversions
- No external dependencies required

### Integration Tests (`internal/repository/*_integration_test.go`)
- Use **testcontainers-go** to spin up real PostgreSQL containers
- Test database queries, indexes, and JSONB operations
- Migrations loaded via `postgres.WithInitScripts()` - uses actual migration files from `migrations/` directory
- Containers are automatically created and destroyed per test
- Requires Docker daemon running locally

**Running tests:**
```bash
# Unit tests only
go test ./internal/service/...

# Integration tests only (requires Docker)
go test ./internal/repository/...

# All tests
make test
```

**Benefits of testcontainers:**
- Tests run against the same PostgreSQL version as production (17-alpine)
- No need for manual database setup or cleanup
- True isolation between test runs
- Catches SQL compatibility issues early
- Uses actual migration files (single source of truth)

**How migrations work in tests:**
- Migration files are auto-discovered using `filepath.Glob("*.up.sql")`
- All `.up.sql` files from `migrations/` directory are automatically loaded
- Files execute in alphabetical order (works because migrations are prefixed: 000001, 000002, etc.)
- Adding new migrations requires zero test code changes
- Migrations run before container is marked "ready"
- No duplication of SQL - tests use the same files as production

## CI/CD

GitHub Actions workflow (`.github/workflows/test.yml`) runs on pull requests to main:
- Sets up Go 1.25 (ubuntu-latest runners have Docker pre-installed)
- Runs unit tests with mocked dependencies
- Runs integration tests with testcontainers (spins up PostgreSQL automatically)
- Combines coverage reports from both test suites
- **Posts coverage report as PR comment** showing overall coverage and changes
- Uploads coverage to Codecov (optional)

The coverage report comment shows:
- Overall test coverage percentage
- Coverage change compared to base branch
- Per-package coverage breakdown
- Links to detailed coverage reports

## Logging

All components use `log/slog` with JSON output to stdout:
- Service: Logs HTTP requests with method, path, status, duration, remote_addr
- Worker: Logs batch processing with size and offset information
- Structured logging with key-value pairs for easy parsing

## Metrics

Prometheus metrics exposed on `/metrics`:
- `events_received_total` - Events received from Kafka
- `events_batches_processed_total` - Batches successfully persisted
- `events_failed_total` - Events that failed to process
- `events_retention_deleted_total` - Events deleted by retention job
