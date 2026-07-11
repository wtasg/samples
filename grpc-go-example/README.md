# gRPC Connect-Go Example

A Go web application demonstrating **gRPC** using the **Connect protocol** — completing the protocol trilogy (REST / SSE → WebSocket → gRPC):

- Protocol Buffers definition (`proto/monitor.proto`) with three RPC methods.
- **`GetStatus`** — Unary RPC: returns a server status snapshot.
- **`StreamCPU`** — Server-Streaming RPC: streams live CPU load samples at a configurable interval.
- **`Echo`** — Unary RPC: echoes the message back with server timestamp and length.
- Go server using `connectrpc.com/connect`, speaking **gRPC**, **gRPC-Web**, and the **Connect protocol** on a single port over HTTP/2 (h2c).
- No Envoy proxy required — the browser calls gRPC methods directly using `fetch`.
- CORS middleware so the browser frontend can call Connect endpoints directly.
- Premium dark-mode dashboard with:
  - Server status panel (unary `GetStatus`).
  - Live CPU streaming chart on Canvas (server-streaming `StreamCPU`).
  - Echo terminal with round-trip latency (unary `Echo`).
  - RPC event log.
- Go integration tests using the Connect test client.

## Prerequisites

- **Go** ≥ 1.21

## Setup

1. Download dependencies:

   ```bash
   go mod download
   ```

2. Run the server:

   ```bash
   go run ./cmd/server/main.go
   ```

3. Open the dashboard in your browser:

   ```text
   http://localhost:60009
   ```

## Running Tests

```bash
go test ./...
```

Tests cover:

- `GetStatus` returns correct fields (status, uptime, CPU count, Go version)
- `StreamCPU` delivers at least one sample with valid CPU percent
- `StreamCPU` clamps interval to minimum 200ms
- `StreamCPU` delivers multiple samples
- `Echo` echoes message with correct length
- `Echo` handles empty string and long messages (truncation)
- Concurrent `GetStatus` calls (10 goroutines)

## Port

| Port    | Protocol        | Notes                              |
| --------- | ----------------- | ------------------------------------ |
| `60009` | HTTP/2 (h2c)    | gRPC, gRPC-Web, Connect on one port |

## Docker

```bash
docker build -t grpc-go-example .
docker run -p 60009:60009 grpc-go-example
```

## Regenerating Protobuf Code

The generated code in `gen/` is committed to the repository. To regenerate after editing `proto/monitor.proto`:

1. Install `protoc` and the Go plugins:

   ```bash
   # Install protoc (https://github.com/protocolbuffers/protobuf/releases)
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
   ```

2. Run protoc:

   ```bash
   protoc \
     --proto_path=proto \
     --go_out=gen/monitor/v1 \
     --go_opt=paths=source_relative \
     --connect-go_out=gen/monitor/v1 \
     --connect-go_opt=paths=source_relative \
     monitor.proto
   ```

## Architecture

```text
Browser (fetch + Connect protocol)
  │  POST /monitor.v1.MonitorService/GetStatus  (JSON, unary)
  │  POST /monitor.v1.MonitorService/StreamCPU  (streaming)
  │  POST /monitor.v1.MonitorService/Echo       (JSON, unary)
  ↓
Go HTTP/2 Server (h2c)
  └── Connect-Go Handler
        ├── MonitorServiceServer.GetStatus
        ├── MonitorServiceServer.StreamCPU  (ticker per stream)
        └── MonitorServiceServer.Echo
```

## Connect vs gRPC vs gRPC-Web

| Feature | gRPC | gRPC-Web | Connect |
| --------- | ------ | ---------- | --------- |
| Transport | HTTP/2 | HTTP/1.1 via proxy | HTTP/1.1 or HTTP/2 |
| Browser support | ❌ (requires proxy) | ✓ (via Envoy) | ✓ (native fetch) |
| Streaming | ✓ | Server-streaming only | Server-streaming (HTTP/1.1+) |
| Proto required | ✓ | ✓ | ✓ |
| Proxy required | ✓ | ✓ | ❌ |
