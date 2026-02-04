# Task 9: Integrate Migrator into Binary Startup - Summary

## Objective
Integrate the embedded migration runner into the main application binary startup sequence, respecting the `AUTO_MIGRATE` feature flag.

## Changes Made

### 1. Updated `cmd/events/main.go`

#### Added Import
```go
"github.com/onlyspans/events/internal/migrator"
```

#### Added Migration Logic
Inserted migration step after database connection and before repository initialization:

```go
// Run database migrations if auto-migrate is enabled
if cfg.Features.AutoMigrate {
    if err := migrator.Run(cfg.Database.DSN()); err != nil {
        logger.Error("migration failed", "error", err)
        os.Exit(1)
    }
} else {
    logger.Info("auto-migrate disabled, skipping database migrations")
}
```

**Key Behaviors:**
- **AUTO_MIGRATE=true (default)**: Runs migrations automatically on startup
- **AUTO_MIGRATE=false**: Skips migrations, logs message
- **Migration failure**: Service exits with error (doesn't start)
- **No migrations needed**: Handled gracefully by migrator (ErrNoChange)

### 2. Created Integration Tests

**File**: `cmd/events/main_integration_test.go`

**Test Coverage:**
1. **TestStartupWithMigrations**: Verifies full startup sequence with auto-migrate
   - Migrations run on startup
   - Migrations are idempotent (can run multiple times safely)
   - Concurrent migrations are serialized (advisory lock works)

2. **TestStartupWithoutMigrations**: Verifies AUTO_MIGRATE=false mode
   - Service can start when migrations are pre-applied
   - No migration code runs when flag is disabled

**Test Infrastructure:**
- Uses testcontainers-go for real PostgreSQL instances
- Tests run against postgres:17-alpine (same as production)
- Full isolation between test runs
- Tests both migration success and idempotency

## Verification Results

### Unit/Integration Tests
```bash
go test ./cmd/events/... -v -race
```
**Result**: ✅ All tests pass
- TestStartupWithMigrations: PASS (5.01s)
- TestStartupWithoutMigrations: PASS (2.76s)

### Build Verification
```bash
make build
```
**Result**: ✅ Binary builds successfully

### Manual Testing

#### Test 1: AUTO_MIGRATE=true
```bash
POSTGRES_USER=events POSTGRES_PASSWORD=events POSTGRES_DB=events \
AUTO_MIGRATE=true KAFKA_ENABLED=false ./bin/events
```

**Logs:**
```
INFO: starting event logs service
INFO: feature flags kafka_enabled=false auto_migrate=true
INFO: database connection established
INFO: starting database migrations
INFO: acquired migration lock
INFO: no migrations to apply, database is up to date
INFO: released migration lock
INFO: retention service started
INFO: Kafka consumer disabled
INFO: starting HTTP server port=8080
```

**Database Verification:**
```sql
\dt
-- Result: events, settings, schema_migrations tables exist

SELECT * FROM schema_migrations;
-- Result: version=2, dirty=false
```

#### Test 2: AUTO_MIGRATE=false
```bash
POSTGRES_USER=events POSTGRES_PASSWORD=events POSTGRES_DB=events \
AUTO_MIGRATE=false KAFKA_ENABLED=false ./bin/events
```

**Logs:**
```
INFO: starting event logs service
INFO: feature flags kafka_enabled=false auto_migrate=false
INFO: database connection established
INFO: auto-migrate disabled, skipping database migrations
INFO: retention service started
INFO: Kafka consumer disabled
INFO: starting HTTP server port=8080
```

#### Test 3: Health Check After Migration
```bash
curl http://localhost:9090/healthz
```
**Result**: `{"status":"UP"}`

## Migration Flow

### Startup Sequence
1. Load configuration (reads `AUTO_MIGRATE` from env)
2. Connect to PostgreSQL
3. Test database connection (ping)
4. **[NEW]** Run migrations if `AUTO_MIGRATE=true`
   - Migrator acquires PostgreSQL advisory lock
   - Runs `golang-migrate/v4` with embedded migrations
   - Releases advisory lock
   - Handles `ErrNoChange` gracefully
5. Initialize repositories and services
6. Start retention service (cron)
7. Start Kafka consumer (if enabled)
8. Start HTTP server

### Error Handling
- **Migration failure**: Service exits with error code 1
- **Already migrated**: Logs "no migrations to apply" and continues
- **Concurrent migrations**: Advisory lock ensures serialization
- **AUTO_MIGRATE=false**: Skips migration step entirely

## Acceptance Criteria Status

- [x] Migrations run automatically on startup when `AUTO_MIGRATE=true`
- [x] Service exits with error if migrations fail
- [x] Service starts without migrations when `AUTO_MIGRATE=false`
- [x] Logs clearly indicate migration status
- [x] Multiple instances can start concurrently without migration conflicts

## Benefits

1. **Simplified Deployment**: No separate migration container needed
2. **Automatic Setup**: Fresh deployments work out-of-the-box
3. **Safe Concurrency**: Advisory locks prevent race conditions
4. **Flexible Configuration**: Can disable for pre-migrated databases
5. **Clear Logging**: Migration status visible in application logs
6. **Fail-Safe**: Service won't start with broken migrations

## Next Steps

**Task 10**: Remove the separate migration container from docker-compose.yaml now that migrations are embedded in the service binary.
