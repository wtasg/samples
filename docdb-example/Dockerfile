# Stage 1: Build binary using root context to access local lib dependency
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy the client library (since it's a replaced dependency in go.mod)
COPY docdb-client-lib-example /app/docdb-client-lib-example

# Copy the server directory
COPY docdb-example /app/docdb-example

WORKDIR /app/docdb-example
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o docdb-server ./cmd/server

# Stage 2: Runtime
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/docdb-example/docdb-server .
EXPOSE 60013
VOLUME ["/app/data"]
CMD ["./docdb-server", "--addr", "0.0.0.0:60013", "--data", "/app/data"]
