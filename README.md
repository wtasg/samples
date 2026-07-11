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
