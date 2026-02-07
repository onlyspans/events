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

### Unified Binary System

The service consists of a single binary (`cmd/events/main.go`) that combines all functionality:

1. **HTTP API Server**: REST API endpoints for event ingestion, searches, exports, settings management, health checks, and metrics. Runs on port 8080 by default.

2. **Kafka Consumer (Optional)**: Background consumer that reads events from Kafka and persists them to PostgreSQL. Enabled via `KAFKA_ENABLED=true`. Uses batch processing (100 events or 500ms timeout) for efficiency.

3. **Embedded Migrations**: Database migrations are embedded in the binary and run automatically on startup when `AUTO_MIGRATE=true` (default). No separate migration container needed.

**Deployment Modes:**
- **Minimal (Default)**: HTTP API only with direct event ingestion endpoints. Requires only PostgreSQL.
- **Full (Kafka-enabled)**: HTTP API + Kafka consumer. Requires PostgreSQL and Kafka infrastructure.

### Layer Structure

```
cmd/
  └── events/      - Unified binary entry point (HTTP API + optional Kafka consumer)
internal/
  ├── config/      - Configuration loading from environment variables/.env
  ├── consumer/    - Kafka consumer with SASL/SCRAM authentication support
  ├── domain/      - Core entities (Event, Settings) with JSONB mapping
  ├── dto/         - Data transfer objects for API requests/responses
  ├── handler/     - HTTP handlers (events, settings, health, ingestion)
  ├── migrator/    - Embedded migration runner using golang-migrate (migrations are .sql files in this directory)
  ├── repository/  - Database access layer (PostgreSQL)
  └── service/     - Business logic including retention service with cron
```

### Key Design Patterns

- **Embedded Migrations**: Database migrations are embedded in the binary using `//go:embed` and run automatically on startup. PostgreSQL advisory locks prevent concurrent migration execution.
- **Optional Kafka Support**: Kafka consumer is conditionally started based on `KAFKA_ENABLED` flag, allowing for simplified deployments without messaging infrastructure.
- **HTTP Event Ingestion**: Events can be ingested directly via HTTP endpoints (`POST /events/ingest`, `POST /events/ingest/batch`) as an alternative to Kafka.
- **Manual Offset Management**: The Kafka consumer manually commits offsets after successful batch processing to ensure at-least-once delivery semantics.
- **Batch Processing**: Events are processed in batches of 100 or every 500ms to optimize database writes and Kafka commits.
- **JSONB Details**: The `details` field uses PostgreSQL JSONB with a GIN index for flexible querying of nested data.
- **Graceful Shutdown**: The binary handles SIGINT/SIGTERM with context cancellation and 30-second timeouts for all goroutines.
- **Connection Pooling**: Database connections configured with max 25 open, 5 idle, 5-minute lifetime.

## Development Commands

### Building

```bash
make build              # Build unified events binary
```

### Running Locally

**Minimal Mode (HTTP API only, requires PostgreSQL only):**
```bash
# Start PostgreSQL
docker compose up postgres -d

# Run the service (migrations run automatically)
KAFKA_ENABLED=false make run
# Or simply:
make run                # KAFKA_ENABLED defaults to false
```

**Full Mode (HTTP API + Kafka consumer):**
```bash
# Start dependencies
docker compose up postgres kafka -d

# Run the service with Kafka enabled
KAFKA_ENABLED=true make run
```

### Testing

```bash
make test               # Run all tests with race detector
```

### Database Migrations

**Automatic Migrations (Default)**
Migrations are embedded in the binary and run automatically on startup when `AUTO_MIGRATE=true` (default).

```bash
# Migrations run automatically when starting the service
make run        # Runs migrations, then starts HTTP API

# To disable automatic migrations
AUTO_MIGRATE=false make run
```

### Docker

```bash
# Minimal deployment (2 containers: postgres + events)
docker compose up       # Starts PostgreSQL and events service
                        # Migrations run automatically on startup (embedded in binary)
                        # Kafka is NOT started by default (KAFKA_ENABLED=false)

# Full deployment with Kafka (4 containers: postgres + events + zookeeper + kafka)
# Option 1: Using compose override (recommended - automatically sets KAFKA_ENABLED=true)
docker compose -f compose.yaml -f compose.kafka.yaml up

# Option 2: Using profiles (requires KAFKA_ENABLED=true in .env)
docker compose --profile kafka up

# Build Docker image
make docker-build       # Build Docker image locally (builds events:latest)
```

## Configuration

All configuration is via environment variables with defaults in `internal/config/config.go`. A `.env` file is auto-loaded if present.

**Critical Configuration:**
- **Feature Flags:**
  - `KAFKA_ENABLED` (default: false) - Enable Kafka consumer
  - `AUTO_MIGRATE` (default: true) - Run migrations automatically on startup
- **Database connection:** `POSTGRES_*` variables
- **Kafka configuration** (only used if `KAFKA_ENABLED=true`):
  - `KAFKA_HOST`, `KAFKA_PORT`, `KAFKA_TOPIC`
  - `KAFKA_USERNAME`, `KAFKA_PASSWORD` (optional, enables SASL/SCRAM-SHA-512)
- **Event retention:** `RETENTION_PERIOD_DAYS` (default: 90), `RETENTION_CRON` (default: "0 2 * * *")
- **Export limits:** `MAX_EXPORT_SIZE` (default: 10000)

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

**Event Ingestion:**
- `POST /events/ingest` - Ingest single event directly via HTTP
- `POST /events/ingest/batch` - Ingest multiple events in batch (max 100, returns 207 Multi-Status)

**Event Management:**
- `POST /events` - Search events with pagination and filters
- `POST /events/export` - Export events to CSV (streams response)

**Settings:**
- `GET /settings` - Get current retention and export settings
- `PUT /settings` - Update retention and export settings

**Health & Metrics:**
- `GET /healthz` - Liveness probe (always returns 200)
- `GET /readyz` - Readiness probe (checks database connectivity)
- `GET /metrics` - Prometheus metrics

## Kafka Consumer Behavior (Optional)

The Kafka consumer is only active when `KAFKA_ENABLED=true`. When enabled:

- Consumes from topic specified in `KAFKA_TOPIC` (default: "events")
- Consumer group: `KAFKA_GROUP_ID` (default: "events-group")
- Batches up to 100 messages or 500ms timeout
- Manual offset commits only after successful batch persistence
- Failed messages are logged and skipped (offset still committed to avoid blocking)
- Prometheus metrics track received, processed, and failed event counts

**Alternative to Kafka:** Use HTTP ingestion endpoints (`POST /events/ingest`, `POST /events/ingest/batch`) for direct event submission without Kafka infrastructure.

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
3. Create migration in `migrations/` to add column and index if needed (e.g., `000003_add_new_field.up.sql` and `000003_add_new_field.down.sql`)
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
- Migrations loaded via `internal/migrations` package from `migrations/` directory
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
- Migrations are stored in the `migrations/` directory at project root
- All `.up.sql` and `.down.sql` files are loaded from the filesystem
- Files execute in alphabetical order (works because migrations are prefixed: 000001, 000002, etc.)
- Adding new migrations requires zero test code changes - just add new .sql files to `migrations/`
- Migrations use golang-migrate's standard file-based approach

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
