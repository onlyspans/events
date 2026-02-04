# Build stage
FROM golang:1.25 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY internal/ internal/
COPY migrations/ migrations/

# Build unified binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o events ./cmd/events

# Runtime stage
FROM gcr.io/distroless/static:nonroot
WORKDIR /app

# Copy binary from builder
COPY --from=builder /workspace/events .

# Run as non-root user
USER 65532:65532

# Expose HTTP port
EXPOSE 8080

ENTRYPOINT ["/app/events"]
