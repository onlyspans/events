# events

A microservice that manages domain events for dev-platform.

## Stack
1. Golang 1.25
2. [golang-migrate](https://github.com/golang-migrate/migrate) (embedded migrations)
3. PostgreSQL 17 with JSONB support
4. Apache Kafka (optional) via IBM Sarama
5. Docker Compose for local development
6. Prometheus metrics

## Architecture

The service consists of a **unified single binary** (`cmd/events/main.go`) that combines all functionality:

### Components

1. **HTTP API Server**: REST API endpoints for event ingestion, searches, exports, settings management, health checks, and metrics. Runs on port 8080 by default.

2. **Kafka Consumer (Optional)**: Background consumer that reads events from Kafka and persists them to PostgreSQL. Enabled via `KAFKA_ENABLED=true`. Uses batch processing (100 events or 500ms timeout) for efficiency.

3. **Embedded Migrations**: Database migrations are embedded in the binary and run automatically on startup when `AUTO_MIGRATE=true` (default). No separate migration container needed.

### Deployment Modes

- **Minimal (Default)**: HTTP API only with direct event ingestion endpoints. Requires only PostgreSQL (2 containers).
- **Full (Kafka-enabled)**: HTTP API + Kafka consumer. Requires PostgreSQL and Kafka infrastructure (4 containers).

### How it Works

**Minimal Mode (HTTP Ingestion):**
1. Client sends events via HTTP POST to `/events/ingest` or `/events/ingest/batch`
2. Service validates and stores events directly in PostgreSQL
3. Events are searchable via `/events` endpoint and exportable via `/events/export`

**Full Mode (Kafka + HTTP):**
1. Events can be published to Kafka topic (default: "events")
2. Kafka consumer reads events in batches and persists to PostgreSQL
3. Additionally, events can be ingested directly via HTTP endpoints
4. All events are searchable and exportable regardless of ingestion method

## Quick Start

```bash
# 1. Set up configuration (only POSTGRES_PASSWORD required)
cp .env.example .env
# Edit .env and set POSTGRES_PASSWORD=your-secure-password

# 2. Start services with Docker Compose
docker compose up

# 3. Service runs on http://localhost:8080
curl http://localhost:8080/healthz
```

The service will automatically:
- Run database migrations on startup
- Start HTTP API server on port 8080
- Serve event ingestion, search, and export endpoints

## Configuration

The service uses environment variables for configuration with sensible defaults. Only **one required variable**: `POSTGRES_PASSWORD`.

### Configuration Files

- **`.env.example`** - Minimal configuration template (recommended starting point)
  - Only 1 required variable: `POSTGRES_PASSWORD`
  - All other settings have sensible defaults
  - Perfect for simple deployments without Kafka

- **`.env.kafka.example`** - Configuration template for Kafka-enabled deployments
  - Includes Kafka connection settings
  - Shows SASL authentication setup
  - Advanced consumer tuning options

### Quick Setup

**Minimal deployment (HTTP API only, no Kafka):**
```bash
# Copy the minimal example
cp .env.example .env

# Edit and set your database password
nano .env  # Change POSTGRES_PASSWORD value

# Start the service
docker compose up
```

**Kafka-enabled deployment:**
```bash
# Copy the Kafka-enabled example
cp .env.kafka.example .env

# Edit and configure:
# 1. Set POSTGRES_PASSWORD
# 2. Update KAFKA_HOST and KAFKA_PORT if needed
# 3. Configure KAFKA_USERNAME and KAFKA_PASSWORD for production
nano .env

# Start with Kafka using compose override (recommended)
docker compose -f compose.yaml -f compose.kafka.yaml up

# Alternative: Use profile (requires KAFKA_ENABLED=true in .env)
# docker compose --profile kafka up
```

### Required vs Optional Variables

**Required (must be set):**
- `POSTGRES_PASSWORD` - Database password (no default for security)

**Optional (defaults work for most cases):**
- `SERVER_PORT` (default: 8080)
- `POSTGRES_HOST` (default: localhost)
- `POSTGRES_PORT` (default: 5432)
- `POSTGRES_USER` (default: postgres)
- `POSTGRES_DB` (default: events)
- `KAFKA_ENABLED` (default: false)
- `AUTO_MIGRATE` (default: true)
- `RETENTION_PERIOD_DAYS` (default: 90)
- `MAX_EXPORT_SIZE` (default: 10000)

See `.env.example` or `.env.kafka.example` for complete configuration reference with descriptions.

### Environment Variables Reference

Complete list of all configuration options:

| Variable | Required | Default        | Description |
|----------|----------|----------------|-------------|
| **Server Configuration** ||                ||
| `SERVER_PORT` | No | `8080`         | HTTP server port |
| **Database Configuration** ||                ||
| `POSTGRES_HOST` | No | `localhost`    | PostgreSQL host |
| `POSTGRES_PORT` | No | `5432`         | PostgreSQL port |
| `POSTGRES_USER` | No | `postgres`     | PostgreSQL username |
| `POSTGRES_PASSWORD` | **Yes** | *(none)*       | PostgreSQL password |
| `POSTGRES_DB` | No | `events`       | PostgreSQL database name |
| **Feature Flags** ||                ||
| `KAFKA_ENABLED` | No | `false`        | Enable Kafka consumer |
| `AUTO_MIGRATE` | No | `true`         | Run migrations on startup |
| **Kafka Configuration** (only used if `KAFKA_ENABLED=true`) ||                ||
| `KAFKA_HOST` | No | `localhost`    | Kafka broker host |
| `KAFKA_PORT` | No | `9092`         | Kafka broker port |
| `KAFKA_TOPIC` | No | `events`       | Kafka topic to consume |
| `KAFKA_GROUP_ID` | No | `events-group` | Kafka consumer group ID |
| `KAFKA_USERNAME` | No | *(none)*       | Kafka SASL username (enables SASL if set) |
| `KAFKA_PASSWORD` | No | *(none)*       | Kafka SASL password |
| `KAFKA_CONSUMER_OFFSET` | No | `newest`       | Initial offset: `newest` or `oldest` |
| **Event Retention** ||                ||
| `RETENTION_PERIOD_DAYS` | No | `90`           | Days to keep events before deletion |
| `RETENTION_CRON` | No | `0 2 * * *`    | Cron schedule for retention job (daily at 2 AM) |
| **Export Limits** ||                ||
| `MAX_EXPORT_SIZE` | No | `10000`        | Maximum events per CSV export |

For detailed configuration examples, see [docs/CONFIGURATION.md](docs/CONFIGURATION.md).

## Deployment Profiles

Docker Compose supports two deployment profiles for different use cases:

### Default Profile (Minimal Deployment)

The default profile runs a minimal 2-container setup without Kafka:

```bash
# Start minimal deployment (postgres + events only)
docker compose up -d

# Check running containers (should show 2)
docker compose ps
```

**Containers:**
- `postgres` - PostgreSQL database
- `events` - Unified events service (HTTP API only)

**Features:**
- HTTP event ingestion endpoints
- Event search and export
- Settings management
- Database migrations (automatic)
- **KAFKA_ENABLED=false** (Kafka consumer disabled)

**Use cases:**
- Simple deployments
- Development without Kafka
- Testing HTTP ingestion
- Minimal resource usage

### Kafka Profile (Full Deployment)

The kafka profile adds Kafka infrastructure for event streaming. There are two ways to enable Kafka:

**Option 1: Using profiles (manual KAFKA_ENABLED required)**
```bash
# Start with kafka profile
docker compose --profile kafka up -d

# Manually set KAFKA_ENABLED=true via environment variable or .env file
# Add to .env: KAFKA_ENABLED=true

# Check running containers (should show 4)
docker compose --profile kafka ps
```

**Option 2: Using compose override file (recommended)**
```bash
# Start with compose override (automatically sets KAFKA_ENABLED=true)
docker compose -f compose.yaml -f compose.kafka.yaml up -d

# Check running containers (should show 4)
docker compose -f compose.yaml -f compose.kafka.yaml ps
```

**Containers:**
- `postgres` - PostgreSQL database
- `events` - Unified events service (HTTP API + Kafka consumer)
- `zookeeper` - Kafka coordination service
- `kafka` - Apache Kafka message broker

**Features:**
- All default profile features
- Kafka consumer for event streaming
- Background event processing from Kafka topic
- **KAFKA_ENABLED=true** (Kafka consumer enabled)

**Use cases:**
- Production deployments
- Event streaming architectures
- High-throughput ingestion
- Integration with existing Kafka infrastructure

### Switching Between Profiles

```bash
# Stop current deployment
docker compose down

# Start with different profile
docker compose up -d                    # Default (no Kafka)
docker compose --profile kafka up -d    # Kafka-enabled

# Check which services are running
docker compose ps
```

**Note:** When using the kafka profile, the events service automatically sets `KAFKA_ENABLED=true` and waits for Kafka to be healthy before starting.

## API Endpoints

The service provides a comprehensive REST API for event management:

### Event Ingestion

**POST /events/ingest** - Ingest single event via HTTP

Request body:
```json
{
  "user_name": "john.doe",
  "category": "document",
  "action": "create",
  "document_name": "project-plan.pdf",
  "project": "awesome-project",
  "environment": "production",
  "tenant": "acme-corp",
  "details": {
    "size_bytes": 1024,
    "mime_type": "application/pdf"
  }
}
```

Response (201 Created):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Required fields:** `user_name`, `category`, `action`

**Optional fields:** `document_name`, `project`, `environment`, `tenant`, `correlation_id`, `trace_id`, `details` (JSONB object)

---

**POST /events/ingest/batch** - Ingest multiple events in batch (max 100)

Request body:
```json
{
  "events": [
    {
      "user_name": "john.doe",
      "category": "document",
      "action": "create"
    },
    {
      "user_name": "jane.smith",
      "category": "document",
      "action": "update"
    }
  ]
}
```

Response (207 Multi-Status):
```json
{
  "success_count": 1,
  "failure_count": 1,
  "errors": [
    {
      "index": 1,
      "error": "validation failed: user_name is required"
    }
  ]
}
```

**Note:** Batch endpoint processes all events (doesn't stop on first error) and returns partial success status.

### Event Search

**POST /events** - Search events with filters and pagination

Request body:
```json
{
  "user_name": "john.doe",
  "category": "document",
  "action": "create",
  "project": "awesome-project",
  "environment": "production",
  "start_date": "2024-01-01T00:00:00Z",
  "end_date": "2024-12-31T23:59:59Z",
  "page": 1,
  "page_size": 50
}
```

Response (200 OK):
```json
{
  "events": [...],
  "total": 150,
  "page": 1,
  "page_size": 50,
  "total_pages": 3
}
```

**All filter fields are optional.** Empty request returns all events with default pagination (page 1, 50 items).

### Event Export

**POST /events/export** - Export events to CSV

Uses same filter parameters as search endpoint. Streams CSV response with all matching events (up to `max_export_size` limit).

Response headers:
```
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="events_export.csv"
```

### Settings Management

**GET /settings** - Get current retention and export settings

Response:
```json
{
  "retention_period_days": 90,
  "max_export_size": 10000
}
```

**PUT /settings** - Update retention and export settings

Request body:
```json
{
  "retention_period_days": 180,
  "max_export_size": 50000
}
```

### Health & Metrics

**GET /healthz** - Liveness probe (always returns 200)

**GET /readyz** - Readiness probe (checks database connectivity)

**GET /metrics** - Prometheus metrics endpoint

Exposed metrics:
- `events_received_total` - Events received from Kafka
- `events_batches_processed_total` - Batches successfully persisted
- `events_failed_total` - Events that failed to process
- `events_retention_deleted_total` - Events deleted by retention job

For detailed API documentation, see [docs/INGESTION.md](docs/INGESTION.md).