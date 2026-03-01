FROM golang:1.25 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY migrations/ migrations/
COPY pkg/ pkg/
COPY gen/ gen/

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o events ./cmd/server

FROM gcr.io/distroless/static:nonroot
WORKDIR /app

COPY --from=builder /workspace/events .
COPY --from=builder /workspace/migrations ./migrations

USER 65532:65532
EXPOSE 8080
EXPOSE 9090

ENTRYPOINT ["/app/events"]
