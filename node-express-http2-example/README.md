# HTTP/2 & HTTPS Dual Express CPU Monitor Example

This is a self-contained, secure Node.js + Express application served simultaneously over HTTP/2 (ALPN `h2`) and standard HTTPS (HTTP/1.1 fallback) using Node's native network modules and the `http2-express` compatibility bridge.

It showcases:
1. **Dual-Protocol Listening**:
   - **HTTP/2 Server** listening on port `60006`.
   - **HTTPS Server** (HTTP/1.1) listening on port `60446`.
2. **Build-Time SSL/TLS Setup**: Generates a self-signed key/cert pair via OpenSSL.
3. **Endpoints**:
   - **Heartbeat** (`/heartbeat`): Server health, timestamp, and uptime.
   - **CPU Load (Query/Response)** (`/load`): Returns instant CPU load calculated over a 100ms window.
   - **CPU Load (Streaming)** (`/stream`): Uses Server-Sent Events (SSE) to push real-time CPU load updates every 1 second.
4. **Rate Limiting**: Limits incoming API traffic on `/load` and `/heartbeat` to prevent spamming (returns HTTP 429).
5. **Interactive Dashboard**: A beautiful, premium dark mode frontend dashboard built with vanilla CSS and canvas charting that visualizes real-time data, displays rate limit exceptions, and **dynamically detects and displays whether your connection is using HTTP/2 or HTTP/1.1**!

---

## File Structure

```text
node-express-http2-example/
├── certs/                 # Automatically created; contains key.pem and cert.pem
├── public/                # Static frontend assets
│   ├── index.html         # Dashboard template
│   ├── styles.css         # Custom styles, animations, and dark theme
│   └── app.js             # Live SSE chart, gauge, and protocol detection
├── scripts/
│   └── generate-certs.js  # SSL certificate generator run at build step
├── server.js              # HTTP/2 and HTTPS Express server setup
├── test.js                # Integration test suite
└── package.json           # Scripts and dependencies
```

---

## Installation & Running

### Prerequisites

Make sure you have [Node.js](https://nodejs.org/) (v19+ recommended for HTTP/2 support) and `openssl` installed.

### Setup

Install the required npm packages:

```bash
npm install
```

### Build Step

Generate the self-signed SSL/TLS certificates:

```bash
npm run build
```

### Run the Server

Start both the HTTP/2 and HTTPS servers (the `prestart` hook will automatically execute `npm run build` beforehand if not already generated):

```bash
npm start
```

The console will output:
```text
HTTP/2 secure server running at: https://localhost:60006
HTTPS secure server running at: https://localhost:60446
```

### Accessing the Dashboards

Since it uses self-signed SSL certificates, your browser will display a security warning (e.g., "Your connection is not private"). Click **Advanced** -> **Proceed to localhost** to bypass it.

1. **HTTP/2 Connection**: Open **[https://localhost:60006/](https://localhost:60006/)**. The dashboard's protocol badge in the top-left will show **H2**.
2. **HTTP/1.1 Connection**: Open **[https://localhost:60446/](https://localhost:60446/)**. The dashboard's protocol badge will show **HTTP/1.1**.

---

## Endpoint API Reference

### 1. Heartbeat
- **Path**: `GET /heartbeat`
- **Rate Limited**: Yes (max 15 requests/15 seconds)
- **Response Status**: `200 OK`
- **Response Body**:
  ```json
  {
    "status": "alive",
    "timestamp": "2026-07-11T17:15:00.000Z",
    "uptime": 45.67
  }
  ```

### 2. Instant CPU Load
- **Path**: `GET /load`
- **Rate Limited**: Yes (max 15 requests/15 seconds)
- **Response Status**: `200 OK` (or `429 Too Many Requests`)
- **Response Body (200)**:
  ```json
  {
    "cpu": 14.82,
    "timestamp": "2026-07-11T17:15:02.123Z"
  }
  ```

### 3. CPU Load Stream
- **Path**: `GET /stream`
- **Headers**: `Content-Type: text/event-stream`, `Cache-Control: no-cache`
- **Pushed Events**: Sent every 1 second.
  ```text
  data: {"cpu":5.12,"timestamp":"2026-07-11T17:15:03.000Z"}

  data: {"cpu":7.89,"timestamp":"2026-07-11T17:15:04.000Z"}
  ```

---

## Running Integration Tests

Run the automated test runner to verify both the HTTP/2 and HTTPS listeners concurrently:

```bash
npm test
```
