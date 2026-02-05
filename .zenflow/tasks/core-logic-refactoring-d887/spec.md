# Technical Specification: Core Logic Refactoring

## Task Assessment

**Difficulty Level: Hard**

This is a comprehensive refactoring task that touches the entire codebase architecture. It requires:
- Architectural changes (interface-based design, dependency injection)
- Cross-cutting concerns (middleware, error handling, validation)
- Configuration and lifecycle management improvements
- Code deduplication and pattern standardization
- Maintaining backward compatibility with existing API contracts

---

## Technical Context

| Aspect | Details |
|--------|---------|
| Language | Go 1.25 |
| Database | PostgreSQL with JSONB support |
| Messaging | Kafka (IBM Sarama) |
| Migrations | golang-migrate (embedded) |
| Metrics | Prometheus |
| Testing | testcontainers-go for integration tests |

### Current Codebase Structure
```
cmd/events/main.go          - Entry point (~260 lines)
internal/
  config/config.go          - Configuration loading (~190 lines)
  consumer/kafka_consumer.go - Kafka consumer (~260 lines)
  domain/event.go           - Core entities
  dto/event.go              - Data transfer objects (~190 lines)
  handler/                  - HTTP handlers (~260 lines total)
  repository/               - Database access (~340 lines)
  service/                  - Business logic (~430 lines)
  migrator/                 - Database migrations
```

---

## Key Issues Identified

### 1. Interface Segregation & Dependency Inversion Violations

**Problem**: Handlers and services depend on concrete implementations rather than interfaces, making testing difficult and creating tight coupling.

| File | Issue |
|------|-------|
| `handler/event_handler.go:17` | Depends on `*service.EventService` (concrete) |
| `handler/settings_handler.go:14` | Depends on `*service.SettingsService` (concrete) |
| `handler/health_handler.go:12` | Depends on `*sql.DB` directly |
| `service/settings_service.go:15` | Depends on `*repository.SettingsRepository` (concrete) |

### 2. Code Duplication

**Problem**: Repetitive patterns across handlers and services.

| Location | Duplication |
|----------|-------------|
| All handlers | HTTP method validation, JSON decoding, error responses |
| `cmd/events/main.go:114-123` | Manual method-based routing with switch statements |
| `service/event_service.go:343-431` | `CreateEvent` and `CreateEventsBatch` share ~80% logic |
| `dto/event.go:42-76` | `SearchEventsRequest` and `ExportEventsRequest` are nearly identical |

### 3. Configuration & Validation Gaps

**Problem**: No validation of critical configuration at startup.

| File | Issue |
|------|-------|
| `config/config.go:102` | Empty DSN is accepted without validation |
| `config/config.go:87-128` | `Load()` never returns an error for invalid config |
| Multiple files | Magic numbers scattered (batch sizes, timeouts, pool sizes) |

### 4. Error Handling Issues

**Problem**: Inconsistent error handling and lost error context.

| Location | Issue |
|----------|-------|
| `handler/event_handler.go:65-67` | JSON encoding errors are logged but client receives success |
| `consumer/kafka_consumer.go:177` | Uses `context.Background()` ignoring shutdown signals |
| Service layer | Errors lose contextual information in batch operations |

### 5. HTTP Handler Concerns

**Problem**: Handlers mix HTTP concerns with business logic validation.

| Issue | Impact |
|-------|--------|
| No middleware stack | Logging, recovery, CORS, rate limiting manually implemented |
| Handlers validate page sizes | Should be done in a validation layer |
| Response encoding after headers | Cannot properly handle encoding failures |

### 6. Inconsistent Naming

**Problem**: Field naming differs between DTO and domain layers.

| DTO Field | Domain Field | Location |
|-----------|--------------|----------|
| `UserName` | `User` | `dto/event.go:90` vs `domain/event.go` |
| `user_name` (JSON) | `user` (JSON elsewhere) | Inconsistent API contract |

---

## Implementation Approach

### Phase 1: Foundation - Interfaces & Configuration

1. **Define service interfaces** in a dedicated `internal/ports` package
2. **Add configuration validation** with descriptive error messages
3. **Centralize magic numbers** in configuration structs

### Phase 2: Handler Layer Refactoring

1. **Create HTTP middleware stack** for cross-cutting concerns
2. **Extract common handler patterns** into reusable utilities
3. **Standardize error responses** with structured error types

### Phase 3: Service Layer Improvements

1. **Eliminate code duplication** in event processing
2. **Improve error context** with structured error wrapping
3. **Standardize DTO/domain conversions** with clear mappers

### Phase 4: Consumer & Lifecycle

1. **Fix context propagation** in Kafka consumer
2. **Improve graceful shutdown** handling
3. **Add health check interfaces**

---

## Source Code Changes

### New Files to Create

| Path | Purpose |
|------|---------|
| `internal/ports/repositories.go` | Repository interfaces |
| `internal/ports/services.go` | Service interfaces |
| `internal/http/middleware/` | HTTP middleware (logging, recovery, validation) |
| `internal/http/response/` | Standardized response helpers |
| `internal/apperr/errors.go` | Application error types |

### Files to Modify

| Path | Changes |
|------|---------|
| `internal/config/config.go` | Add `Validate()` method, centralize constants |
| `internal/handler/*.go` | Use interfaces, remove boilerplate, use middleware |
| `internal/service/*.go` | Implement interfaces, improve error context |
| `internal/consumer/kafka_consumer.go` | Fix context usage, add interface |
| `cmd/events/main.go` | Use router, wire dependencies via interfaces |
| `internal/dto/event.go` | Fix naming consistency, merge duplicate request types |

### Files to Delete

None - all changes are refactoring existing code.

---

## Interface Definitions

### Repository Interfaces (`internal/ports/repositories.go`)

```go
package ports

type EventRepository interface {
    SaveBatch(ctx context.Context, events []*domain.Event) error
    Search(ctx context.Context, query SearchQuery) ([]*domain.Event, int64, error)
    DeleteOlderThan(ctx context.Context, cutoffDate time.Time) (int64, error)
}

type SettingsRepository interface {
    Get(ctx context.Context) (*domain.Settings, error)
    Save(ctx context.Context, settings *domain.Settings) error
}

type HealthChecker interface {
    Ping(ctx context.Context) error
}
```

### Service Interfaces (`internal/ports/services.go`)

```go
package ports

type EventService interface {
    SearchEvents(ctx context.Context, req dto.SearchEventsRequest) (*dto.QueryResult, error)
    ExportCSV(ctx context.Context, req dto.ExportEventsRequest, w io.Writer) error
    CreateEvent(ctx context.Context, req dto.EventIngestRequest) (uuid.UUID, error)
    CreateEventsBatch(ctx context.Context, reqs []dto.EventIngestRequest) dto.BatchIngestResponse
    IngestEvents(ctx context.Context, events []*dto.EventDTO) error
}

type SettingsService interface {
    GetSettings(ctx context.Context) (*dto.SettingsDTO, error)
    UpdateSettings(ctx context.Context, settings *dto.SettingsDTO) (*dto.SettingsDTO, error)
}
```

---

## Configuration Changes

### Add Validation Method

```go
func (c *Config) Validate() error {
    if c.Database.DSN == "" {
        return errors.New("POSTGRES_DSN is required")
    }
    if c.EventLog.RetentionPeriodDays < 1 || c.EventLog.RetentionPeriodDays > 3650 {
        return errors.New("RETENTION_PERIOD_DAYS must be between 1 and 3650")
    }
    if c.EventLog.MaxExportSize < 1 {
        return errors.New("MAX_EXPORT_SIZE must be positive")
    }
    return nil
}
```

### Centralized Constants

```go
type ServerConfig struct {
    Port         string
    ReadTimeout  time.Duration
    WriteTimeout time.Duration
    IdleTimeout  time.Duration
}

type DatabaseConfig struct {
    DSN             string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}

type HandlerConfig struct {
    MaxPageSize  int
    MaxBatchSize int
}
```

---

## API Changes

### Request DTO Consolidation

Merge `SearchEventsRequest` and `ExportEventsRequest` into a single base type:

```go
type EventFilterRequest struct {
    User          string     `json:"user,omitempty"`
    Category      string     `json:"category,omitempty"`
    Action        string     `json:"action,omitempty"`
    Document      string     `json:"document,omitempty"`
    Project       string     `json:"project,omitempty"`
    Environment   string     `json:"environment,omitempty"`
    Tenant        string     `json:"tenant,omitempty"`
    CorrelationID string     `json:"correlationId,omitempty"`
    TraceID       string     `json:"traceId,omitempty"`
    StartDate     *time.Time `json:"startDate,omitempty"`
    EndDate       *time.Time `json:"endDate,omitempty"`
    SortBy        string     `json:"sortBy,omitempty"`
    SortOrder     string     `json:"sortOrder,omitempty"`
}

type SearchEventsRequest struct {
    EventFilterRequest
    Page int `json:"page,omitempty"`
    Size int `json:"size,omitempty"`
}

type ExportEventsRequest = EventFilterRequest
```

### Field Naming Consistency

Standardize `EventIngestRequest` to use `user` instead of `user_name`:

```go
type EventIngestRequest struct {
    Timestamp     time.Time              `json:"timestamp"`
    User          string                 `json:"user"`           // Changed from user_name
    Category      string                 `json:"category"`
    Action        string                 `json:"action"`
    DocumentName  string                 `json:"documentName,omitempty"`  // camelCase
    // ...
}
```

**Note**: This is a breaking API change. Consider adding backward compatibility with custom JSON unmarshaling or deprecation period.

---

## Verification Approach

### Unit Tests
```bash
# Run unit tests with race detection
go test -race ./internal/service/... ./internal/dto/... ./internal/config/...
```

### Integration Tests
```bash
# Run integration tests (requires Docker)
go test -race ./internal/repository/... ./internal/handler/... ./cmd/events/...
```

### Full Test Suite
```bash
make test
```

### Lint Checks
```bash
# If golangci-lint is available
golangci-lint run ./...
```

### Manual Verification
1. Start service: `make run`
2. Test health endpoints: `curl localhost:8080/healthz`
3. Test event ingestion: `curl -X POST localhost:8080/events/ingest -d '...'`
4. Test search: `curl -X POST localhost:8080/events -d '...'`

---

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Breaking API changes (field naming) | Add backward-compatible JSON unmarshaling |
| Test failures during refactoring | Refactor incrementally, run tests after each change |
| Behavioral changes in error handling | Document expected error responses |
| Kafka consumer context changes | Test with actual Kafka in integration tests |

---

## Out of Scope

The following improvements are identified but excluded from this refactoring:

1. **Router library adoption** (chi, gorilla/mux) - Significant change, could be separate task
2. **Structured logging context propagation** - Good improvement but not critical
3. **Rate limiting and circuit breakers** - New features, not refactoring
4. **Database transaction isolation changes** - Requires careful analysis
5. **Dead letter queue for Kafka** - New feature

---

## Code Style Guidelines

**Comments**: Keep comments minimal. Only add comments to explain non-trivial logic, edge cases, or workarounds. Avoid obvious comments that just restate what the code does. Well-named functions and variables should be self-documenting.

---

## Success Criteria

1. All existing tests pass
2. Handlers depend on interfaces, not concrete implementations
3. Configuration validation fails fast on startup for invalid settings
4. No duplicate code patterns in handlers
5. Service methods have consistent error wrapping
6. Magic numbers are centralized in configuration
7. Code is simpler, more readable, and follows Go idioms
