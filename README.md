# events

A microservice that manages domain events for dev-platform.

## Stack

- Go 1.25
- PostgreSQL 17 with JSONB support
- [golang-migrate](https://github.com/golang-migrate/migrate) (embedded migrations)
- Prometheus metrics

## Quick Start

```bash
# 1. Copy config and set your database DSN
cp configs/.env.example configs/.env
# Edit configs/.env — set POSTGRES_DSN

# 2. Start with Docker Compose
make compose-up

# 3. Service is available at http://localhost:8080
curl http://localhost:8080/healthz
```

## Local Development

**Run without Docker (requires PostgreSQL running locally):**

```bash
# Set env variables or edit configs/.env
make run
```

**Run tests:**

```bash
make test              # all tests
make test-unit         # unit tests only
make test-integration  # integration tests (requires Docker)
```

## Configuration

Config is loaded from `configs/.env`, then `configs/.env.local` (overrides). See `configs/.env.example` for reference.

**Required:**

| Variable | Description |
|---|---|
| `POSTGRES_DSN` | PostgreSQL connection string, e.g. `postgres://postgres:pass@localhost:5432/events?sslmode=disable` |

**Optional:**

| Variable | Default | Description |
|---|---|---|
| `AUTO_MIGRATE` | `true` | Run DB migrations on startup |
| `RETENTION_PERIOD_DAYS` | `90` | Days to keep events |
| `RETENTION_CRON` | `0 2 * * *` | Cron schedule for retention job |
| `MAX_EXPORT_SIZE` | `10000` | Max events per CSV export |

## API

### Ingestion

**`POST /events/ingest`** — ingest a single event

```json
{
  "user_name": "john.doe",
  "category": "document",
  "action": "create",
  "document_name": "plan.pdf",
  "project": "my-project",
  "environment": "production",
  "tenant": "acme",
  "details": { "size_bytes": 1024 }
}
```

Required: `user_name`, `category`, `action`. Response: `201 Created` with `{ "id": "..." }`.

---

**`POST /events/ingest/batch`** — ingest up to 100 events

```json
{ "events": [ { "user_name": "...", "category": "...", "action": "..." } ] }
```

Response: `207 Multi-Status` with `success_count`, `failure_count`, `errors`.

### Search & Export

**`POST /events`** — search events with filters and pagination

```json
{
  "user_name": "john.doe",
  "category": "document",
  "start_date": "2024-01-01T00:00:00Z",
  "end_date": "2024-12-31T23:59:59Z",
  "page": 1,
  "page_size": 50
}
```

All fields optional. Returns `{ "events": [...], "total": 150, "page": 1, "page_size": 50, "total_pages": 3 }`.

**`POST /events/export`** — export matching events to CSV (same filters as search)

### Settings

**`GET /settings`** — get retention and export settings

**`PUT /settings`** — update settings

```json
{ "retention_period_days": 180, "max_export_size": 50000 }
```

### Health & Metrics

| Endpoint | Description |
|---|---|
| `GET /healthz` | Liveness probe |
| `GET /readyz` | Readiness probe (checks DB) |
| `GET /metrics` | Prometheus metrics |
