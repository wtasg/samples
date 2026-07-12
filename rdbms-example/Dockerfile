# Stage 1: Build binary using root context to access local lib dependency
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy the client library (since it's a replaced dependency in go.mod)
COPY rdbms-client-lib-example /app/rdbms-client-lib-example

# Copy the server directory
COPY rdbms-example /app/rdbms-example

WORKDIR /app/rdbms-example
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o toydb-server ./cmd/server

# Stage 2: Runtime
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/rdbms-example/toydb-server .
EXPOSE 9090
VOLUME ["/app/data"]
CMD ["./toydb-server", "--addr", "0.0.0.0:9090", "--data", "/app/data"]
