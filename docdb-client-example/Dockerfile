# Stage 1: Build binary using root context to access local lib dependency
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy the client library
COPY docdb-client-lib-example /app/docdb-client-lib-example

# Copy the client app
COPY docdb-client-example /app/docdb-client-example

WORKDIR /app/docdb-client-example
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o docdb-client ./cmd/client

# Stage 2: Runtime image
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/docdb-client-example/docdb-client .
COPY --from=builder /app/docdb-client-example/static ./static
EXPOSE 8080
CMD ["./docdb-client", "--port", "8080", "--db", "http://docdb-server:60013"]
