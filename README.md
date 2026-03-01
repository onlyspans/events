# events

A microservice that manages domain events for dev-platform.

## Stack

- Go 1.25
- PostgreSQL 17 with JSONB support
- [golang-migrate](https://github.com/golang-migrate/migrate) (embedded migrations)
- Prometheus metrics
- gRPC (protobuf, [buf](https://buf.build/))

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

**First-time setup** (installs tools, downloads deps, generates proto code):

```bash
make init
```

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
| `GRPC_PORT` | `9090` | gRPC server port |

## API

**HTTP** — see [`test/http/`](test/http/) for ready-to-use request examples.

**gRPC** (port `9090`) — services: `events.v1.EventService`, `events.v1.SettingsService`. Proto sources: [`api/proto/`](api/proto/).

```bash
# List available services (requires grpcurl)
grpcurl -plaintext localhost:9090 list
```

> The generated Go code lives in `gen/` (gitignored). Run `make proto-gen` after changing `.proto` files.
