# WebSocket Live Dashboard

A Node.js and Express web application demonstrating **full-duplex real-time communication** over WebSocket (WSS):

- Automatic SSL/TLS key and certificate generation on startup.
- WebSocket endpoint `/ws` (WSS) — bidirectional: server pushes live CPU stats, clients send commands.
- Supported client→server commands: `ping` (latency), `set-interval` (control push rate), `broadcast` (fanout to all clients).
- REST endpoint `/heartbeat` with API rate limiting (HTTP 429).
- Premium dark-mode dashboard with:
  - Live animated CPU time-series chart drawn on Canvas.
  - Round-trip latency tracker (ping/pong).
  - Configurable push interval slider (250ms – 5s).
  - Broadcast message panel (multi-client chat).
  - Real-time WebSocket event log.
- Exponential backoff auto-reconnect on disconnect.
- Integration tests covering connection, CPU push, ping/pong, interval control, broadcast, and error handling.

## Prerequisites

- **Node.js** ≥ 20
- **OpenSSL** (for self-signed certificate generation)

## Setup

1. Install dependencies:

   ```bash
   npm install
   ```

2. Generate SSL/TLS certificates and start the server:

   ```bash
   npm start
   ```

   The server will automatically generate a self-signed certificate in `certs/` on first run.

3. Open the dashboard in your browser:

   ```text
   https://localhost:60008
   ```

   > Your browser will warn about the self-signed certificate — this is expected. Click "Advanced" → "Proceed".

## Running Tests

```bash
npm test
```

Tests cover:

- REST `/heartbeat` endpoint (status, timestamp, uptime)
- WebSocket connection and welcome message
- Server-push CPU messages
- Ping → Pong round-trip with echo
- `set-interval` command and clamping (≥ 250ms)
- Broadcast fanout to multiple clients
- Unknown message type error handling

## Port

| Port  | Protocol | Endpoint |
| ------- | ---------- | ---------- |
| `60008` | WSS (HTTPS) | `wss://localhost:60008/ws` |

## Docker

```bash
docker build -t express-websocket-example .
docker run -p 60008:60008 express-websocket-example
```

## Architecture

```text
Browser
  │  wss://localhost:60008/ws
  ↓
HTTPS Server (Express + node:https)
  └── WebSocket Server (ws library, same HTTPS server)
        ├── CPU push loop (per client, configurable interval)
        ├── ping → pong handler
        ├── set-interval → interval-ack handler
        └── broadcast → fanout to all wss.clients
```

## WebSocket vs SSE

This example is a direct extension of the `node-express-https-example` (SSE) and `node-express-http2-example`:

| Feature | SSE | WebSocket |
| --------- | ----- | ----------- |
| Direction | Server → Client only | Full-duplex (both ways) |
| Protocol | HTTP/1.1+ | Separate WebSocket upgrade |
| Browser API | `EventSource` | `WebSocket` |
| Framing | Text only | Text or Binary |
| Reconnect | Automatic | Manual |
