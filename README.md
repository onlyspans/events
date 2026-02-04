# events

A microservice that manages domain events for dev-platform.

## Stack
1. Golang 1.25
2. [golang-migrate](https://github.com/golang-migrate/migrate)
3. Docker (see Dockerfile)
4. Native golang worker implementation (without libs)
5. Docker Compose for local development

## How does it work
1. Worker reads Kafka topic
2. Worker writes event to its database
3. Service has handlers to search or export events

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

# Start with Kafka profile
docker compose --profile kafka up
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