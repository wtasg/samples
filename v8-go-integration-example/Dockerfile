# Build stage
FROM golang:1.26-bookworm AS builder

WORKDIR /app

# Copy dependency definition files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go app with CGO enabled (v8go requires CGO)
ENV CGO_ENABLED=1
RUN go build -o server ./src/server/main.go

# Run stage
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the compiled server binary
COPY --from=builder /app/server .

# Copy the static web client directory
COPY --from=builder /app/src/client ./src/client

# Default port
EXPOSE 60001

# Run the server
CMD ["./server"]
