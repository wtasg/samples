# Sample Projects

## Projects

### 1. [V8 + Go Integration Example](/v8-go-integration-example/README.md)

A web application demonstrating:

- In-memory JavaScript sandboxed execution in Go using V8 isolates.
- Bidirectional callbacks between JavaScript and Go (capturing console outputs, performing calculations in Go, fetching resources via host HTTP client).
- Responsive web interface with 50/50 layout, split-screen resizable, multi-tab scratchpad.
- Complete functional E2E tests.

For detailed setup, installation, and run instructions, refer to the [V8 + Go Integration Example README](/v8-go-integration-example/README.md).

### 2. [Electron Canvas Studio](/electron-canvas-example/README.md)

An Electron desktop application showcasing HTML5 Canvas:

- Supports drawing lines, circles, squares, rectangles, and freehand paths.
- Includes undo/redo history, canvas clearing, and PNG export.
- Saves canvas drawing state during window resizing.
- Includes Jasmine unit tests and Playwright E2E integration tests.

For detailed setup, installation, and run instructions, refer to the [Electron Canvas Studio README](/electron-canvas-example/README.md).

### 3. [Graphviz Flow](/dynamic-graphviz-svg-example/README.md)

A python FastAPI web application for dynamically rendering and building Graphviz SVG diagrams:

- Live code editor for Graphviz DOT language with real-time compilation.
- Drag-and-drop shape palette (Rectangle, Ellipse, Circle, Diamond, Database) to add nodes.
- Interactive drag-and-drop arrow creation between nodes in the SVG.
- Move and reposition nodes dynamically (in coordinate-based layouts like `neato`).
- Properties inspector to customize node/edge labels, shapes, and colors.
- Pan and zoom support with a beautiful grid viewport.
- Complete backend integration tests using pytest.

For detailed setup, installation, and run instructions, refer to the [Graphviz Flow README](/dynamic-graphviz-svg-example/README.md).

### 4. [MCP Command Server](/py-mcp-ls-example/README.md)

A python Model Context Protocol (MCP) server implementing a dynamic plugin architecture:

- Central CRUDL (Create, Read, Update, Delete, List) Plugin Registry.
- Dynamic runtime plugin loading from the `plugins/` directory.
- Implements the `ls` tool as a dynamic plugin with safe subprocess invocation.
- Supports running over standard I/O (stdio) transport or Server-Sent Events (SSE) HTTP transport.
- Configurable port (`60004`) for the publicly exposed SSE listener.

For detailed setup, installation, and run instructions, refer to the [MCP Command Server README](/py-mcp-ls-example/README.md).

### 5. [Secure Express CPU Monitor Example](/node-express-https-example/README.md)

A Node.js and Express web application running over HTTPS:

- Automatic SSL/TLS key and certificate generation on startup.
- Endpoint `/heartbeat` to check server health and uptime.
- Endpoint `/api/cpu/load` for query/response CPU load calculation.
- Endpoint `/api/cpu/stream` to stream real-time CPU load using Server-Sent Events (SSE).
- API rate limiting protecting endpoints (returning HTTP 429).
- Premium system monitoring dark-mode dashboard visualizing CPU streams on a canvas chart.

For detailed setup, installation, and run instructions, refer to the [Secure Express CPU Monitor README](/node-express-https-example/README.md).

### 6. [HTTP/2 & HTTPS Dual Express Example](/node-express-http2-example/README.md)

A Node.js and Express web application running over HTTP/2 (TLS) and HTTPS co-existing:

- Automatic SSL/TLS key and certificate generation on startup.
- Endpoint `/heartbeat` to check server health and uptime.
- Endpoint `/load` for query/response CPU load calculation.
- Endpoint `/stream` to stream real-time CPU load using Server-Sent Events (SSE).
- API rate limiting protecting endpoints (returning HTTP 429).
- Premium system monitoring dark-mode dashboard visualizing CPU streams on a canvas chart and detecting the connection protocol in real-time.

For detailed setup, installation, and run instructions, refer to the [HTTP/2 & HTTPS Dual Express README](/node-express-http2-example/README.md).

### 7. [HTTP/3 & HTTPS Dual Express Example](/node-express-http3-example/README.md)

A Node.js and Express web application running over HTTP/3 (QUIC) and HTTPS:

- Automatic SSL/TLS key and certificate generation on startup.
- Endpoint `/heartbeat` to check server health and uptime.
- Endpoint `/load` for query/response CPU load calculation.
- Endpoint `/stream` to stream real-time CPU load using Server-Sent Events (SSE).
- API rate limiting protecting endpoints (returning HTTP 429).
- Premium system monitoring dark-mode dashboard visualizing CPU streams on a canvas chart and displaying connection protocol dynamically.

For detailed setup, installation, and run instructions, refer to the [HTTP/3 & HTTPS Dual Express README](/node-express-http3-example/README.md).

### 8. [Express Server Benchmarking Suite](/benchmark-node-express-servers-example/README.md)

A Dockerized load-testing and verification suite to compare performance across different HTTP protocol versions:

- Isolates tests inside local virtual networks so benchmarks do not disrupt running host systems.
- Compares HTTP/1.1 (HTTPS), HTTP/2 (Multiplexed), and HTTP/3 (QUIC) protocols under varying concurrency levels.
- Employs self-healing client connection-recycling QUIC logic to tolerate extreme load spikes.
- Auto-generates Markdown comparison tables reporting requests/sec and latency percentiles.

For detailed setup and test instructions, refer to the [Express Server Benchmarking Suite README](/benchmark-node-express-servers-example/README.md).

### 9. [WebSocket Live Dashboard](/node-express-websocket-example/README.md)

A Node.js and Express web application demonstrating full-duplex real-time communication over WebSocket (WSS):

- Automatic SSL/TLS key and certificate generation on startup.
- WebSocket endpoint `/ws` (WSS) — server pushes live CPU stats, clients send commands.
- Supported client→server message types: `ping` (latency), `set-interval` (push rate), `broadcast` (fanout).
- Premium dark-mode dashboard with live CPU chart, round-trip latency tracker, configurable interval slider, broadcast panel.
- Exponential backoff auto-reconnect on disconnect.
- Integration tests covering connection, CPU push, ping/pong, interval control, broadcast, and error handling.

For detailed setup, installation, and run instructions, refer to the [WebSocket Live Dashboard README](/node-express-websocket-example/README.md).

### 10. [gRPC Connect-Go Dashboard](/grpc-go-example/README.md)

A Go web application demonstrating gRPC using the Connect protocol — completing the protocol trilogy (HTTPS → HTTP/2 → HTTP/3 → WebSocket → gRPC):

- Protocol Buffers definition with three RPC methods: `GetStatus` (unary), `StreamCPU` (server-streaming), `Echo` (unary).
- Go server using `connectrpc.com/connect`, speaking gRPC, gRPC-Web, and Connect protocol on a single port.
- No Envoy proxy required — the browser calls gRPC methods directly via `fetch`.
- Premium dark-mode dashboard with server status card, live CPU streaming chart, Echo terminal with RTT display.
- Go integration tests using the Connect test client.

For detailed setup, installation, and run instructions, refer to the [gRPC Connect-Go README](/grpc-go-example/README.md).

### 11. [WebAssembly Browser Explorer](/wasm-browser-example/README.md)

A web application demonstrating Rust compiled to WebAssembly, running native-speed computation in the browser — the browser-side counterpart to the V8-in-Go example:

- Rust crate compiled to WASM via `wasm-pack` + `wasm-bindgen`.
- **Mandelbrot Renderer** — full Mandelbrot set rendered on Canvas entirely in WASM.
- **Fibonacci Race** — benchmarks `fibonacci(n)` × 10,000 calls in JS vs WASM side-by-side.
- **Prime Sieve Benchmark** — Sieve of Eratosthenes (up to 5M) in JS vs WASM.
- **FNV-1a Hash Terminal** — computes 64-bit FNV-1a hash of arbitrary text in Rust.
- 12 Rust unit tests + 10 integration tests covering server endpoints and WASM artifact validity.

For detailed setup, installation, and run instructions, refer to the [WebAssembly Browser Explorer README](/wasm-browser-example/README.md).

### 12. [ToyDB Studio](/rdbms-client-example/README.md)

A web client interface and relational database daemon demonstrating pure Go implementation of database internals:

- Exposes all operations over **Connect-RPC / gRPC** binary wire protocol.
- Integrates all five custom data structures: **B+ Tree** (indexes), **Red-Black Tree** (sorting), **Trie** (catalog & prefix search), **Bloom Filter** (existence gate), and **Rabin-Karp** (rolling-hash search).
- **Interactive web UI dashboard** built with clean HTML/CSS/JS that communicates via the `rdbms-client-lib` client library.
- Supports writing query templates, executing SQL, viewing live table schemas, column metadata, primary key definitions, and execution runtimes in a responsive layout.

For detailed setup, installation, and run instructions, refer to the [ToyDB Studio README](/rdbms-client-example/README.md).

### 13. [DocDB Studio](/docdb-client-example/README.md)

A web client interface and NoSQL document database daemon demonstrating pure Go implementation of document database internals:

- Exposes all operations over **Connect-RPC / gRPC** binary wire protocol.
- Integrates all five custom data structures: **LSM Tree** (storage engine), **Hash Map** (O(1) ID lookup), **Skip List** (memtable & sorting), **Bloom Filter** (per-SST lookup gate), and **Inverted Index** (secondary field index).
- **Interactive web UI dashboard** built with clean HTML/CSS/JS that communicates via the `github.com/docdb/client` client library.
- Supports writing query templates, executing Javascript-style command pipelines, viewing live collections, document counts, database sizes, and toggling between formatted JSON or flattened table views.

For detailed setup, installation, and run instructions, refer to the [DocDB Studio README](/docdb-client-example/README.md).

### 14. [Docker Sidecar Observability Example](/docker-sidecar-example/README.md)

A multi-container Docker example demonstrating the sidecar pattern for observability:

- **Nginx Proxy Sidecar**: Shares the network namespace of the Go application, acting as the external ingress endpoint (`60015`).
- **Prometheus Scraper Sidecar**: Shares the network namespace of the Go application, scraping metrics from `localhost` and exposing its UI on `60016`.
- **Fluent Bit Logging Sidecar**: Tails log files in a shared Docker volume, parsing and outputting enriched logs to console/stdout.
- **Grafana Dashboard**: Preconfigured monitoring dashboard listening on `60017` connecting to Prometheus.

For detailed setup, installation, and run instructions, refer to the [Docker Sidecar Observability README](/docker-sidecar-example/README.md).

## Ports Reference

1. `60001` : [V8 + Go Integration Example](/v8-go-integration-example/README.md)
2. Desktop App: [Electron Canvas Studio](/electron-canvas-example/README.md) (Runs locally)
3. `60003` : [Graphviz Flow](/dynamic-graphviz-svg-example/README.md)
4. `60004` : [MCP Command Server](/py-mcp-ls-example/README.md)
5. `60005` : [Secure Express CPU Monitor Example](/node-express-https-example/README.md)
6. `60006` : [HTTP/2 & HTTPS Dual Express Example (HTTP/2)](/node-express-http2-example/README.md)
7. `60446` : [HTTP/2 & HTTPS Dual Express Example (HTTPS fallback)](/node-express-http2-example/README.md)
8. `60007` : [HTTP/3 & HTTPS Dual Express Example (HTTP/3)](/node-express-http3-example/README.md)
9. `60447` : [HTTP/3 & HTTPS Dual Express Example (HTTPS fallback)](/node-express-http3-example/README.md)
10. `60008` : [WebSocket Live Dashboard](/node-express-websocket-example/README.md)
11. `60009` : [gRPC Connect-Go Dashboard](/grpc-go-example/README.md)
12. `60010` : [WebAssembly Browser Explorer](/wasm-browser-example/README.md)
13. `60011` : [ToyDB Daemon Server (gRPC)](/rdbms-example/README.md)
14. `60012` : [ToyDB Studio Web UI Client](/rdbms-client-example/README.md)
15. `60013` : [DocDB Daemon Server (gRPC)](/docdb-example/README.md)
16. `60014` : [DocDB Studio Web UI Client](/docdb-client-example/README.md)
17. `60015` : [Docker Sidecar Example (Nginx Ingress)](/docker-sidecar-example/README.md)
18. `60016` : [Docker Sidecar Example (Prometheus UI)](/docker-sidecar-example/README.md)
19. `60017` : [Docker Sidecar Example (Grafana Dashboard)](/docker-sidecar-example/README.md)


## Docker Deployment

This repository includes a root `docker-compose.yml` to run the web applications together.

### Running with Docker Compose

1. Copy the sample environment file to create your active `.env` configuration:

   ```bash
   cp sample.env .env
   ```

2. Configure the ports in the `.env` file (defaults are `60001` for V8 + Go and `60003` for Graphviz Flow).
3. Build and launch all services in detached mode:

   ```bash
   docker compose up --build --detach
   ```

4. Verify the services are running:

   ```bash
   docker compose ps
   ```

5. To stop the running containers:

   ```bash
   docker compose down
   ```
