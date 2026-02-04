# Full SDD workflow

## Configuration
- **Artifacts Path**: {@artifacts_path} → `.zenflow/tasks/{task_id}`

---

## Workflow Steps

### [x] Step: Requirements

Create a Product Requirements Document (PRD) based on the feature description.

1. Review existing codebase to understand current architecture and patterns
2. Analyze the feature definition and identify unclear aspects
3. Ask the user for clarifications on aspects that significantly impact scope or user experience
4. Make reasonable decisions for minor details based on context and conventions
5. If user can't clarify, make a decision, state the assumption, and continue

Save the PRD to `{@artifacts_path}/requirements.md`.

### [x] Step: Technical Specification

Create a technical specification based on the PRD in `{@artifacts_path}/requirements.md`.

1. Review existing codebase architecture and identify reusable components
2. Define the implementation approach

Save to `{@artifacts_path}/spec.md` with:
- Technical context (language, dependencies)
- Implementation approach referencing existing code patterns
- Source code structure changes
- Data model / API / interface changes
- Delivery phases (incremental, testable milestones)
- Verification approach using project lint/test commands

**Status**: ✅ Complete. Created comprehensive technical specification at `.zenflow/tasks/project-simplification-and-impro-6f51/spec.md` covering:
- Current architecture analysis (2 binaries, 6 containers, Kafka+Zookeeper)
- Unified single-binary design with optional Kafka support
- 4 delivery phases (P0-P2 priorities)
- New HTTP ingestion endpoints (`POST /events/ingest`, `/events/ingest/batch`)
- Embedded migration runner using golang-migrate library
- Configuration simplification (25+ vars → 5-10 vars)
- Docker Compose profiles (default minimal, kafka full)
- Detailed implementation approach for each component
- Testing strategy (unit, integration, e2e)
- Risk mitigation (migration conflicts, performance, backward compatibility)
- Success metrics and rollout plan

### [x] Step: Planning

Created detailed implementation plan with 15 concrete tasks organized into 4 phases (Phase 1-4) matching the technical specification's delivery approach. Each task includes:
- Clear scope and deliverables
- References to relevant spec sections
- Verification steps with specific commands
- Unit tests integrated into implementation tasks

The plan follows an incremental delivery strategy:
- **Phase 1 (P0)**: Foundation - Single binary with optional Kafka and HTTP ingestion (7 tasks)
- **Phase 2 (P1)**: Embedded Migrations - Remove separate migration container (3 tasks)
- **Phase 3 (P1)**: Configuration Simplification - Reduce required env vars (2 tasks)
- **Phase 4 (P2)**: Docker Compose Profiles - Support different deployment scenarios (3 tasks)

---

## Implementation Tasks

### [x] Phase 1 - Task 1: Add Configuration Support for Feature Flags
<!-- chat-id: 42d64a91-6115-4451-afc1-3b202ede8d59 -->

**Goal**: Add `KAFKA_ENABLED` and `AUTO_MIGRATE` flags to configuration system.

**Scope**:
- Update `internal/config/config.go` with new `FeatureFlags` struct
- Add `getEnvAsBool()` helper function if not present
- Set defaults: `KAFKA_ENABLED=false`, `AUTO_MIGRATE=true`

**References**: Spec Section 3.3.2

**Implementation**:
1. Add `FeatureFlags` struct to `Config` in `internal/config/config.go`
2. Implement `getEnvAsBool()` helper for parsing boolean env vars
3. Load flags in `Load()` function with appropriate defaults
4. Write unit tests for config loading with different flag combinations

**Verification**:
```bash
# Run config package tests
go test ./internal/config/... -v -race

# Verify defaults are correct
go run cmd/service/main.go -test-config  # Add flag to print config and exit
```

**Acceptance Criteria**:
- [ ] `FeatureFlags` struct added with `KafkaEnabled` and `AutoMigrate` fields
- [ ] Environment variables `KAFKA_ENABLED` and `AUTO_MIGRATE` load correctly
- [ ] Unit tests verify default values and override behavior
- [ ] Existing config tests still pass

---

### [x] Phase 1 - Task 2: Create Unified Binary Entry Point
<!-- chat-id: 9c0b7719-4e84-4d2d-a532-65f8a447759b -->

**Goal**: Merge service and worker functionality into single binary at `cmd/events/main.go`.

**Scope**:
- Create new `cmd/events/main.go` combining logic from `cmd/service/main.go` and `cmd/worker/main.go`
- Use `errgroup` to manage HTTP server and optional Kafka consumer goroutines
- Implement graceful shutdown with shared context cancellation

**References**: Spec Section 3.3.1

**Implementation**:
1. Create `cmd/events/main.go` with unified startup sequence:
   - Load configuration
   - Connect to PostgreSQL
   - Initialize repositories and services
   - Start retention service (existing cron)
   - Conditionally start Kafka consumer (if `KAFKA_ENABLED=true`)
   - Start HTTP server (blocking)
2. Use `golang.org/x/sync/errgroup` to manage goroutines
3. Implement signal handling (SIGINT, SIGTERM) with 30s timeout
4. Update Makefile with new `build` target for `cmd/events`

**Verification**:
```bash
# Build new binary
make build

# Test with Kafka disabled (HTTP only)
KAFKA_ENABLED=false ./bin/events

# Test with Kafka enabled (both HTTP and Kafka consumer)
KAFKA_ENABLED=true KAFKA_HOST=localhost KAFKA_PORT=9092 ./bin/events

# Verify graceful shutdown
kill -SIGTERM <pid>
```

**Acceptance Criteria**:
- [ ] Single binary successfully starts HTTP server when `KAFKA_ENABLED=false`
- [ ] Binary starts both HTTP server and Kafka consumer when `KAFKA_ENABLED=true`
- [ ] Graceful shutdown cancels context and waits for goroutines (max 30s)
- [ ] No goroutine leaks (test with `-race` flag)
- [ ] Health endpoints (`/healthz`, `/readyz`) work correctly

---

### [x] Phase 1 - Task 3: Implement Event Ingestion DTOs
<!-- chat-id: bbb6ee23-f021-42ff-b688-09608ea2db5b -->

**Goal**: Create data transfer objects for HTTP event ingestion with validation.

**Scope**:
- Add `EventIngestRequest`, `BatchIngestRequest`, `BatchIngestResponse`, `BatchError` to `internal/dto/event.go`
- Implement validation logic for required fields
- Add `ToEvent()` conversion method

**References**: Spec Section 4.3

**Implementation**:
1. Add new DTO structs to `internal/dto/event.go`
2. Implement `EventIngestRequest.Validate()` checking required fields (user_name, category, action)
3. Implement `EventIngestRequest.ToEvent()` converting DTO to `domain.Event`
4. Write unit tests for validation (valid, missing fields, empty strings)
5. Write unit tests for DTO to domain conversion

**Verification**:
```bash
# Run DTO tests
go test ./internal/dto/... -v -race

# Test validation logic
go test ./internal/dto/... -run TestEventIngestRequest_Validate -v
```

**Acceptance Criteria**:
- [x] `EventIngestRequest` validates required fields correctly
- [x] `ToEvent()` conversion handles optional fields and JSONB details
- [x] `BatchIngestRequest` and `BatchIngestResponse` structs defined
- [x] Unit tests cover valid input, missing fields, and edge cases
- [x] JSON marshaling/unmarshaling works correctly

---

### [x] Phase 1 - Task 4: Add Event Service Methods for Ingestion
<!-- chat-id: 23dc37e7-3168-452c-b1c1-290af8e0e55f -->

**Goal**: Extend `EventService` with methods to create events from HTTP requests.

**Scope**:
- Add `CreateEvent()` method for single event ingestion
- Add `CreateEventsBatch()` method for batch ingestion with partial success
- Implement server-side field handling (ID, timestamp defaults)

**References**: Spec Section 3.3.5

**Implementation**:
1. Add `CreateEvent(ctx, dto.EventIngestRequest) (uuid.UUID, error)` to `internal/service/event_service.go`
   - Convert DTO to domain event
   - Set `ID` using `uuid.New()`
   - Default `Timestamp` to `time.Now()` if zero
   - Call `repository.Create()`
2. Add `CreateEventsBatch(ctx, []dto.EventIngestRequest) dto.BatchIngestResponse`
   - Validate each event individually
   - Process all events (don't stop on first error)
   - Return response with success/failure counts and error details
3. Write unit tests with mocked repository (no database)
4. Test error handling (validation errors, repository errors)

**Verification**:
```bash
# Run service tests
go test ./internal/service/... -v -race

# Verify mock-based tests pass without database
go test ./internal/service/... -short -v
```

**Acceptance Criteria**:
- [x] `CreateEvent()` successfully creates events with proper defaults
- [x] `CreateEventsBatch()` processes all events and returns detailed response
- [x] Partial success behavior works (some succeed, some fail)
- [x] Unit tests use mocked repository (no DB dependency)
- [x] Error cases are properly handled and tested

---

### [ ] Phase 1 - Task 5: Implement HTTP Ingestion Handlers
<!-- chat-id: 06753f6a-d2c6-47a3-b673-05f34791fb95 -->

**Goal**: Add HTTP handlers for event ingestion endpoints.

**Scope**:
- Add `IngestEvent()` handler for `POST /events/ingest`
- Add `IngestEventsBatch()` handler for `POST /events/ingest/batch`
- Register routes in HTTP server setup

**References**: Spec Sections 3.3.4, 4.2.1, 4.2.2

**Implementation**:
1. Add handler methods to `internal/handler/event_handler.go`:
   - `IngestEvent()`: Parse JSON, validate, call service, return 201 with ID
   - `IngestEventsBatch()`: Parse JSON, validate batch size (max 100), call service, return 207
2. Implement proper error responses (400 Bad Request, 500 Internal Server Error)
3. Add request timeouts (5s for single, 30s for batch)
4. Register routes in main HTTP setup
5. Write handler integration tests using testcontainers

**Verification**:
```bash
# Run handler integration tests
go test ./internal/handler/... -v -race

# Manual test with curl
curl -X POST http://localhost:8080/events/ingest \
  -H "Content-Type: application/json" \
  -d '{"user_name":"test","category":"test","action":"create"}'

# Test batch endpoint
curl -X POST http://localhost:8080/events/ingest/batch \
  -H "Content-Type: application/json" \
  -d '{"events":[{"user_name":"test","category":"test","action":"create"}]}'
```

**Acceptance Criteria**:
- [ ] Single ingestion endpoint returns 201 with event ID
- [ ] Batch ingestion endpoint returns 207 with success/failure counts
- [ ] Validation errors return 400 Bad Request with clear messages
- [ ] Batch size limit (100 events) is enforced
- [ ] Integration tests verify end-to-end flow with real database

---

### [ ] Phase 1 - Task 6: Add Repository Create Method

**Goal**: Ensure repository supports creating events from HTTP ingestion (may already exist).

**Scope**:
- Verify `repository.Create()` method exists and works for ingestion use case
- Add integration tests if not already covered

**References**: Spec Section 3.3.5

**Implementation**:
1. Review `internal/repository/event_repository.go` for `Create()` method
2. If missing, implement: `Create(ctx, *domain.Event) (uuid.UUID, error)`
3. Ensure method handles JSONB details field correctly
4. Add integration tests using testcontainers (if not present)
5. Test with various event field combinations

**Verification**:
```bash
# Run repository integration tests
go test ./internal/repository/... -v -race

# Verify JSONB details handling
go test ./internal/repository/... -run TestCreate -v
```

**Acceptance Criteria**:
- [ ] `Create()` method successfully inserts events
- [ ] JSONB details field is stored and retrievable
- [ ] Integration tests verify database persistence
- [ ] Method handles NULL optional fields correctly

---

### [ ] Phase 1 - Task 7: Update Build and Deployment Configuration

**Goal**: Update Makefile, Dockerfile, and docker-compose.yaml for unified binary.

**Scope**:
- Update Makefile targets to build single binary
- Update Dockerfile to build single image
- Update docker-compose.yaml to use single service container
- Keep Kafka services but make them optional

**References**: Spec Sections 2.1, 5 (Phase 1)

**Implementation**:
1. Update Makefile:
   - Change `build` target to compile `cmd/events/main.go`
   - Remove separate `build-worker` target (or make it alias)
   - Update `run` target for single binary
2. Update Dockerfile:
   - Build `cmd/events/main.go` as single binary
   - Use multi-stage build (build + runtime)
   - Expose port 8080
3. Update docker-compose.yaml:
   - Replace separate `service` and `worker` with single `events` service
   - Add `KAFKA_ENABLED=false` by default
   - Keep Kafka and Zookeeper services for testing
4. Test full docker-compose deployment

**Verification**:
```bash
# Build binary
make build

# Build Docker image
docker build -t events:latest .

# Test compose deployment (simplified, no Kafka)
docker compose up -d postgres events
curl http://localhost:8080/healthz

# Test with Kafka enabled
KAFKA_ENABLED=true docker compose up -d
```

**Acceptance Criteria**:
- [ ] Makefile builds single binary successfully
- [ ] Dockerfile builds single image
- [ ] docker-compose.yaml works with unified service
- [ ] Default deployment runs without Kafka (2 containers: postgres, events)
- [ ] Can enable Kafka by setting environment variable

---

### [ ] Phase 2 - Task 8: Create Embedded Migration Package

**Goal**: Implement embedded migration runner using golang-migrate library.

**Scope**:
- Create `internal/migrator/migrator.go` package
- Embed migration SQL files using `//go:embed`
- Implement `Run()` function using `golang-migrate/v4`
- Add PostgreSQL advisory lock to prevent concurrent migrations

**References**: Spec Sections 3.3.3, 8.1

**Implementation**:
1. Create `internal/migrator/migrator.go`
2. Embed `migrations/*.sql` files using `//go:embed`
3. Implement `Run(dbURL string) error`:
   - Acquire PostgreSQL advisory lock (lock ID: 12345)
   - Create migrate instance from embedded FS using `iofs.New()`
   - Run `migrate.Up()`
   - Release advisory lock
   - Handle `ErrNoChange` (migrations already applied)
4. Add error handling and logging
5. Write unit tests with mocked database (if possible)
6. Write integration tests with testcontainers

**Verification**:
```bash
# Run migrator tests
go test ./internal/migrator/... -v -race

# Test with fresh database
docker run -d --name test-postgres -e POSTGRES_PASSWORD=test -p 5433:5432 postgres:17-alpine
./bin/events --test-migrate  # Add flag to only run migrations and exit

# Test concurrent migration prevention
./bin/events --test-migrate &
./bin/events --test-migrate &  # Should wait for lock or skip
```

**Acceptance Criteria**:
- [ ] Migration files are embedded in binary
- [ ] Migrations run successfully on startup
- [ ] Advisory lock prevents concurrent migrations
- [ ] `ErrNoChange` is handled gracefully (not logged as error)
- [ ] Integration tests verify migrations with testcontainers
- [ ] Existing migration files work without modification

---

### [ ] Phase 2 - Task 9: Integrate Migrator into Binary Startup

**Goal**: Call embedded migrator from main.go on startup when `AUTO_MIGRATE=true`.

**Scope**:
- Add migrator call to `cmd/events/main.go` startup sequence
- Respect `AUTO_MIGRATE` flag
- Handle migration errors gracefully (don't start service if migrations fail)

**References**: Spec Section 3.3.1

**Implementation**:
1. Import `internal/migrator` in `cmd/events/main.go`
2. Add migration step in startup sequence (after DB connect, before service init):
   ```go
   if config.Features.AutoMigrate {
       if err := migrator.Run(dbURL); err != nil {
           log.Error("migration failed", "error", err)
           os.Exit(1)
       }
       log.Info("migrations completed successfully")
   }
   ```
3. Update logging to distinguish between "no changes" and "applied N migrations"
4. Add integration test for full startup sequence with migrations

**Verification**:
```bash
# Test with auto-migrate enabled (default)
docker compose down -v
docker compose up -d postgres
AUTO_MIGRATE=true ./bin/events

# Verify migrations ran
psql -h localhost -U postgres -c "\dt"  # Should show events, settings tables

# Test with auto-migrate disabled
AUTO_MIGRATE=false ./bin/events  # Should start without running migrations
```

**Acceptance Criteria**:
- [ ] Migrations run automatically on startup when `AUTO_MIGRATE=true`
- [ ] Service exits with error if migrations fail
- [ ] Service starts without migrations when `AUTO_MIGRATE=false`
- [ ] Logs clearly indicate migration status
- [ ] Multiple instances can start concurrently without migration conflicts

---

### [ ] Phase 2 - Task 10: Update Docker Compose to Remove Migration Container

**Goal**: Remove separate `migrate` service from docker-compose.yaml now that migrations are embedded.

**Scope**:
- Remove `migrate` service definition
- Remove dependency on `migrate` from other services
- Update documentation

**References**: Spec Section 5 (Phase 2)

**Implementation**:
1. Remove `migrate` service from `docker-compose.yaml`
2. Remove `depends_on: - migrate` from `events` service
3. Keep `depends_on: - postgres` for database dependency
4. Add comment documenting embedded migrations
5. Update README with new startup behavior

**Verification**:
```bash
# Test fresh deployment
docker compose down -v
docker compose up -d

# Verify migrations ran automatically
docker compose logs events | grep migration

# Verify no migrate container
docker compose ps  # Should only show postgres, events (and kafka/zookeeper if enabled)
```

**Acceptance Criteria**:
- [ ] `migrate` service removed from docker-compose.yaml
- [ ] Fresh deployments work correctly (migrations run automatically)
- [ ] Container count reduced (from 3 to 2 in minimal setup)
- [ ] README updated to reflect automatic migrations

---

### [ ] Phase 3 - Task 11: Simplify Configuration Defaults

**Goal**: Review and optimize configuration defaults to reduce required environment variables.

**Scope**:
- Review all config fields in `internal/config/config.go`
- Set sensible defaults for non-critical settings
- Identify truly required vs optional variables

**References**: Spec Section 5 (Phase 3)

**Implementation**:
1. Review config structs and set better defaults:
   - Server port: `8080`
   - Database host: `localhost`, port: `5432`, dbname: `events`
   - Retention period: `90` days
   - Max export size: `10000`
   - Kafka: disabled by default (`KAFKA_ENABLED=false`)
   - Auto-migrate: enabled by default (`AUTO_MIGRATE=true`)
2. Mark truly required variables (only Postgres password)
3. Update config loading to use defaults consistently
4. Test service starts with minimal configuration

**Verification**:
```bash
# Test with only required variables
export POSTGRES_PASSWORD=test
./bin/events

# Verify defaults are used
curl http://localhost:8080/settings  # Should show default retention and export limits
```

**Acceptance Criteria**:
- [ ] Service runs with only `POSTGRES_PASSWORD` set
- [ ] All other config fields have sensible defaults
- [ ] Config documentation clearly marks required vs optional
- [ ] No breaking changes to existing deployments

---

### [ ] Phase 3 - Task 12: Create Minimal Configuration Example

**Goal**: Create `.env.example` with minimal required configuration.

**Scope**:
- Create `.env.example` file with ≤10 variables
- Document required vs optional variables
- Provide examples for common scenarios

**References**: Spec Sections 10.2, 12.3

**Implementation**:
1. Create `.env.example` with structure:
   ```
   # Required: Database connection
   POSTGRES_PASSWORD=your-secure-password

   # Optional: Database connection (defaults shown)
   # POSTGRES_HOST=localhost
   # POSTGRES_PORT=5432
   # POSTGRES_DB=events
   # POSTGRES_USER=postgres

   # Optional: Enable Kafka consumer
   # KAFKA_ENABLED=false

   # Optional: Disable automatic migrations
   # AUTO_MIGRATE=true

   # Optional: Server port
   # SERVER_PORT=8080
   ```
2. Add comments explaining when to override defaults
3. Create separate example for Kafka-enabled deployment
4. Update README referencing `.env.example`

**Verification**:
```bash
# Copy example and test
cp .env.example .env
# Edit POSTGRES_PASSWORD
docker compose up -d

# Verify service starts successfully
curl http://localhost:8080/healthz
```

**Acceptance Criteria**:
- [ ] `.env.example` contains ≤10 variables
- [ ] Required variables are clearly marked
- [ ] Comments explain when to override defaults
- [ ] README references `.env.example` in quickstart

---

### [ ] Phase 4 - Task 13: Implement Docker Compose Profiles

**Goal**: Add profiles to docker-compose.yaml for different deployment scenarios.

**Scope**:
- Add `default` profile (postgres + events only)
- Add `kafka` profile (adds Kafka and Zookeeper)
- Update service definitions with profile annotations

**References**: Spec Section 5 (Phase 4)

**Implementation**:
1. Update `docker-compose.yaml`:
   - Mark `zookeeper` and `kafka` services with `profiles: ["kafka"]`
   - Keep `postgres` and `events` without profiles (default)
   - Update `events` service to set `KAFKA_ENABLED=true` when kafka profile active
2. Use environment variable override for Kafka config in kafka profile
3. Test both profiles
4. Update documentation with profile examples

**Verification**:
```bash
# Test default profile (minimal)
docker compose up -d
docker compose ps  # Should show only postgres, events

# Test kafka profile (full)
docker compose --profile kafka up -d
docker compose ps  # Should show postgres, events, zookeeper, kafka

# Verify KAFKA_ENABLED is set correctly
docker compose exec events env | grep KAFKA_ENABLED  # default: false
docker compose --profile kafka exec events env | grep KAFKA_ENABLED  # kafka profile: true
```

**Acceptance Criteria**:
- [ ] `docker compose up` starts 2-container minimal deployment
- [ ] `docker compose --profile kafka up` starts 4-container full deployment
- [ ] Kafka consumer only runs when kafka profile is active
- [ ] Profiles are documented in README

---

### [ ] Phase 4 - Task 14: Update Documentation

**Goal**: Update README.md and CLAUDE.md with new architecture and usage.

**Scope**:
- Update architecture diagrams and descriptions
- Simplify quickstart guide
- Document new ingestion endpoints
- Update build and deployment instructions
- Create environment variable reference table

**References**: Spec Section 12

**Implementation**:
1. Update README.md:
   - Replace architecture section (single binary instead of two)
   - Simplify quickstart (2 containers instead of 6)
   - Add API section documenting `/events/ingest` endpoints
   - Add environment variable table (required vs optional)
   - Update build commands for single binary
   - Add profile documentation
2. Update CLAUDE.md:
   - Update project overview with unified binary
   - Update commands section
   - Document embedded migrations
   - Update testing instructions
3. Create `docs/INGESTION.md` with detailed API documentation
4. Create `docs/CONFIGURATION.md` with comprehensive env var reference

**Verification**:
- [ ] Follow README quickstart on fresh machine (or in clean container)
- [ ] Verify all documented commands work
- [ ] Check links are valid
- [ ] Review for clarity and completeness

**Acceptance Criteria**:
- [ ] README reflects new simplified architecture
- [ ] Quickstart guide works end-to-end
- [ ] API documentation includes ingestion endpoints
- [ ] Environment variables are documented with defaults
- [ ] CLAUDE.md updated for AI assistant context

---

### [ ] Phase 4 - Task 15: End-to-End Validation and Performance Testing

**Goal**: Comprehensive validation of all phases with E2E and performance tests.

**Scope**:
- Run full test suite (unit, integration)
- Perform E2E manual testing
- Run performance tests on HTTP ingestion
- Validate backward compatibility

**References**: Spec Sections 6, 10

**Implementation**:
1. Run all tests:
   ```bash
   make test  # All unit and integration tests
   ```
2. E2E test scenarios:
   - Fresh deployment (no data)
   - HTTP single event ingestion
   - HTTP batch event ingestion
   - Event search and export
   - Kafka mode (with `KAFKA_ENABLED=true`)
   - Multi-instance concurrent startup
3. Performance test HTTP ingestion:
   ```bash
   hey -n 10000 -c 100 -m POST \
     -H "Content-Type: application/json" \
     -d '{"user_name":"test","category":"test","action":"test"}' \
     http://localhost:8080/events/ingest
   ```
4. Validate metrics:
   - Check Prometheus `/metrics` endpoint
   - Verify no regression in existing metrics
5. Backward compatibility checks:
   - All existing API endpoints work
   - Kafka consumer behavior unchanged when enabled
   - Database schema unchanged

**Verification**:
```bash
# Run complete test suite
make test

# Check coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Performance baseline
hey -n 10000 -c 100 -m POST \
  -H "Content-Type: application/json" \
  -d '{"user_name":"perf","category":"test","action":"load"}' \
  http://localhost:8080/events/ingest

# Verify resource usage
docker stats events  # Should be ~150MB or less
```

**Acceptance Criteria**:
- [ ] All unit and integration tests pass
- [ ] E2E manual test scenarios work correctly
- [ ] HTTP ingestion achieves ≥1000 events/sec
- [ ] Memory usage ≤150MB for events service (without Kafka)
- [ ] All existing API endpoints return identical responses
- [ ] Kafka consumer works when enabled
- [ ] No breaking changes to existing deployments
- [ ] Documentation matches actual behavior
