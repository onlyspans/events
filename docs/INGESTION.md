# Event Ingestion API

This document provides detailed information about event ingestion endpoints.

## Overview

The events service provides two methods for ingesting events:

1. **HTTP Ingestion** (Direct) - Use REST API endpoints to submit events directly
2. **Kafka Ingestion** (Streaming) - Publish events to Kafka topic for asynchronous processing

HTTP ingestion is available in both deployment modes (minimal and Kafka-enabled), while Kafka ingestion requires `KAFKA_ENABLED=true`.

## HTTP Ingestion Endpoints

### Single Event Ingestion

**Endpoint:** `POST /events/ingest`

**Content-Type:** `application/json`

**Description:** Ingest a single event directly via HTTP. The event is immediately validated and persisted to the database.

#### Request Schema

```json
{
  "user_name": "string (required)",
  "category": "string (required)",
  "action": "string (required)",
  "document_name": "string (optional)",
  "project": "string (optional)",
  "environment": "string (optional)",
  "tenant": "string (optional)",
  "correlation_id": "string (optional)",
  "trace_id": "string (optional)",
  "details": {
    "key": "value (optional, JSONB object)"
  }
}
```

#### Field Descriptions

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `user_name` | string | **Yes** | Username or user ID who triggered the event | `"john.doe"` |
| `category` | string | **Yes** | Event category (domain-specific classification) | `"document"`, `"user"`, `"system"` |
| `action` | string | **Yes** | Action performed | `"create"`, `"update"`, `"delete"` |
| `document_name` | string | No | Name of the document/resource affected | `"project-plan.pdf"` |
| `project` | string | No | Project identifier | `"awesome-project"` |
| `environment` | string | No | Environment where event occurred | `"production"`, `"staging"` |
| `tenant` | string | No | Tenant/organization identifier | `"acme-corp"` |
| `correlation_id` | string | No | Correlation ID for distributed tracing | `"550e8400-e29b-41d4-a716-446655440000"` |
| `trace_id` | string | No | Trace ID for distributed tracing | `"trace-123456"` |
| `details` | object | No | Additional structured data (stored as JSONB) | `{"size_bytes": 1024, "mime_type": "application/pdf"}` |

#### Automatic Fields

The following fields are automatically set by the service:

- `id` - UUID generated using `uuid.New()`
- `timestamp` - Current UTC timestamp (if not provided)

#### Request Example

```bash
curl -X POST http://localhost:8080/events/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "user_name": "john.doe",
    "category": "document",
    "action": "create",
    "document_name": "project-plan.pdf",
    "project": "awesome-project",
    "environment": "production",
    "tenant": "acme-corp",
    "correlation_id": "550e8400-e29b-41d4-a716-446655440000",
    "details": {
      "size_bytes": 2048576,
      "mime_type": "application/pdf",
      "version": 2
    }
  }'
```

#### Success Response

**Status Code:** `201 Created`

**Response Body:**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000"
}
```

#### Error Responses

**400 Bad Request** - Validation error

```json
{
  "error": "validation failed: user_name is required"
}
```

**500 Internal Server Error** - Server-side error

```json
{
  "error": "database connection failed"
}
```

---

### Batch Event Ingestion

**Endpoint:** `POST /events/ingest/batch`

**Content-Type:** `application/json`

**Description:** Ingest multiple events in a single request. Maximum batch size is 100 events. The endpoint processes all events and returns a detailed status for each.

#### Request Schema

```json
{
  "events": [
    {
      "user_name": "string (required)",
      "category": "string (required)",
      "action": "string (required)",
      ...
    }
  ]
}
```

#### Request Example

```bash
curl -X POST http://localhost:8080/events/ingest/batch \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      {
        "user_name": "john.doe",
        "category": "document",
        "action": "create",
        "document_name": "file1.pdf"
      },
      {
        "user_name": "jane.smith",
        "category": "document",
        "action": "update",
        "document_name": "file2.pdf"
      },
      {
        "user_name": "invalid",
        "category": "document"
      }
    ]
  }'
```

#### Success Response (Partial Success)

**Status Code:** `207 Multi-Status`

**Response Body:**
```json
{
  "success_count": 2,
  "failure_count": 1,
  "errors": [
    {
      "index": 2,
      "error": "validation failed: action is required"
    }
  ]
}
```

**Field Descriptions:**

| Field | Type | Description |
|-------|------|-------------|
| `success_count` | integer | Number of events successfully ingested |
| `failure_count` | integer | Number of events that failed validation or persistence |
| `errors` | array | List of errors with event index and error message |
| `errors[].index` | integer | Zero-based index of the failed event in the request array |
| `errors[].error` | string | Error message describing why the event failed |

#### Error Responses

**400 Bad Request** - Invalid request format or batch size exceeded

```json
{
  "error": "batch size exceeds maximum of 100 events"
}
```

**500 Internal Server Error** - Server-side error

```json
{
  "error": "database connection failed"
}
```

---

## Kafka Ingestion

**Requirements:** `KAFKA_ENABLED=true` in service configuration

**Topic:** `event-logs` (configurable via `KAFKA_TOPIC`)

**Message Format:** JSON (same schema as HTTP single event ingestion)

### Publishing Events to Kafka

Events published to the Kafka topic are consumed in batches (100 events or 500ms timeout) and persisted to the database.

#### Message Schema

The message body should match the HTTP single event ingestion schema:

```json
{
  "user_name": "john.doe",
  "category": "document",
  "action": "create",
  "document_name": "file.pdf",
  "project": "awesome-project",
  "environment": "production",
  "tenant": "acme-corp",
  "details": {
    "custom_field": "custom_value"
  }
}
```

#### Example: Publishing with kafka-console-producer

```bash
docker exec -it kafka kafka-console-producer \
  --broker-list localhost:9092 \
  --topic event-logs

# Then paste JSON messages (one per line):
{"user_name":"john.doe","category":"document","action":"create"}
```

#### Example: Publishing with Sarama (Go)

```go
import (
    "encoding/json"
    "github.com/IBM/sarama"
)

config := sarama.NewConfig()
config.Producer.Return.Successes = true

producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
if err != nil {
    log.Fatal(err)
}
defer producer.Close()

event := map[string]interface{}{
    "user_name": "john.doe",
    "category":  "document",
    "action":    "create",
}

message, _ := json.Marshal(event)
msg := &sarama.ProducerMessage{
    Topic: "events",
    Value: sarama.StringEncoder(message),
}

partition, offset, err := producer.SendMessage(msg)
if err != nil {
    log.Printf("Failed to send message: %v", err)
}
```

### Kafka Consumer Behavior

When `KAFKA_ENABLED=true`:

- **Consumer Group:** `events-group` (configurable via `KAFKA_GROUP_ID`)
- **Batch Processing:** Up to 100 events or 500ms timeout
- **Offset Management:** Manual commit after successful batch persistence
- **Error Handling:** Failed events are logged and skipped (offset still committed to avoid blocking)
- **Metrics:** Tracks received, processed, and failed event counts

### Authentication

When `KAFKA_USERNAME` and `KAFKA_PASSWORD` are set, SASL/SCRAM-SHA-512 authentication is automatically enabled:

```bash
KAFKA_USERNAME=user
KAFKA_PASSWORD=password
KAFKA_ENABLED=true
```

---

## Performance Considerations

### HTTP Ingestion

**Single Event Endpoint:**
- Best for: Real-time ingestion with immediate confirmation
- Latency: ~10-50ms per event (depends on database)
- Throughput: ~1000-2000 events/second (single instance)

**Batch Event Endpoint:**
- Best for: Bulk imports, high-throughput scenarios
- Latency: ~50-200ms per batch (depends on batch size)
- Throughput: ~5000-10000 events/second (single instance)
- Recommended batch size: 50-100 events

### Kafka Ingestion

- Best for: Asynchronous, decoupled event streaming
- Latency: Eventual consistency (500ms-2s delay)
- Throughput: ~10000+ events/second (depends on Kafka cluster)
- Recommended for: High-volume production systems

---

## Best Practices

### When to Use HTTP Ingestion

1. **Simple Deployments:** No need for Kafka infrastructure
2. **Synchronous Workflows:** Need immediate confirmation of event persistence
3. **Low to Medium Volume:** <5000 events/second
4. **Development/Testing:** Quick setup without dependencies

### When to Use Kafka Ingestion

1. **High Volume:** >10000 events/second
2. **Asynchronous Workflows:** Don't need immediate confirmation
3. **Decoupled Architecture:** Multiple producers/consumers
4. **Production Systems:** Need durability, replication, and fault tolerance

### Data Validation

Both ingestion methods enforce the same validation rules:

1. **Required Fields:** `user_name`, `category`, `action` must be non-empty
2. **Field Length:** No explicit limits (reasonable database limits apply)
3. **JSONB Details:** Must be valid JSON object
4. **Batch Size:** Maximum 100 events per batch request

### Error Handling

**HTTP Ingestion:**
- Returns immediate error response
- Client can retry failed requests

**Kafka Ingestion:**
- Failed events are logged and skipped
- Check service logs for error details
- Monitor `events_failed_total` metric

---

## Monitoring

### Prometheus Metrics

All ingestion methods expose metrics on `/metrics` endpoint:

| Metric | Type | Description |
|--------|------|-------------|
| `events_received_total` | Counter | Events received from Kafka |
| `events_batches_processed_total` | Counter | Batches successfully persisted |
| `events_failed_total` | Counter | Events that failed to process |

### Health Checks

- **Liveness:** `GET /healthz` - Always returns 200
- **Readiness:** `GET /readyz` - Checks database connectivity

---

## Examples

### Full HTTP Ingestion Workflow

```bash
# 1. Ingest single event
EVENT_ID=$(curl -s -X POST http://localhost:8080/events/ingest \
  -H "Content-Type: application/json" \
  -d '{
    "user_name": "john.doe",
    "category": "document",
    "action": "create",
    "project": "awesome-project"
  }' | jq -r '.id')

echo "Event created with ID: $EVENT_ID"

# 2. Search for the event
curl -X POST http://localhost:8080/events \
  -H "Content-Type: application/json" \
  -d '{
    "user_name": "john.doe",
    "project": "awesome-project",
    "page": 1,
    "page_size": 10
  }'

# 3. Export events to CSV
curl -X POST http://localhost:8080/events/export \
  -H "Content-Type: application/json" \
  -d '{
    "project": "awesome-project"
  }' > events.csv
```

### Batch Ingestion with Error Handling

```bash
# Ingest batch with mixed valid/invalid events
RESPONSE=$(curl -s -X POST http://localhost:8080/events/ingest/batch \
  -H "Content-Type: application/json" \
  -d '{
    "events": [
      {
        "user_name": "john.doe",
        "category": "document",
        "action": "create"
      },
      {
        "user_name": "jane.smith",
        "category": "document"
      }
    ]
  }')

# Parse response
SUCCESS_COUNT=$(echo $RESPONSE | jq -r '.success_count')
FAILURE_COUNT=$(echo $RESPONSE | jq -r '.failure_count')

echo "Successfully ingested: $SUCCESS_COUNT events"
echo "Failed: $FAILURE_COUNT events"

if [ $FAILURE_COUNT -gt 0 ]; then
  echo "Errors:"
  echo $RESPONSE | jq -r '.errors[]'
fi
```

---

## Troubleshooting

### Common Issues

**1. Validation Errors**

```json
{"error": "validation failed: user_name is required"}
```

**Solution:** Ensure all required fields (`user_name`, `category`, `action`) are present and non-empty.

---

**2. Batch Size Exceeded**

```json
{"error": "batch size exceeds maximum of 100 events"}
```

**Solution:** Split large batches into chunks of ≤100 events.

---

**3. Database Connection Errors**

```json
{"error": "database connection failed"}
```

**Solution:** Check database connectivity via `/readyz` endpoint. Verify `POSTGRES_*` environment variables.

---

**4. Kafka Consumer Not Processing Events**

**Symptoms:** Events published to Kafka topic but not appearing in database.

**Checklist:**
- Is `KAFKA_ENABLED=true`?
- Is Kafka service healthy? Check `docker compose ps`
- Check service logs: `docker compose logs events`
- Verify topic name: `KAFKA_TOPIC` matches your producer
- Check consumer metrics: `curl http://localhost:8080/metrics | grep events_received`

---

## See Also

- [Configuration Guide](CONFIGURATION.md) - Environment variable reference
- [README](../README.md) - Project overview and quick start
- [CLAUDE.md](../CLAUDE.md) - Development guide for AI assistants
