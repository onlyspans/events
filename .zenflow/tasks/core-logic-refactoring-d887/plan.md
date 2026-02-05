# Spec and build

## Configuration
- **Artifacts Path**: {@artifacts_path} → `.zenflow/tasks/{task_id}`

---

## Agent Instructions

Ask the user questions when anything is unclear or needs their input. This includes:
- Ambiguous or incomplete requirements
- Technical decisions that affect architecture or user experience
- Trade-offs that require business context

Do not make assumptions on important decisions — get clarification first.

---

## Workflow Steps

### [x] Step: Technical Specification

Assess the task's difficulty, as underestimating it leads to poor outcomes.
- easy: Straightforward implementation, trivial bug fix or feature
- medium: Moderate complexity, some edge cases or caveats to consider
- hard: Complex logic, many caveats, architectural considerations, or high-risk changes

Create a technical specification for the task that is appropriate for the complexity level:
- Review the existing codebase architecture and identify reusable components.
- Define the implementation approach based on established patterns in the project.
- Identify all source code files that will be created or modified.
- Define any necessary data model, API, or interface changes.
- Describe verification steps using the project's test and lint commands.

**Output**: See `spec.md` for the complete technical specification.

**Assessment**: Hard - Comprehensive architectural refactoring touching all layers.

---

### [ ] Step: Define Interfaces and Ports

Create the interface abstractions that will decouple the layers.

**Tasks:**
- [ ] Create `internal/ports/repositories.go` with `EventRepository`, `SettingsRepository`, and `HealthChecker` interfaces
- [ ] Create `internal/ports/services.go` with `EventService` and `SettingsService` interfaces
- [ ] Add interface documentation explaining the contract
- [ ] Ensure existing implementations satisfy the interfaces (compile-time checks)
- [ ] Run tests to verify no regressions

**Files:**
- Create: `internal/ports/repositories.go`
- Create: `internal/ports/services.go`
- Modify: `internal/repository/event_repository.go` (add interface check)
- Modify: `internal/repository/settings_repository.go` (add interface check)
- Modify: `internal/service/event_service.go` (add interface check)
- Modify: `internal/service/settings_service.go` (add interface check)

**Verification:** `go build ./... && go test -race ./internal/...`

---

### [ ] Step: Add Configuration Validation

Add validation to catch configuration errors at startup.

**Tasks:**
- [ ] Add `Validate()` method to `Config` struct
- [ ] Validate required fields (POSTGRES_DSN)
- [ ] Validate numeric ranges (retention days, export size, pool sizes)
- [ ] Call `Validate()` in `main.go` after `Load()`
- [ ] Centralize magic numbers into config structs (timeouts, batch sizes, pool sizes)
- [ ] Update tests for configuration validation
- [ ] Run tests

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `cmd/events/main.go`

**Verification:** `go test -race ./internal/config/...`

---

### [ ] Step: Create Application Error Types

Define structured error types for consistent error handling.

**Tasks:**
- [ ] Create `internal/apperr/errors.go` with `AppError` type
- [ ] Define error codes/types (validation, not found, internal)
- [ ] Add helper constructors (`NewValidationError`, `NewInternalError`)
- [ ] Add error wrapping support
- [ ] Write unit tests for error types

**Files:**
- Create: `internal/apperr/errors.go`
- Create: `internal/apperr/errors_test.go`

**Verification:** `go test -race ./internal/apperr/...`

---

### [ ] Step: Refactor Handlers to Use Interfaces

Update handlers to depend on interfaces instead of concrete types.

**Tasks:**
- [ ] Update `EventHandler` to accept `ports.EventService` interface
- [ ] Update `SettingsHandler` to accept `ports.SettingsService` interface
- [ ] Update `HealthHandler` to accept `ports.HealthChecker` interface
- [ ] Create a `DBHealthChecker` adapter wrapping `*sql.DB`
- [ ] Update `cmd/events/main.go` to wire interfaces
- [ ] Run existing tests to verify no regressions

**Files:**
- Modify: `internal/handler/event_handler.go`
- Modify: `internal/handler/settings_handler.go`
- Modify: `internal/handler/health_handler.go`
- Create: `internal/handler/health_adapter.go`
- Modify: `cmd/events/main.go`

**Verification:** `go test -race ./internal/handler/... ./cmd/events/...`

---

### [ ] Step: Create HTTP Middleware and Response Helpers

Extract common HTTP patterns into reusable middleware and helpers.

**Tasks:**
- [ ] Create `internal/http/middleware/logging.go` - request logging middleware
- [ ] Create `internal/http/middleware/recovery.go` - panic recovery middleware
- [ ] Create `internal/http/middleware/chain.go` - middleware chaining utility
- [ ] Create `internal/http/response/response.go` - JSON response helpers
- [ ] Create `internal/http/response/errors.go` - error response formatting
- [ ] Move logging middleware from `main.go` to new package
- [ ] Write unit tests for middleware and response helpers
- [ ] Update `main.go` to use the new middleware chain

**Files:**
- Create: `internal/http/middleware/logging.go`
- Create: `internal/http/middleware/recovery.go`
- Create: `internal/http/middleware/chain.go`
- Create: `internal/http/middleware/middleware_test.go`
- Create: `internal/http/response/response.go`
- Create: `internal/http/response/errors.go`
- Create: `internal/http/response/response_test.go`
- Modify: `cmd/events/main.go`

**Verification:** `go test -race ./internal/http/...`

---

### [ ] Step: Simplify Handler Methods

Remove boilerplate from handlers using the new helpers.

**Tasks:**
- [ ] Remove duplicate method validation (handlers already route by method in main.go)
- [ ] Use response helpers for JSON encoding
- [ ] Use structured error responses from `apperr` package
- [ ] Simplify context timeout patterns
- [ ] Update handler tests if needed

**Files:**
- Modify: `internal/handler/event_handler.go`
- Modify: `internal/handler/settings_handler.go`
- Modify: `internal/handler/health_handler.go`

**Verification:** `go test -race ./internal/handler/...`

---

### [ ] Step: Consolidate DTO Types and Fix Naming

Reduce duplication in DTOs and standardize field naming.

**Tasks:**
- [ ] Create `EventFilterRequest` base type with common filter fields
- [ ] Have `SearchEventsRequest` embed `EventFilterRequest` + pagination
- [ ] Make `ExportEventsRequest` an alias for `EventFilterRequest`
- [ ] Rename `UserName` to `User` in `EventIngestRequest` for consistency
- [ ] Add backward-compatible JSON unmarshaling for `user_name` field
- [ ] Update all usages throughout the codebase
- [ ] Update and run tests

**Files:**
- Modify: `internal/dto/event.go`
- Modify: `internal/dto/event_test.go`
- Modify: `internal/service/event_service.go`

**Verification:** `go test -race ./internal/dto/... ./internal/service/...`

---

### [ ] Step: Eliminate Service Code Duplication

Refactor event service to remove duplicate code patterns.

**Tasks:**
- [ ] Extract common event processing logic from `CreateEvent` and `CreateEventsBatch`
- [ ] Create internal helper method `processIngestRequest()`
- [ ] Simplify the batch method to iterate over the helper
- [ ] Improve error context in batch processing (include event index)
- [ ] Run tests to verify behavior is preserved

**Files:**
- Modify: `internal/service/event_service.go`
- Modify: `internal/service/event_service_test.go`

**Verification:** `go test -race ./internal/service/...`

---

### [ ] Step: Refactor Settings Service to Use Interface

Update settings service to depend on repository interface.

**Tasks:**
- [ ] Change `SettingsService.repo` from `*repository.SettingsRepository` to `ports.SettingsRepository`
- [ ] Update constructor signature
- [ ] Update `main.go` wiring
- [ ] Verify tests pass

**Files:**
- Modify: `internal/service/settings_service.go`
- Modify: `cmd/events/main.go`

**Verification:** `go test -race ./internal/service/...`

---

### [ ] Step: Fix Kafka Consumer Context Handling

Improve context propagation and shutdown handling in the Kafka consumer.

**Tasks:**
- [ ] Replace `context.Background()` with proper context in `ConsumeClaim`
- [ ] Use session context instead of background context for event ingestion
- [ ] Fix the `ready` channel race condition (don't recreate in loop)
- [ ] Improve error handling for batch processing failures
- [ ] Add interface for event ingestion dependency
- [ ] Update consumer to accept interface instead of concrete service

**Files:**
- Modify: `internal/consumer/kafka_consumer.go`
- Modify: `cmd/events/main.go`

**Verification:** Manual testing with Kafka + `go test -race ./...`

---

### [ ] Step: Final Cleanup and Documentation

Final pass for consistency and documentation.

**Tasks:**
- [ ] Review all modified files for consistent style
- [ ] Ensure all new packages have package-level documentation
- [ ] Remove any dead code or unused imports
- [ ] Run full test suite
- [ ] Run linter if available
- [ ] Create report.md with summary of changes

**Files:**
- All modified files
- Create: `.zenflow/tasks/core-logic-refactoring-d887/report.md`

**Verification:** `make test` + manual review

---

## Notes

- Each step is designed to be incrementally testable
- Run tests after each step before proceeding
- If a step causes test failures, fix before moving forward
- The order of steps matters - later steps depend on earlier ones
