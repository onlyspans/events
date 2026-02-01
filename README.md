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