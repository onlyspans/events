# Java to Go Migration - Event Logs Microservice

This document details the migration from the Java Spring Boot implementation to idiomatic Go.

## Architecture Overview

The Go implementation maintains the same functionality as the Java version while following Go best practices:

### Project Structure

```
.
├── cmd/
│   ├── service/          # HTTP API service
│   │   └── main.go
│   └── worker/           # Kafka consumer worker
│       └── main.go
├── internal/
│   ├── config/           # Configuration management
│   ├── consumer/         # Kafka consumer implementation
│   ├── domain/           # Domain entities
│   ├── dto/              # Data transfer objects
│   ├── handler/          # HTTP handlers
│   ├── repository/       # Database access layer
│   └── service/          # Business logic
├── migrations/           # Database migrations (golang-migrate format)
├── Dockerfile            # Multi-stage Docker build
├── compose.yaml          # Docker Compose setup
├── Makefile             # Build and development tasks
└── go.mod               # Go dependencies
```

## Key Design Decisions

### 1. Native Go Patterns Over Java Idioms

**Interfaces**
- Java: Multiple interface layers (IEventService, IEventStorage, etc.)
- Go: Interfaces defined at the point of use, repositories and services use concrete types
- Why: Go favors accepting interfaces and returning structs, keeping code simple

**Error Handling**
- Java: Exceptions and try-catch blocks
- Go: Explicit error returns with descriptive wrapping
- Why: Go's error handling makes failure modes explicit and easier to trace

**Dependency Injection**
- Java: Spring Framework annotation-based DI
- Go: Explicit constructor functions
- Why: More transparent, easier to understand, and no reflection overhead

**Logging**
- Java: SLF4J with Logback
- Go: Structured logging with slog (Go 1.21+)
- Why: Native structured logging with zero dependencies, JSON output for production

### 2. Concurrency and Performance

**Kafka Consumer**
- Custom batching implementation using goroutines and channels
- Manual offset management for reliability
- Configurable batch sizes with time-based flushing
- No external consumer library overhead

**Database Connection Pooling**
- Native database/sql pool configuration
- Explicit connection lifecycle management
- Prepared statements for batch inserts

**HTTP Server**
- Native net/http with graceful shutdown
- Request timeout configuration
- Middleware pattern for logging and metrics

### 3. Cost Optimization Features

**Memory Efficiency**
- Reduced memory allocations through proper slice pre-allocation
- Efficient JSON encoding/decoding with streaming
- Connection pooling to minimize overhead

**Binary Size**
- Multi-stage Docker builds
- Distroless base images (~5MB vs ~200MB for Java)
- CGO disabled for static binaries

**Resource Usage**
- No JVM overhead (heap, GC pauses)
- Lower CPU usage for equivalent throughput
- Faster startup time (<100ms vs ~5s for Spring Boot)

## Component Mapping

### Domain Layer

| Java | Go | Notes |
|------|-----|-------|
| EventEntity | domain.Event | JSONB handled with custom Scan/Value |
| SettingsEntity | domain.Settings | Simplified structure |

### DTO Layer

| Java | Go | Notes |
|------|-----|-------|
| EventDto | dto.EventDTO | JSON tags match Java field names |
| SearchEventsRequest | dto.SearchEventsRequest | Identical structure |
| QueryResult | dto.QueryResult | Calculated totalPages inline |

### Repository Layer

| Java | Go | Notes |
|------|-----|-------|
| EventRepository (JPA) | repository.EventRepository | Raw SQL with builder pattern |
| EventSpecification | SearchQuery struct | Type-safe query building |
| SettingsRepository | repository.SettingsRepository | Simple CRUD operations |

### Service Layer

| Java | Go | Notes |
|------|-----|-------|
| EventService | service.EventService | Business logic preserved |
| SettingsService | service.SettingsService | Validation handled explicitly |
| RetentionService | service.RetentionService | Uses robfig/cron for scheduling |

### Consumer

| Java | Go | Notes |
|------|-----|-------|
| KafkaEventConsumer | consumer.KafkaConsumer | Native implementation with IBM/sarama |

### Controllers

| Java | Go | Notes |
|------|-----|-------|
| EventController | handler.EventHandler | Standard http.Handler pattern |
| SettingsController | handler.SettingsHandler | RESTful routing |
| ReadinessController | handler.HealthHandler | Database health checks |

## Configuration

Environment variables (same as Java implementation):

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
KAFKA_USERNAME=
KAFKA_PASSWORD=

# Kafka Consumer Tuning
KAFKA_MAX_POLL_RECORDS=100
KAFKA_FETCH_MIN_BYTES=1
KAFKA_FETCH_MAX_WAIT_MS=500

# Event Logs
RETENTION_PERIOD_DAYS=90
MAX_EXPORT_SIZE=10000
RETENTION_CRON=0 2 * * *
```

## Building and Running

### Local Development

```bash
# Download dependencies
go mod download

# Run service
make run-service

# Run worker
make run-worker

# Run tests
make test
```

### Docker Build

```bash
# Build all targets
docker build -t events-service:latest --target service .
docker build -t events-worker:latest --target worker .

# Or use Docker Compose
docker compose up
```

### Database Migrations

```bash
# Using golang-migrate
migrate -path migrations -database "postgresql://user:pass@localhost:5432/eventlogs?sslmode=disable" up

# Or via Makefile
make migrate-up
```

## API Endpoints

All endpoints match the Java implementation:

- `POST /events` - Search events
- `POST /events/export` - Export events to CSV
- `GET /settings` - Get current settings
- `PUT /settings` - Update settings
- `GET /readyz` - Readiness probe
- `GET /healthz` - Liveness probe
- `GET /metrics` - Prometheus metrics

## Metrics

Prometheus metrics (compatible with Java version):

- `event_logs_received` - Total events received from Kafka
- `event_logs_ingested` - Total events successfully stored
- `event_logs_searched` - Total search operations
- `event_logs_exported` - Total events exported
- `event_logs_batches_processed` - Total Kafka batches processed
- `event_logs_failed` - Total failed events

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run with race detector
go test -race ./...
```

## Performance Comparison

Expected improvements over Java version:

- **Startup Time**: <100ms vs ~5s (50x faster)
- **Memory Usage**: ~50MB vs ~500MB (10x reduction)
- **Binary Size**: ~15MB vs ~200MB (13x smaller)
- **Throughput**: Similar or better due to efficient concurrency
- **Latency**: Lower p99 latency due to no GC pauses

## Migration Checklist

- [x] Domain models and entities
- [x] DTOs with JSON serialization
- [x] Database repositories with query building
- [x] Service layer business logic
- [x] HTTP handlers and routing
- [x] Kafka consumer with batching
- [x] Configuration management
- [x] Health checks and readiness probes
- [x] Prometheus metrics
- [x] Retention scheduler
- [x] CSV export functionality
- [x] Graceful shutdown
- [x] Database migrations
- [x] Docker multi-stage builds
- [x] Logging with structured JSON

## Known Differences

1. **Validation**: Java uses bean validation annotations, Go uses explicit validation in service layer
2. **Mappers**: Java uses MapStruct, Go uses explicit conversion functions
3. **Transactions**: Go uses explicit transaction management instead of @Transactional
4. **Configuration**: Go uses environment variables directly instead of application.properties

## Next Steps

1. Add comprehensive integration tests
2. Set up CI/CD pipeline
3. Add API documentation (OpenAPI/Swagger)
4. Performance benchmarking
5. Load testing to validate improvements
