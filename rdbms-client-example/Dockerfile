# Stage 1: Build binary using root context to access local lib dependency
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy the client library
COPY rdbms-client-lib-example /app/rdbms-client-lib-example

# Copy the client app
COPY rdbms-client-example /app/rdbms-client-example

WORKDIR /app/rdbms-client-example
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o toydb-client ./cmd/client

# Stage 2: Runtime image
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/rdbms-client-example/toydb-client .
COPY --from=builder /app/rdbms-client-example/static ./static
EXPOSE 8080
CMD ["./toydb-client", "--port", "8080", "--db", "http://toydb-server:9090"]
