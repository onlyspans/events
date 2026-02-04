# Configuration Guide

This document provides comprehensive information about configuring the events service.

## Overview

The events service uses **environment variables** for all configuration with sensible defaults. Configuration can be provided via:

1. **Environment variables** - Direct shell export or Docker Compose
2. **`.env` file** - Automatically loaded from working directory
3. **Docker Compose** - Via `environment:` section or `env_file:`

## Quick Start

### Minimal Configuration

Only **one required variable**: `POSTGRES_PASSWORD`

```bash
# Copy the minimal example
cp .env.example .env

# Edit and set your database password
nano .env  # Change POSTGRES_PASSWORD value

# Start the service
docker compose up
```

### Kafka-Enabled Configuration

```bash
# Copy the Kafka-enabled example
cp .env.kafka.example .env

# Edit and configure:
# 1. Set POSTGRES_PASSWORD
# 2. Update KAFKA_HOST and KAFKA_PORT if needed
# 3. Configure KAFKA_USERNAME and KAFKA_PASSWORD for production
nano .env

# Start with Kafka using compose override
docker compose -f compose.yaml -f compose.kafka.yaml up
```

---

## Configuration Reference

### Server Configuration

#### SERVER_PORT

**Type:** Integer
**Default:** `8080`
**Required:** No

HTTP server port for API endpoints.

**Example:**
```bash
SERVER_PORT=8080
```

---

### Database Configuration

#### POSTGRES_HOST

**Type:** String
**Default:** `localhost`
**Required:** No

PostgreSQL database host. In Docker Compose, typically set to service name (`postgres`).

**Example:**
```bash
POSTGRES_HOST=localhost          # Local development
POSTGRES_HOST=postgres           # Docker Compose
POSTGRES_HOST=db.example.com     # Remote database
```

---

#### POSTGRES_PORT

**Type:** Integer
**Default:** `5432`
**Required:** No

PostgreSQL database port.

**Example:**
```bash
POSTGRES_PORT=5432
```

---

#### POSTGRES_USER

**Type:** String
**Default:** `postgres`
**Required:** No

PostgreSQL username. Uses standard PostgreSQL default (`postgres`).

**Example:**
```bash
POSTGRES_USER=postgres
POSTGRES_USER=events_user
```

---

#### POSTGRES_PASSWORD

**Type:** String
**Default:** *(none)*
**Required:** **Yes**

PostgreSQL password. This is the **only required configuration variable**.

**Security Note:** Never commit passwords to version control. Use `.env` files (excluded from git) or secret management systems in production.

**Example:**
```bash
POSTGRES_PASSWORD=your-secure-password
```

---

#### POSTGRES_DB

**Type:** String
**Default:** `events`
**Required:** No

PostgreSQL database name. Database must exist or be created by initialization scripts.

**Example:**
```bash
POSTGRES_DB=events
POSTGRES_DB=event_logs
```

---

### Feature Flags

#### KAFKA_ENABLED

**Type:** Boolean
**Default:** `false`
**Required:** No

Enable or disable Kafka consumer. When `false`, only HTTP ingestion endpoints are available.

**Values:**
- `true` - Enable Kafka consumer (requires Kafka infrastructure)
- `false` - Disable Kafka consumer (HTTP-only mode)

**Example:**
```bash
# Minimal deployment (HTTP only)
KAFKA_ENABLED=false

# Full deployment (HTTP + Kafka)
KAFKA_ENABLED=true
```

**Impact:**
- When `true`: Service starts Kafka consumer in background
- When `false`: Service runs as HTTP API only (2-container minimal setup)

---

#### AUTO_MIGRATE

**Type:** Boolean
**Default:** `true`
**Required:** No

Enable or disable automatic database migrations on startup.

**Values:**
- `true` - Run migrations automatically on startup (recommended)
- `false` - Skip migrations (manual migration required)

**Example:**
```bash
# Automatic migrations (default)
AUTO_MIGRATE=true

# Manual migrations
AUTO_MIGRATE=false
```

**Use Cases:**
- `true`: Simplified deployments, development, testing
- `false`: Manual control, zero-downtime deployments, production with separate migration step

---

### Kafka Configuration

**Note:** These settings are **only used when `KAFKA_ENABLED=true`**.

#### KAFKA_HOST

**Type:** String
**Default:** `localhost`
**Required:** No (if Kafka enabled)

Kafka broker host. In Docker Compose, typically set to service name (`kafka`).

**Example:**
```bash
KAFKA_HOST=localhost              # Local development
KAFKA_HOST=kafka                  # Docker Compose
KAFKA_HOST=kafka.example.com      # Remote broker
```

---

#### KAFKA_PORT

**Type:** Integer
**Default:** `9092`
**Required:** No (if Kafka enabled)

Kafka broker port.

**Example:**
```bash
KAFKA_PORT=9092
KAFKA_PORT=29092  # External port in some Docker setups
```

---

#### KAFKA_TOPIC

**Type:** String
**Default:** `events`
**Required:** No (if Kafka enabled)

Kafka topic to consume events from. Topic must exist before starting the consumer.

**Example:**
```bash
KAFKA_TOPIC=events
KAFKA_TOPIC=production-events
```

---

#### KAFKA_GROUP_ID

**Type:** String
**Default:** `events-group`
**Required:** No (if Kafka enabled)

Kafka consumer group ID. Multiple instances with the same group ID share the load (partitions are distributed).

**Example:**
```bash
KAFKA_GROUP_ID=events-group
KAFKA_GROUP_ID=events-consumer-prod
```

**Behavior:**
- Same group ID: Load balancing (partitions distributed)
- Different group IDs: Independent consumers (all receive same messages)

---

#### KAFKA_USERNAME

**Type:** String
**Default:** *(none)*
**Required:** No (if Kafka enabled)

Kafka SASL username for authentication. When set (along with `KAFKA_PASSWORD`), enables SASL/SCRAM-SHA-512 authentication.

**Example:**
```bash
KAFKA_USERNAME=events_consumer
```

**Security Note:** Leave empty for local development without authentication.

---

#### KAFKA_PASSWORD

**Type:** String
**Default:** *(none)*
**Required:** No (if Kafka enabled)

Kafka SASL password for authentication. Used with `KAFKA_USERNAME` to enable SASL/SCRAM-SHA-512.

**Example:**
```bash
KAFKA_PASSWORD=secure-kafka-password
```

**Security Note:** Never commit passwords to version control.

---

#### KAFKA_CONSUMER_OFFSET

**Type:** String (enum)
**Default:** `newest`
**Required:** No (if Kafka enabled)

Initial offset when starting a new consumer group (no previous offset stored).

**Values:**
- `newest` - Start from latest messages (skip old messages)
- `oldest` - Start from beginning of topic (process all messages)

**Example:**
```bash
KAFKA_CONSUMER_OFFSET=newest   # Production (skip old messages)
KAFKA_CONSUMER_OFFSET=oldest   # Development (replay all messages)
```

**Note:** Only applies when consumer group has no committed offset. Existing groups continue from last committed offset.

---

### Event Retention

#### RETENTION_PERIOD_DAYS

**Type:** Integer
**Default:** `90`
**Required:** No

Number of days to keep events before automatic deletion. Events older than this period are deleted by the retention cron job.

**Example:**
```bash
RETENTION_PERIOD_DAYS=90    # 3 months
RETENTION_PERIOD_DAYS=365   # 1 year
RETENTION_PERIOD_DAYS=30    # 1 month
```

**Note:** Can be updated at runtime via `PUT /settings` API endpoint.

---

#### RETENTION_CRON

**Type:** String (cron expression)
**Default:** `0 2 * * *`
**Required:** No

Cron schedule for retention job execution. Uses standard cron format.

**Default:** Daily at 2:00 AM UTC

**Example:**
```bash
RETENTION_CRON="0 2 * * *"      # Daily at 2 AM (default)
RETENTION_CRON="0 3 * * 0"      # Weekly on Sunday at 3 AM
RETENTION_CRON="0 */6 * * *"    # Every 6 hours
```

**Cron Format:**
```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6) (Sunday to Saturday)
│ │ │ │ │
* * * * *
```

---

### Export Limits

#### MAX_EXPORT_SIZE

**Type:** Integer
**Default:** `10000`
**Required:** No

Maximum number of events that can be exported in a single CSV export request.

**Example:**
```bash
MAX_EXPORT_SIZE=10000    # Default
MAX_EXPORT_SIZE=50000    # Larger exports
MAX_EXPORT_SIZE=1000     # Smaller exports
```

**Note:** Can be updated at runtime via `PUT /settings` API endpoint.

**Performance Impact:** Larger exports consume more memory and take longer to generate.

---

## Configuration Examples

### Example 1: Minimal Local Development

**Scenario:** Local development with HTTP ingestion only, no Kafka.

**.env:**
```bash
# Required
POSTGRES_PASSWORD=dev_password

# Optional overrides (if needed)
# SERVER_PORT=8080
# POSTGRES_HOST=localhost
```

**Docker Compose:**
```bash
docker compose up -d postgres events
```

**Result:** 2 containers running (postgres + events)

---

### Example 2: Kafka-Enabled Development

**Scenario:** Local development with Kafka consumer enabled.

**.env:**
```bash
# Required
POSTGRES_PASSWORD=dev_password

# Kafka (optional, using defaults)
# KAFKA_ENABLED=true  # Set by compose.kafka.yaml
# KAFKA_HOST=kafka
# KAFKA_TOPIC=events
```

**Docker Compose:**
```bash
docker compose -f compose.yaml -f compose.kafka.yaml up -d
```

**Result:** 4 containers running (postgres + events + zookeeper + kafka)

---

### Example 3: Production with Remote Kafka (No Authentication)

**Scenario:** Production deployment with remote Kafka cluster (no SASL).

**.env:**
```bash
# Required
POSTGRES_PASSWORD=prod_secure_password

# Feature Flags
KAFKA_ENABLED=true
AUTO_MIGRATE=true

# Database
POSTGRES_HOST=postgres.prod.internal
POSTGRES_DB=events
POSTGRES_USER=events_prod

# Kafka
KAFKA_HOST=kafka.prod.internal
KAFKA_PORT=9092
KAFKA_TOPIC=production-events
KAFKA_GROUP_ID=events-consumer-prod
KAFKA_CONSUMER_OFFSET=newest

# Retention
RETENTION_PERIOD_DAYS=365
RETENTION_CRON="0 3 * * *"

# Export
MAX_EXPORT_SIZE=50000
```

---

### Example 4: Production with Kafka SASL Authentication

**Scenario:** Production deployment with authenticated Kafka cluster.

**.env:**
```bash
# Required
POSTGRES_PASSWORD=prod_secure_password

# Feature Flags
KAFKA_ENABLED=true
AUTO_MIGRATE=true

# Kafka with SASL
KAFKA_HOST=kafka.prod.internal
KAFKA_PORT=9093
KAFKA_TOPIC=production-events
KAFKA_GROUP_ID=events-consumer-prod
KAFKA_USERNAME=events_consumer
KAFKA_PASSWORD=kafka_secure_password
KAFKA_CONSUMER_OFFSET=newest
```

---

### Example 5: HTTP-Only Production (No Kafka)

**Scenario:** Production deployment using only HTTP ingestion (no Kafka infrastructure).

**.env:**
```bash
# Required
POSTGRES_PASSWORD=prod_secure_password

# Feature Flags
KAFKA_ENABLED=false  # Explicitly disable Kafka
AUTO_MIGRATE=true

# Database
POSTGRES_HOST=postgres.prod.internal
POSTGRES_DB=events
POSTGRES_USER=events_prod

# Retention
RETENTION_PERIOD_DAYS=180
RETENTION_CRON="0 2 * * *"

# Export
MAX_EXPORT_SIZE=25000
```

---

## Configuration Loading Order

The service loads configuration in the following priority order (highest to lowest):

1. **Environment variables** (highest priority)
2. **`.env` file** (if present in working directory)
3. **Default values** (lowest priority)

**Example:**

If you have `POSTGRES_PORT=5433` in `.env` but also export `POSTGRES_PORT=5434` in your shell, the service will use `5434` (environment variable takes precedence).

---

## Validation

### Startup Validation

The service validates configuration on startup and exits with an error if:

1. **Required fields are missing:**
   - `POSTGRES_PASSWORD` is empty

2. **Invalid values:**
   - `SERVER_PORT` is not a valid integer
   - `POSTGRES_PORT` is not a valid integer
   - `RETENTION_PERIOD_DAYS` is not a positive integer

### Runtime Validation

Some settings can be updated via API and are validated at runtime:

- `PUT /settings` validates `retention_period_days > 0` and `max_export_size > 0`

---

## Best Practices

### Security

1. **Never commit secrets:**
   - Add `.env` to `.gitignore` (already included)
   - Use `.env.example` for templates (without secrets)

2. **Use strong passwords:**
   - Generate random passwords for `POSTGRES_PASSWORD`
   - Use different passwords for dev/staging/prod

3. **Enable Kafka authentication in production:**
   - Set `KAFKA_USERNAME` and `KAFKA_PASSWORD`
   - Use SASL/SCRAM-SHA-512 for secure communication

### Performance

1. **Database connection pooling:**
   - Service uses max 25 open connections, 5 idle connections
   - No configuration needed (hardcoded optimized values)

2. **Kafka batch processing:**
   - Batches up to 100 events or 500ms timeout
   - No configuration needed (hardcoded optimized values)

3. **Export limits:**
   - Start with default `MAX_EXPORT_SIZE=10000`
   - Increase if needed but monitor memory usage

### Operations

1. **Enable automatic migrations:**
   - Keep `AUTO_MIGRATE=true` for simplified deployments
   - Use `AUTO_MIGRATE=false` only for zero-downtime production deployments

2. **Monitor retention settings:**
   - Adjust `RETENTION_PERIOD_DAYS` based on storage capacity
   - Schedule retention job during low-traffic hours

3. **Use appropriate deployment mode:**
   - Start with `KAFKA_ENABLED=false` for simple use cases
   - Enable Kafka only when needed for high-volume streaming

---

## Troubleshooting

### Service Won't Start

**1. Check required configuration:**
```bash
# Verify POSTGRES_PASSWORD is set
docker compose config | grep POSTGRES_PASSWORD
```

**2. Check database connectivity:**
```bash
# Test database connection
docker compose exec postgres psql -U postgres -d events -c "SELECT 1"
```

**3. Check logs for configuration errors:**
```bash
docker compose logs events | grep -i config
docker compose logs events | grep -i error
```

---

### Kafka Consumer Not Starting

**1. Verify Kafka is enabled:**
```bash
docker compose config | grep KAFKA_ENABLED
# Should show: KAFKA_ENABLED=true
```

**2. Check Kafka connectivity:**
```bash
docker compose exec kafka kafka-topics --list --bootstrap-server localhost:9092
```

**3. Verify topic exists:**
```bash
docker compose exec kafka kafka-topics --describe --topic events --bootstrap-server localhost:9092
```

---

### Migrations Failing

**1. Check database permissions:**
```bash
docker compose exec postgres psql -U postgres -c "\du"
```

**2. Disable auto-migrate and run manually:**
```bash
# Set in .env
AUTO_MIGRATE=false

# Run migrations manually using migrate tool
make migrate-up
```

**3. Check migration status:**
```bash
docker compose exec postgres psql -U postgres -d events -c "SELECT * FROM schema_migrations"
```

---

## See Also

- [Ingestion Guide](INGESTION.md) - Event ingestion API documentation
- [README](../README.md) - Project overview and quick start
- [CLAUDE.md](../CLAUDE.md) - Development guide for AI assistants
- [.env.example](../.env.example) - Minimal configuration template
- [.env.kafka.example](../.env.kafka.example) - Kafka-enabled configuration template
