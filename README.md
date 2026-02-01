# events

A microservice that manages domain events for dev-platform.

## Stack
1. Golang 1.25
2. [golang-migrate](https://github.com/golang-migrate/migrate)
3. Docker (see Dockerfile)
4. Native golang worker implementation (without libs)
5. Docker Compose for local development

## How does it work
1. Worker reads Kafka topic
2. Worker writes event to its database
3. Service has handlers to search or export events

## Quick Start

```bash
# 1. Configuration is ready - .env file has sensible defaults
# 2. Start services with Docker Compose
docker compose up

# 3. Service runs on http://localhost:8080
curl http://localhost:8080/healthz
```

## Configuration

The service uses environment variables for configuration. A `.env` file is provided with defaults for local development.

Key configuration files:
- `.env` - Local development configuration (already configured)
- `.env.example` - Template with all available options and documentation

To customize configuration:
1. Edit `.env` file directly, or
2. Set environment variables before running the service

See `.env.example` for all available configuration options including:
- Server settings (port)
- PostgreSQL connection (host, port, credentials)
- Kafka settings (broker, topic, consumer group)
- Event retention and export limits