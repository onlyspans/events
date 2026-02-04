# Product Requirements Document: Project Simplification for Docker Compose Deployment

## 1. Background

### 1.1 Current Architecture

The current system is designed for production Kubernetes environments and consists of:

**Two Separate Binaries:**
- **HTTP Service** (`cmd/service/main.go`): REST API server on port 8080
- **Kafka Worker** (`cmd/worker/main.go`): Background Kafka consumer

**Infrastructure Dependencies:**
- PostgreSQL 17 (with JSONB, GIN indexes)
- Apache Kafka + Zookeeper (2 containers)
- Database migration runner (golang-migrate)

**Deployment Complexity:**
- 3 separate Docker images (service, worker, migrations)
- Helm charts with separate deployments for service and worker
- Complex Kafka setup with Zookeeper dependency
- Manual offset management in Kafka consumer
- Separate CI/CD pipelines for multiple images

**Production Features:**
- SASL/SCRAM authentication for Kafka
- Batch processing (100 events or 500ms timeout)
- Prometheus metrics exposure
- Cron-based retention service (runs in service binary)
- Connection pooling, health checks, graceful shutdown

### 1.2 Problem Statement

The current architecture is excellent for production Kubernetes environments but presents significant barriers for **user self-deployment via Docker Compose**:

1. **Infrastructure Overhead**: Users must run 5+ containers (postgres, zookeeper, kafka, service, worker, migrations)
2. **Configuration Complexity**: Multiple binaries require coordinated environment variable configuration
3. **Operational Complexity**: Managing two separate processes (service + worker) for a single logical application
4. **Resource Usage**: Kafka + Zookeeper consume significant memory/CPU for simple deployments
5. **Debugging Difficulty**: Distributed logs across multiple containers make troubleshooting harder

**Trade-off**: The current design prioritizes horizontal scalability and fault isolation at the cost of deployment simplicity.

## 2. Goals and Non-Goals

### 2.1 Goals

**Primary Objective**: Simplify the architecture to enable **easy self-deployment via Docker Compose** while maintaining core functionality.

**Specific Goals:**
1. Reduce infrastructure dependencies from 5+ containers to 2-3 containers
2. Consolidate service and worker into a single deployable binary
3. Eliminate or simplify Kafka dependency for basic deployments
4. Maintain backward compatibility with existing API endpoints
5. Preserve database schema and migration system
6. Keep existing features: event storage, search, export, retention, health checks, metrics

### 2.2 Non-Goals

1. **Not changing production deployment**: Production Kubernetes deployments can continue using the current multi-binary architecture
2. **Not removing advanced features**: Kafka integration, SASL auth, and batch processing remain available as optional features
3. **Not changing the data model**: PostgreSQL schema and JSONB-based event storage stay unchanged
4. **Not modifying API contracts**: Existing HTTP endpoints and request/response formats remain compatible

## 3. Proposed Simplifications

### 3.1 Simplification Analysis

Below is a comprehensive analysis of potential simplifications, ordered by impact:

#### 3.1.1 **[HIGH IMPACT] Merge Service and Worker into Single Binary**

**Current State:**
- Two separate binaries: `cmd/service/main.go` and `cmd/worker/main.go`
- Service runs HTTP server + retention cron
- Worker runs Kafka consumer
- Requires 2 containers in compose, 2 deployments in k8s

**Simplification Options:**

**Option A: Single Binary with Mode Flag**
- Add `--mode` flag: `service`, `worker`, or `all` (default)
- In `all` mode, runs both HTTP server and Kafka consumer in same process
- Users run single container for simple deployments
- Production can still run separate containers with mode flags

**Option B: HTTP Server with Embedded Consumer**
- Remove worker binary entirely
- Embed Kafka consumer as goroutine in service binary
- Single container, single process

**Recommendation**: **Option B** for maximum simplification
- **Pros**: Fewer moving parts, single container, simpler configuration, easier debugging
- **Cons**: Can't scale service and worker independently (but this is fine for self-hosted deployments)
- **Impact**: Reduces containers from 5 to 4, eliminates binary coordination

#### 3.1.2 **[HIGH IMPACT] Replace Kafka with Direct HTTP Ingestion**

**Current State:**
- Events arrive via Kafka (topic: `events`)
- Requires Kafka + Zookeeper (2 containers, ~500MB memory)
- Complex configuration (brokers, authentication, consumer groups)

**Simplification Options:**

**Option A: Add HTTP POST Endpoint for Event Ingestion**
- Add `POST /events/ingest` endpoint to directly insert events
- Kafka becomes optional (enabled via config flag)
- Clients can POST events directly to the service

**Option B: Keep Kafka but Make it Optional**
- Kafka consumer only starts if `KAFKA_ENABLED=true`
- If disabled, service only provides read/search/export APIs
- Users must use external tools to populate database

**Option C: Replace Kafka with Embedded Message Queue**
- Use in-memory queue (channel) or lightweight broker (NATS embedded)
- Removes Zookeeper dependency
- Still provides async processing benefits

**Recommendation**: **Option A** (with Option B as fallback)
- **Pros**: Eliminates 2 containers (kafka, zookeeper), removes ~500MB overhead, simplifies configuration
- **Cons**: Loses Kafka guarantees (at-least-once delivery, offset management), loses decoupling between producers and service
- **Impact**: Reduces containers from 4 to 2 (postgres, service)
- **Mitigation**: Keep Kafka support as optional feature (controlled by env var)

#### 3.1.3 **[MEDIUM IMPACT] Embed Database Migrations in Service Binary**

**Current State:**
- Separate migration container runs golang-migrate
- Requires coordination (service must wait for migrations)
- compose.yaml has dedicated `migrate` service

**Simplification Options:**

**Option A: Embed golang-migrate in Service Binary**
- Service binary runs migrations on startup before starting HTTP server
- Use `github.com/golang-migrate/migrate/v4` as library
- Remove separate migration container

**Option B: Use SQL-based Migration Bundling**
- Embed migration SQL files in Go binary using `embed` package
- Run migrations via `database/sql` on startup

**Recommendation**: **Option A**
- **Pros**: Removes 1 container, ensures migrations run automatically, simpler compose.yaml
- **Cons**: Service startup time increases slightly, harder to rollback migrations separately
- **Impact**: Reduces containers from 3 to 2 (postgres, service), eliminates migration coordination

#### 3.1.4 **[MEDIUM IMPACT] Simplify Configuration**

**Current State:**
- 25+ environment variables across Kafka, Postgres, Service configs
- Separate configs for service and worker
- Complex Kafka authentication options

**Simplification Options:**

**Option A: Reduce Required Environment Variables**
- Provide sensible defaults for all non-critical settings
- Only require: database connection, optional Kafka connection
- Document minimal `.env` file (5-10 variables)

**Option B: Support Configuration File**
- Add support for `config.yaml` or `config.toml`
- Environment variables override config file values

**Option C: Interactive Setup Script**
- Provide `setup.sh` that generates `.env` file interactively

**Recommendation**: **Option A**
- **Pros**: Easier to get started, fewer decisions for users
- **Cons**: Less flexibility (but can be overridden)
- **Impact**: Improves UX without architectural changes

#### 3.1.5 **[LOW IMPACT] Simplify Docker Compose Setup**

**Current State:**
- compose.yaml with 5+ services
- Volume mounts for data persistence
- Health checks and dependencies

**Simplification Options:**

**Option A: Provide Multiple compose.yaml Files**
- `compose.simple.yaml`: Just postgres + service (no Kafka)
- `compose.full.yaml`: Full stack with Kafka
- `compose.dev.yaml`: Development setup with hot reload

**Option B: Use Docker Compose Profiles**
- Default profile: minimal setup (postgres, service)
- `kafka` profile: adds Kafka + Zookeeper
- `dev` profile: adds development tools

**Recommendation**: **Option B**
- **Pros**: Single file, easy to extend, clear separation
- **Cons**: Requires understanding Docker Compose profiles
- **Impact**: Better UX, no architectural changes

#### 3.1.6 **[LOW IMPACT] Reduce Kubernetes Complexity**

**Current State:**
- Helm charts with separate deployments for service and worker
- 3 separate image repositories
- Complex resource limits, probes, service accounts

**Simplification Options:**

**Option A: Keep Helm Charts as-is**
- Helm is for production, complexity is justified
- Focus simplification efforts on Docker Compose

**Option B: Provide Simplified Kubernetes Manifests**
- Add `k8s/simple/` directory with basic deployments
- Use single image, single deployment
- For users who want k8s but not Helm complexity

**Recommendation**: **Option A**
- **Rationale**: Helm users expect and can handle complexity; Docker Compose users cannot
- **Impact**: No changes needed

### 3.2 **Simplification Priority Matrix**

| Simplification | Impact | Effort | Risk | Priority |
|----------------|--------|--------|------|----------|
| Merge service + worker | High | Medium | Low | **P0** |
| Make Kafka optional with HTTP ingestion | High | High | Medium | **P0** |
| Embed migrations in binary | Medium | Low | Low | **P1** |
| Simplify configuration | Medium | Low | Low | **P1** |
| Docker Compose profiles | Low | Low | Low | **P2** |
| Kubernetes simplification | Low | Low | Low | **P3** |

## 4. Proposed Architecture (Simplified)

### 4.1 Single Binary Architecture

```
┌─────────────────────────────────────────────┐
│         events (Single Binary)              │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │       HTTP Server (Port 8080)        │  │
│  │  - POST /events/search               │  │
│  │  - POST /events/export               │  │
│  │  - POST /events/ingest (NEW)         │  │
│  │  - GET/PUT /settings                 │  │
│  │  - GET /healthz, /readyz, /metrics   │  │
│  └──────────────────────────────────────┘  │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │   Kafka Consumer (Optional)          │  │
│  │   - Enabled if KAFKA_ENABLED=true    │  │
│  │   - Runs in separate goroutine       │  │
│  └──────────────────────────────────────┘  │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │   Retention Service (Cron)           │  │
│  └──────────────────────────────────────┘  │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │   Migration Runner (Startup)         │  │
│  │   - Runs before HTTP server starts   │  │
│  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
           │
           ▼
    ┌──────────────┐
    │  PostgreSQL  │
    └──────────────┘
```

### 4.2 Deployment Configurations

**Simple Deployment (Docker Compose):**
```yaml
services:
  postgres:
    image: postgres:17-alpine
    # ... config ...

  events:
    image: onlyspans/events:latest
    environment:
      - POSTGRES_HOST=postgres
      - POSTGRES_DB=events
      - POSTGRES_USER=events
      - POSTGRES_PASSWORD=events
      - KAFKA_ENABLED=false  # <-- Kafka disabled
    ports:
      - "8080:8080"
    depends_on:
      - postgres
```

**Full Deployment (with Kafka):**
```yaml
services:
  postgres: # ... same as above ...

  kafka:  # ... kafka + zookeeper ...

  events:
    image: onlyspans/events:latest
    environment:
      - POSTGRES_HOST=postgres
      - KAFKA_ENABLED=true
      - KAFKA_HOST=kafka
      - KAFKA_TOPIC=events
    ports:
      - "8080:8080"
```

### 4.3 API Changes

**New Endpoint for Direct Event Ingestion:**
```
POST /events/ingest
Content-Type: application/json

{
  "timestamp": "2025-01-15T10:30:00Z",
  "user_name": "john.doe",
  "category": "document",
  "action": "create",
  "document_name": "proposal.pdf",
  "project": "alpha",
  "environment": "production",
  "tenant": "acme",
  "details": {
    "file_size": 1024000,
    "mime_type": "application/pdf"
  },
  "correlation_id": "abc-123",
  "trace_id": "xyz-789"
}

Response: 201 Created
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2025-01-15T10:30:00Z"
}
```

**Batch Ingestion Support:**
```
POST /events/ingest/batch
Content-Type: application/json

{
  "events": [
    { /* event 1 */ },
    { /* event 2 */ },
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
    }
  ]
}
```

## 5. Migration Path

### 5.1 Backward Compatibility

**Production Deployments (Kubernetes with Helm):**
- Can continue using current architecture with no changes
- Service and worker remain separate deployments
- Kafka remains required dependency

**Docker Compose Deployments:**
- Migrate to simplified single-binary deployment
- Optionally keep Kafka if needed for integration with other systems
- Migration involves updating compose.yaml and env vars

### 5.2 Feature Flags

Introduce environment variables to control optional features:

```bash
# Kafka Consumer
KAFKA_ENABLED=true/false (default: false)

# Auto-run Migrations
AUTO_MIGRATE=true/false (default: true)

# Retention Service
RETENTION_ENABLED=true/false (default: true)
```

## 6. Success Metrics

### 6.1 Deployment Simplicity
- **Before**: 5+ containers, 25+ env vars, 3 Docker images
- **After**: 2 containers (postgres, events), ~10 env vars, 1 Docker image
- **Target**: New user can deploy in <5 minutes with 10-line compose.yaml

### 6.2 Resource Usage
- **Before**: ~600MB memory (kafka + zookeeper + service + worker)
- **After**: ~150MB memory (service only, with embedded consumer)
- **Target**: 75% reduction in memory footprint for simple deployments

### 6.3 User Experience
- **Metric**: Time to first successful deployment
- **Target**: Reduce from ~30 minutes to ~5 minutes
- **Metric**: Number of configuration decisions
- **Target**: Reduce from ~25 to ~5 required env vars

## 7. Open Questions

1. **Kafka Compatibility**: Should we maintain 100% compatible Kafka consumer behavior (manual offset management, batch processing) when Kafka is enabled?
   - **Recommendation**: Yes, keep existing Kafka behavior unchanged when enabled

2. **HTTP Ingestion Performance**: Will direct HTTP ingestion handle high event volumes (10K+ events/sec)?
   - **Recommendation**: Add rate limiting and document performance limits; users needing high throughput should use Kafka mode

3. **Migration Rollback**: How to handle migration rollbacks if migrations run automatically on startup?
   - **Recommendation**: Add `AUTO_MIGRATE=false` flag for production; provide separate migration container/command for controlled rollouts

4. **Multi-instance Deployments**: What happens if multiple instances run with embedded migrations?
   - **Recommendation**: Use PostgreSQL advisory locks in migration runner to prevent concurrent migrations

5. **Monitoring and Debugging**: Will single-binary deployment make debugging harder?
   - **Recommendation**: Maintain separate log prefixes for HTTP, consumer, retention components; expose component-specific metrics

## 8. Assumptions

1. **Primary use case** for simplification is self-hosted deployments by individual users or small teams, not large-scale production deployments
2. **Event volume** for simplified deployments will be <1000 events/sec (can be handled by HTTP ingestion)
3. **Users prefer simplicity** over advanced features like independent service/worker scaling
4. **Docker Compose** is the primary deployment method for simplified architecture
5. **Backward compatibility** with existing API endpoints and database schema is mandatory
6. **Production Kubernetes deployments** will continue using current architecture (no migration required)

## 9. Dependencies and Constraints

### 9.1 Dependencies
- PostgreSQL 17 with JSONB support (unchanged)
- golang-migrate/migrate library for embedded migrations
- Optional: Kafka client library (IBM Sarama) when Kafka mode is enabled

### 9.2 Constraints
- Must maintain existing database schema (no breaking changes)
- Must preserve all existing HTTP API endpoints
- Must support existing `.env` configuration format
- Cannot change Prometheus metrics names (for monitoring continuity)

## 10. Next Steps

1. **Validate Assumptions**: Discuss this PRD with stakeholders and users
2. **Prioritize Simplifications**: Decide which simplifications to implement (P0 vs P1 vs P2)
3. **Create Technical Specification**: Design implementation details for approved simplifications
4. **Build Prototype**: Implement single-binary architecture with optional Kafka
5. **Test and Validate**: Ensure backward compatibility and measure resource improvements
6. **Document Migration Path**: Create guides for migrating existing deployments
