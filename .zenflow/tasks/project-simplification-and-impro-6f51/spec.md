# Technical Specification: Project Simplification for Docker Compose

## 1. Technical Context

### 1.1 Current Technology Stack
- **Language**: Go 1.25
- **Database**: PostgreSQL 17 (with JSONB, GIN indexes)
- **Message Broker**: Apache Kafka (via IBM Sarama library)
- **Infrastructure**: Zookeeper (Kafka dependency)
- **Migration Tool**: golang-migrate/migrate v4
- **Metrics**: Prometheus client
- **Testing**: testcontainers-go for integration tests

### 1.2 Current Binary Structure
```
events/
├── cmd/
│   ├── service/main.go    # HTTP API server (port 8080)
│   └── worker/main.go     # Kafka consumer
└── internal/
    ├── config/            # Environment-based configuration
    ├── consumer/          # Kafka consumer with SASL/SCRAM auth
    ├── domain/            # Event and Settings entities
    ├── dto/               # Request/response objects
    ├── handler/           # HTTP handlers
    ├── repository/        # Database access layer
    └── service/           # Business logic (includes retention cron)
```

### 1.3 Current Container Deployment
- **postgres**: PostgreSQL 17-alpine
- **zookeeper**: Kafka dependency (confluentinc/cp-zookeeper:7.5.0)
- **kafka**: Message broker (confluentinc/cp-kafka:7.5.0)
- **migrate**: golang-migrate runner (migrate/migrate:v4.17.0)
- **service**: HTTP API server (custom image)
- **worker**: Kafka consumer (custom image)

Total: **6 containers** requiring coordination, health checks, and network configuration.

## 2. Implementation Approach

### 2.1 Architecture Overview

The simplified architecture consolidates functionality into a single deployable binary while maintaining optional Kafka support for production use cases.

**Simplified Binary Structure:**
```
events/ (Single Binary)
├── HTTP Server (always enabled)
│   ├── Existing endpoints: /events, /events/export, /settings, /healthz, /readyz, /metrics
│   └── New endpoints: POST /events/ingest, POST /events/ingest/batch
├── Kafka Consumer (optional, controlled by KAFKA_ENABLED flag)
│   └── Existing functionality unchanged when enabled
├── Retention Service (cron-based, existing)
└── Migration Runner (embedded, runs on startup)
```

### 2.2 Priority Implementation

Based on the PRD's priority matrix, we'll implement in phases:

**P0 (Critical for Simplification):**
1. Add HTTP event ingestion endpoints (`POST /events/ingest` and `/events/ingest/batch`)
2. Make Kafka consumer optional (controlled by `KAFKA_ENABLED` env var)
3. Merge service and worker binaries into single binary

**P1 (Important for UX):**
4. Embed database migrations in service binary (run on startup)
5. Simplify configuration with better defaults

**P2 (Nice-to-have):**
6. Add Docker Compose profiles for different deployment scenarios

## 3. Source Code Structure Changes

### 3.1 New Package Structure

```
events/
├── cmd/
│   └── events/                    # Single binary entry point
│       └── main.go                # Unified main (service + optional worker)
├── internal/
│   ├── config/
│   │   └── config.go              # Add KAFKA_ENABLED, AUTO_MIGRATE flags
│   ├── consumer/
│   │   └── kafka_consumer.go      # No changes (used when KAFKA_ENABLED=true)
│   ├── handler/
│   │   └── event_handler.go       # Add IngestEvent, IngestEventsBatch methods
│   ├── ingestion/                 # NEW: Event ingestion service
│   │   └── ingestion_service.go   # Validation and persistence logic
│   ├── migrator/                  # NEW: Embedded migration runner
│   │   └── migrator.go            # Run migrations on startup using golang-migrate/v4
│   └── ... (existing packages)
```

### 3.2 Removed Components
- `cmd/worker/main.go` - functionality merged into `cmd/events/main.go`
- Separate migration container - replaced by embedded migrator

### 3.3 Key Code Changes

#### 3.3.1 Unified Binary (`cmd/events/main.go`)

**Startup Sequence:**
1. Load configuration (including new flags: `KAFKA_ENABLED`, `AUTO_MIGRATE`)
2. Connect to PostgreSQL
3. **Run database migrations** (if `AUTO_MIGRATE=true`)
4. Initialize repositories and services
5. Start retention service (cron)
6. **Conditionally start Kafka consumer** (if `KAFKA_ENABLED=true`)
7. Start HTTP server (blocking)
8. Handle graceful shutdown (cancel context, wait for goroutines)

**Key Implementation Points:**
- Use `errgroup.Group` from `golang.org/x/sync/errgroup` to manage multiple goroutines
- Kafka consumer runs in background goroutine with shared context
- Single shutdown signal handler cancels context for all components

#### 3.3.2 Configuration Changes (`internal/config/config.go`)

**New Environment Variables:**
```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Kafka    KafkaConfig      // Existing
    EventLog EventLogConfig
    Features FeatureFlags     // NEW
}

type FeatureFlags struct {
    KafkaEnabled  bool  // Default: false
    AutoMigrate   bool  // Default: true
}
```

**Configuration Loading:**
```go
Features: FeatureFlags{
    KafkaEnabled: getEnvAsBool("KAFKA_ENABLED", false),
    AutoMigrate:  getEnvAsBool("AUTO_MIGRATE", true),
}
```

#### 3.3.3 Embedded Migrator (`internal/migrator/migrator.go`)

**Implementation Strategy:**
- Use `github.com/golang-migrate/migrate/v4` as library (not CLI)
- Embed migration SQL files using Go 1.16+ `embed` package
- Use PostgreSQL advisory locks to prevent concurrent migrations

**Key Functions:**
```go
package migrator

import (
    "embed"
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Run(dbURL string) error {
    // Create source from embedded FS
    sourceDriver, err := iofs.New(migrationsFS, "migrations")

    // Create migrate instance
    m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, dbURL)

    // Run migrations with advisory lock
    return m.Up()
}
```

**PostgreSQL Advisory Lock:**
- Use `pg_advisory_lock(12345)` at start of migration
- Release lock with `pg_advisory_unlock(12345)` on completion
- Prevents multiple instances from running migrations concurrently

#### 3.3.4 HTTP Event Ingestion (`internal/handler/event_handler.go`)

**New Handler Methods:**

```go
// IngestEvent handles POST /events/ingest
func (h *EventHandler) IngestEvent(w http.ResponseWriter, r *http.Request) {
    var eventDTO dto.EventIngestRequest
    if err := json.NewDecoder(r.Body).Decode(&eventDTO); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Validate required fields
    if err := eventDTO.Validate(); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Create event
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

    eventID, err := h.eventService.CreateEvent(ctx, eventDTO)
    if err != nil {
        http.Error(w, "Failed to create event", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{
        "id": eventID.String(),
        "created_at": time.Now().Format(time.RFC3339),
    })
}

// IngestEventsBatch handles POST /events/ingest/batch
func (h *EventHandler) IngestEventsBatch(w http.ResponseWriter, r *http.Request) {
    var req dto.BatchIngestRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Validate batch size (max 100 events)
    if len(req.Events) > 100 {
        http.Error(w, "Batch size exceeds maximum (100)", http.StatusBadRequest)
        return
    }

    ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
    defer cancel()

    result := h.eventService.CreateEventsBatch(ctx, req.Events)

    w.WriteHeader(http.StatusMultiStatus)
    json.NewEncoder(w).Encode(result)
}
```

#### 3.3.5 Event Service Extensions (`internal/service/event_service.go`)

**New Methods:**

```go
// CreateEvent creates a single event (for HTTP ingestion)
func (s *EventService) CreateEvent(ctx context.Context, dto dto.EventIngestRequest) (uuid.UUID, error) {
    event := dto.ToEvent()  // Convert DTO to domain.Event

    // Set server-side fields
    event.ID = uuid.New()
    if event.Timestamp.IsZero() {
        event.Timestamp = time.Now()
    }

    return s.eventRepo.Create(ctx, event)
}

// CreateEventsBatch creates multiple events (for batch ingestion)
func (s *EventService) CreateEventsBatch(ctx context.Context, dtos []dto.EventIngestRequest) dto.BatchIngestResponse {
    response := dto.BatchIngestResponse{
        SuccessCount: 0,
        FailureCount: 0,
        Errors:       []dto.BatchError{},
    }

    // Process each event
    for i, eventDTO := range dtos {
        if err := eventDTO.Validate(); err != nil {
            response.FailureCount++
            response.Errors = append(response.Errors, dto.BatchError{
                Index: i,
                Error: err.Error(),
            })
            continue
        }

        _, err := s.CreateEvent(ctx, eventDTO)
        if err != nil {
            response.FailureCount++
            response.Errors = append(response.Errors, dto.BatchError{
                Index: i,
                Error: "failed to create event",
            })
        } else {
            response.SuccessCount++
        }
    }

    return response
}
```

**Note:** Batch processing uses individual inserts. For production high-throughput, could be optimized with `COPY` or bulk insert, but this is sufficient for simplified deployments.

## 4. Data Model / API / Interface Changes

### 4.1 Database Schema

**No changes** to existing schema - maintains backward compatibility.

### 4.2 New API Endpoints

#### 4.2.1 Single Event Ingestion

```
POST /events/ingest
Content-Type: application/json

Request Body:
{
  "timestamp": "2025-01-15T10:30:00Z",       // Optional, defaults to now
  "user_name": "john.doe",                    // Required
  "category": "document",                     // Required
  "action": "create",                         // Required
  "document_name": "proposal.pdf",           // Optional
  "project": "alpha",                         // Optional
  "environment": "production",                // Optional
  "tenant": "acme",                          // Optional
  "details": {                               // Optional JSONB object
    "file_size": 1024000,
    "mime_type": "application/pdf"
  },
  "correlation_id": "abc-123",               // Optional
  "trace_id": "xyz-789"                      // Optional
}

Response: 201 Created
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-01-15T10:30:00Z"
}

Error Responses:
- 400 Bad Request: Invalid JSON or missing required fields
- 500 Internal Server Error: Database error
```

#### 4.2.2 Batch Event Ingestion

```
POST /events/ingest/batch
Content-Type: application/json

Request Body:
{
  "events": [
    { /* event object 1 */ },
    { /* event object 2 */ },
    // ... up to 100 events
  ]
}

Response: 207 Multi-Status
{
  "success_count": 98,
  "failure_count": 2,
  "errors": [
    {
      "index": 5,
      "error": "missing required field: user_name"
    },
    {
      "index": 12,
      "error": "failed to create event"
    }
  ]
}

Error Responses:
- 400 Bad Request: Invalid JSON or batch size > 100
- 500 Internal Server Error: System error (partial success still returns 207)
```

### 4.3 New DTOs (`internal/dto/event.go`)

```go
type EventIngestRequest struct {
    Timestamp     time.Time              `json:"timestamp"`
    UserName      string                 `json:"user_name"`
    Category      string                 `json:"category"`
    Action        string                 `json:"action"`
    DocumentName  string                 `json:"document_name,omitempty"`
    Project       string                 `json:"project,omitempty"`
    Environment   string                 `json:"environment,omitempty"`
    Tenant        string                 `json:"tenant,omitempty"`
    Details       map[string]interface{} `json:"details,omitempty"`
    CorrelationID string                 `json:"correlation_id,omitempty"`
    TraceID       string                 `json:"trace_id,omitempty"`
}

func (e *EventIngestRequest) Validate() error {
    if e.UserName == "" {
        return fmt.Errorf("missing required field: user_name")
    }
    if e.Category == "" {
        return fmt.Errorf("missing required field: category")
    }
    if e.Action == "" {
        return fmt.Errorf("missing required field: action")
    }
    return nil
}

func (e *EventIngestRequest) ToEvent() *domain.Event {
    return &domain.Event{
        Timestamp:     e.Timestamp,
        UserName:      e.UserName,
        Category:      e.Category,
        Action:        e.Action,
        DocumentName:  e.DocumentName,
        Project:       e.Project,
        Environment:   e.Environment,
        Tenant:        e.Tenant,
        Details:       e.Details,
        CorrelationID: e.CorrelationID,
        TraceID:       e.TraceID,
    }
}

type BatchIngestRequest struct {
    Events []EventIngestRequest `json:"events"`
}

type BatchIngestResponse struct {
    SuccessCount int          `json:"success_count"`
    FailureCount int          `json:"failure_count"`
    Errors       []BatchError `json:"errors"`
}

type BatchError struct {
    Index int    `json:"index"`
    Error string `json:"error"`
}
```

### 4.4 Existing API Endpoints

**No changes** to existing endpoints:
- `POST /events` (search)
- `POST /events/export` (CSV export)
- `GET /settings`, `PUT /settings`
- `GET /healthz`, `GET /readyz`, `GET /metrics`

## 5. Delivery Phases

### Phase 1: Foundation (P0 - Critical)
**Goal**: Single binary with optional Kafka support and HTTP ingestion

**Tasks:**
1. Create `cmd/events/main.go` unifying service and worker logic
2. Add `KAFKA_ENABLED` flag to config
3. Refactor worker startup to be conditional (goroutine when enabled)
4. Add HTTP ingestion endpoints (`/events/ingest`, `/events/ingest/batch`)
5. Implement `EventIngestRequest` DTO with validation
6. Add `CreateEvent` and `CreateEventsBatch` to `EventService`
7. Update Makefile to build single binary
8. Update Dockerfile to build single image

**Acceptance Criteria:**
- Binary runs HTTP server and optionally Kafka consumer in same process
- When `KAFKA_ENABLED=false`, only HTTP server runs
- When `KAFKA_ENABLED=true`, both HTTP server and Kafka consumer run
- HTTP ingestion endpoints successfully persist events to database
- All existing tests pass
- Integration tests verify HTTP ingestion

### Phase 2: Embedded Migrations (P1 - Important)
**Goal**: Remove separate migration container

**Tasks:**
1. Create `internal/migrator/migrator.go` package
2. Embed migration SQL files using `//go:embed`
3. Implement migration runner using `golang-migrate/v4` library
4. Add PostgreSQL advisory lock logic to prevent concurrent migrations
5. Add `AUTO_MIGRATE` flag to config (default: true)
6. Call migrator from main.go before starting services
7. Update compose.yaml to remove `migrate` service
8. Document manual migration option (`AUTO_MIGRATE=false`)

**Acceptance Criteria:**
- Service automatically runs migrations on startup (when `AUTO_MIGRATE=true`)
- Multiple instances starting simultaneously don't cause migration conflicts
- Existing migration files work without modification
- Can disable auto-migration for controlled rollouts
- compose.yaml no longer includes `migrate` service

### Phase 3: Configuration Simplification (P1 - Important)
**Goal**: Reduce required configuration from 25+ to ~5-10 variables

**Tasks:**
1. Review all config defaults in `config.go`
2. Set sensible defaults for non-critical settings
3. Create `.env.example` with minimal configuration
4. Document required vs optional environment variables
5. Update README with simplified setup instructions

**Acceptance Criteria:**
- Default `.env` file has ≤10 variables
- Service runs successfully with only database connection configured
- Documentation clearly separates required vs optional config

### Phase 4: Docker Compose Profiles (P2 - Nice-to-have)
**Goal**: Support different deployment scenarios with profiles

**Tasks:**
1. Update compose.yaml with profile annotations
2. Create profiles: `default` (postgres + service), `kafka` (adds Kafka/Zookeeper)
3. Document profile usage in README
4. Test both profiles

**Acceptance Criteria:**
- `docker compose up` starts minimal setup (2 containers: postgres, events)
- `docker compose --profile kafka up` starts full setup (4 containers: postgres, zookeeper, kafka, events)
- Documentation explains when to use each profile

## 6. Verification Approach

### 6.1 Testing Strategy

#### Unit Tests
- Test new `EventIngestRequest.Validate()` method
- Test `EventService.CreateEvent()` and `CreateEventsBatch()` with mocked repository
- Test configuration loading with new flags
- Run existing unit test suite to ensure no regressions

**Command:**
```bash
go test ./internal/service/... -v -race
go test ./internal/dto/... -v -race
```

#### Integration Tests
- Add integration test for HTTP ingestion endpoint
- Verify events persisted via HTTP are searchable
- Test embedded migration runner against testcontainer
- Test Kafka consumer still works when `KAFKA_ENABLED=true`

**Command:**
```bash
go test ./internal/repository/... -v -race
go test ./internal/handler/... -v -race  # New handler integration tests
```

#### End-to-End Tests
- Deploy with compose.yaml (default profile)
- POST event to `/events/ingest`
- Verify event appears in database
- Search for event via `/events`
- Test with Kafka profile enabled
- Verify graceful shutdown

**Manual Test Script:**
```bash
# Start simplified deployment
docker compose up -d

# Wait for service to be ready
curl http://localhost:8080/readyz

# Ingest a test event
curl -X POST http://localhost:8080/events/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "user_name": "test.user",
    "category": "test",
    "action": "create",
    "details": {"test": true}
  }'

# Search for the event
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{"filters": {"user_name": "test.user"}, "page": 0, "size": 10}'

# Check metrics
curl http://localhost:8080/metrics | grep events_
```

### 6.2 Performance Testing

**HTTP Ingestion Throughput:**
- Test with `wrk` or `hey` tool
- Target: 1000 events/sec single instance
- Monitor database connection pool utilization
- Compare with Kafka consumer throughput

**Command:**
```bash
hey -n 10000 -c 100 -m POST \
  -H "Content-Type: application/json" \
  -d '{"user_name":"test","category":"test","action":"test"}' \
  http://localhost:8080/events/ingest
```

### 6.3 Backward Compatibility

**Verify:**
- Existing API endpoints return identical responses
- Database schema unchanged
- Kafka consumer behavior identical when enabled
- Prometheus metrics names unchanged
- Helm charts work with new image (using `KAFKA_ENABLED=true`)

### 6.4 Migration Testing

**Scenarios to Test:**
1. Fresh deployment (no existing data) - migrations run on first start
2. Existing deployment (with data) - new version runs pending migrations
3. Multi-instance deployment - only one instance runs migrations (advisory lock)
4. Migration failure - service doesn't start, clear error message

## 7. Dependencies

### 7.1 New Dependencies

```go
// go.mod additions
require (
    github.com/golang-migrate/migrate/v4 v4.17.0  // Embedded migration runner
    golang.org/x/sync v0.10.0                      // errgroup for goroutine management
)
```

### 7.2 Existing Dependencies (unchanged)
- `github.com/IBM/sarama` - Kafka client
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/prometheus/client_golang` - Metrics
- `github.com/joho/godotenv` - .env loading

### 7.3 Build Dependencies
- Go 1.25+
- Docker (for integration tests via testcontainers)
- golang-migrate CLI (optional, for manual migrations)

## 8. Risks and Mitigations

### 8.1 Risk: Migration Conflicts in Multi-Instance Deployments

**Scenario**: Multiple service instances start simultaneously, attempting to run migrations concurrently.

**Impact**: Database corruption, failed migrations, service startup failures.

**Mitigation**:
- Use PostgreSQL advisory locks (`pg_advisory_lock`) in migration runner
- Only one instance acquires lock and runs migrations
- Other instances wait or skip if migrations already complete
- Add configurable retry logic with exponential backoff

**Implementation**:
```sql
-- In migration runner
SELECT pg_advisory_lock(12345);  -- Acquire lock
-- Run migrations
SELECT pg_advisory_unlock(12345);  -- Release lock
```

### 8.2 Risk: HTTP Ingestion Performance

**Scenario**: HTTP ingestion cannot handle production event volumes (10K+ events/sec).

**Impact**: Event loss, service degradation, increased latency.

**Mitigation**:
- Document performance limits for HTTP ingestion mode
- Recommend Kafka mode (`KAFKA_ENABLED=true`) for high-throughput scenarios
- Add rate limiting to prevent overload
- Implement connection pooling optimization
- Consider batch ingestion endpoint for bulk uploads

**Guidance**: In simplified deployment documentation, clearly state:
> "HTTP ingestion is suitable for up to 1000 events/sec. For higher volumes, enable Kafka mode with `KAFKA_ENABLED=true`."

### 8.3 Risk: Backward Compatibility with Helm Charts

**Scenario**: Existing Kubernetes deployments break when switching to unified binary.

**Impact**: Production outages, rollback required.

**Mitigation**:
- Helm chart uses `KAFKA_ENABLED=true` by default (preserves current behavior)
- Keep separate service and worker deployments in Helm initially
- Provide migration guide for consolidating to single deployment
- Version Helm chart appropriately (major version bump)

### 8.4 Risk: Migration Rollback Complexity

**Scenario**: Need to rollback migrations after failed deployment.

**Impact**: Cannot easily revert to previous schema version.

**Mitigation**:
- Add `AUTO_MIGRATE=false` flag for production deployments
- Provide separate migration tool/command for controlled rollouts
- Document rollback procedures in operations guide
- Maintain separate `migrate` container option in compose.yaml (commented out)

## 9. Open Questions

### 9.1 Resolved Design Decisions

**Q: Should we maintain separate binaries for production Kubernetes deployments?**
**A**: No. Single binary with `KAFKA_ENABLED` flag provides same functionality. Kubernetes can still run separate pods with different configurations if desired, but binary is unified.

**Q: What happens if HTTP ingestion and Kafka both write same event (deduplication)?**
**A**: No deduplication. Database uses `gen_random_uuid()` for IDs, so duplicates will have different IDs. This is acceptable - users should choose one ingestion method per environment.

**Q: Should batch ingestion be transactional (all-or-nothing)?**
**A**: No. Use partial success model (207 Multi-Status). Allows maximum event ingestion even if some events fail validation. Client can retry failed events individually.

### 9.2 Outstanding Questions

**Q: How to handle migration rollbacks in production?**
**Options:**
1. Provide separate `events migrate down` command
2. Require manual rollback using golang-migrate CLI
3. Don't support rollbacks (migrations are forward-only)

**Recommendation**: Option 1 - Add `--migrate-down N` flag to binary for rollback support.

**Q: Should we expose migration status endpoint (e.g., `GET /migrations`)?**
**Recommendation**: Yes, add in Phase 2. Endpoint returns current schema version and pending migrations.

**Q: What's the migration strategy for existing production deployments?**
**Recommendation**: Blue-green deployment:
1. Deploy new version alongside old version
2. Route traffic to new version
3. Monitor for issues
4. Decommission old version
5. Update Helm charts to use `KAFKA_ENABLED=true`

## 10. Success Metrics

### 10.1 Deployment Complexity Reduction
- **Baseline**: 6 containers (postgres, zookeeper, kafka, migrate, service, worker)
- **Target**: 2 containers (postgres, events) for simple deployment
- **Measurement**: Count of services in compose.yaml default profile

### 10.2 Configuration Simplicity
- **Baseline**: 25+ environment variables required
- **Target**: ≤10 required environment variables
- **Measurement**: Count of required variables in `.env.example`

### 10.3 Time to First Deployment
- **Baseline**: ~30 minutes (setup Kafka, configure 25 env vars, deploy 6 containers)
- **Target**: <5 minutes (configure 5-10 env vars, deploy 2 containers)
- **Measurement**: Time from `git clone` to successful event ingestion

### 10.4 Resource Usage
- **Baseline**: ~600MB memory (all containers combined)
- **Target**: ~150MB memory (postgres + events only)
- **Measurement**: `docker stats` output for running containers

### 10.5 API Compatibility
- **Target**: 100% backward compatibility with existing endpoints
- **Measurement**: Existing API integration tests pass without modification

## 11. Future Enhancements (Out of Scope)

These items are explicitly **not** included in this specification but could be considered in future iterations:

1. **Alternative Databases**: Support for SQLite or embedded databases for even simpler deployments
2. **Built-in UI**: Web interface for event browsing and searching
3. **Event Streaming**: WebSocket endpoint for real-time event streaming
4. **Authentication**: API key or OAuth2 authentication for ingestion endpoints
5. **Schema Validation**: JSON Schema validation for event `details` field
6. **Event Replay**: Ability to replay events from database back to Kafka
7. **High-Performance Batch Insert**: Optimize batch ingestion with PostgreSQL `COPY` protocol
8. **Multi-tenancy**: Tenant isolation at database level
9. **Event Archival**: Archive old events to S3/object storage instead of deletion
10. **Distributed Tracing**: OpenTelemetry integration for observability

## 12. Documentation Updates Required

### 12.1 README.md
- Update architecture diagram (single binary)
- Simplify quickstart guide (2 containers instead of 6)
- Document new ingestion endpoints
- Add environment variable reference table
- Update build and deployment instructions

### 12.2 CLAUDE.md
- Update project overview with new architecture
- Document unified binary structure
- Update development commands (single binary)
- Add migration embedding details
- Update testing strategy

### 12.3 New Documentation Files
- `docs/MIGRATION.md` - Guide for migrating from multi-binary to single-binary deployment
- `docs/INGESTION.md` - API documentation for HTTP ingestion endpoints
- `docs/CONFIGURATION.md` - Comprehensive environment variable reference
- `.env.example` - Minimal example configuration

### 12.4 API Documentation
- Update OpenAPI/Swagger spec with new endpoints
- Add request/response examples for ingestion endpoints
- Document batch ingestion partial success behavior

## 13. Rollout Plan

### 13.1 Development Environment
1. Merge Phase 1 changes to feature branch
2. Test with local compose.yaml
3. Run full test suite (unit + integration)
4. Manual verification of HTTP ingestion

### 13.2 Staging/QA Environment
1. Deploy to staging with `KAFKA_ENABLED=false` (test simplified mode)
2. Deploy to staging with `KAFKA_ENABLED=true` (test production mode)
3. Run performance tests
4. Validate metrics and logging

### 13.3 Production Kubernetes (Helm)
1. Update Helm chart with new image
2. Set `KAFKA_ENABLED=true` (preserves current behavior)
3. Deploy to one environment (e.g., dev)
4. Monitor for 24 hours
5. Gradual rollout to other environments

### 13.4 Docker Compose Users
1. Release new version with updated compose.yaml
2. Publish migration guide (old → new compose.yaml)
3. Provide side-by-side comparison
4. Announce in release notes and documentation

## 14. Definition of Done

**Phase 1 (P0) is considered complete when:**
- ✅ Single binary runs both HTTP server and optional Kafka consumer
- ✅ `KAFKA_ENABLED` flag controls Kafka consumer startup
- ✅ HTTP ingestion endpoints (`/events/ingest`, `/events/ingest/batch`) work correctly
- ✅ All existing tests pass without modification
- ✅ New integration tests cover HTTP ingestion
- ✅ Dockerfile builds single image
- ✅ compose.yaml works with new image (both profiles)
- ✅ Documentation updated (README, CLAUDE.md)

**Phase 2 (P1) is considered complete when:**
- ✅ Migrations run automatically on service startup (when `AUTO_MIGRATE=true`)
- ✅ PostgreSQL advisory locks prevent concurrent migration issues
- ✅ compose.yaml no longer includes separate `migrate` service
- ✅ Integration tests verify embedded migrations
- ✅ Manual migration option documented (`AUTO_MIGRATE=false`)

**Phase 3 (P1) is considered complete when:**
- ✅ `.env.example` contains ≤10 required variables
- ✅ Service runs successfully with minimal configuration
- ✅ Configuration documentation complete (required vs optional)

**Phase 4 (P2) is considered complete when:**
- ✅ Docker Compose profiles implemented (`default`, `kafka`)
- ✅ `docker compose up` starts 2-container deployment
- ✅ `docker compose --profile kafka up` starts full deployment
- ✅ Profile usage documented in README
