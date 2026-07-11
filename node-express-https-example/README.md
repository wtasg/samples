# Node.js + Express HTTPS CPU Monitor Example

This is a self-contained, secure Node.js + Express application served over HTTPS. It showcases:

1. **Dynamic SSL/TLS certificate generation**: Generates a self-signed key/cert pair on startup if not present.
2. **Heartbeat endpoint**: Simple ping route indicating status, uptime, and latency.
3. **CPU load monitoring**:
   - **Query/Response form** (`/api/cpu/load`): Returns instant CPU load calculated over a 100ms window to avoid event loop blocking.
   - **Streaming form** (`/api/cpu/stream`): Uses Server-Sent Events (SSE) to push real-time CPU load updates every 1 second.
4. **Rate limiting**: Limits incoming API traffic to demonstrate request constraints (returns HTTP 429 upon exceeding limits).
5. **Interactive Dashboard**: A beautiful, premium dark mode frontend dashboard built with vanilla CSS and canvas charting that visualizes real-time data and displays rate limit exceptions.

---

## File Structure

```text
node-express-https-example/
├── certs/                 # Automatically created; contains key.pem and cert.pem
├── public/                # Static frontend assets
│   ├── index.html         # Dashboard template
│   ├── styles.css         # Styling, animations, and dark theme
│   └── app.js             # EventSource stream connection & gauge animations
├── cert-generator.js      # SSL certificate check & creation
├── server.js              # Express routing, rate limiter, & HTTPS startup
├── test.js                # Integration test suite
└── package.json           # Scripts and dependencies
```

---

## Installation & Running

### Prerequisites

Make sure you have [Node.js](https://nodejs.org/) and `openssl` (for certificate generation) installed.

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

This script generates `key.pem` and `cert.pem` inside the `certs/` directory using OpenSSL.

### Run the Server

Start the HTTPS server (the `prestart` hook will automatically execute `npm run build` beforehand to compile the certificates if not already generated):

```bash
npm start
```
```

### Accessing the Dashboard

1. Open your browser and navigate to: **[https://localhost:60005/](https://localhost:60005/)**
2. Since it uses a self-signed SSL certificate, your browser will display a security warning (e.g., "Your connection is not private").
3. Bypass this warning (click **Advanced** -> **Proceed to localhost**) to view the secure dashboard.

---

## Endpoint API Reference

All requests must be made over **HTTPS**.

### 1. Heartbeat

- **Path**: `GET /heartbeat`

- **Rate Limited**: Yes (shared limit)
- **Response Status**: `200 OK`
- **Response Body**:

  ```json
  {
    "status": "alive",
    "timestamp": "2026-07-11T16:40:00.000Z",
    "uptime": 23.45
  }
  ```

### 2. Instant CPU Load (Query/Response)

- **Path**: `GET /api/cpu/load`

- **Rate Limited**: Yes (shared limit)
- **Response Status**: `200 OK` (or `429 Too Many Requests` if rate limit exceeded)
- **Response Body (200)**:

  ```json
  {
    "cpu": 12.34,
    "timestamp": "2026-07-11T16:40:02.123Z"
  }
  ```

- **Response Body (429)**:

  ```json
  {
    "error": "Too many requests",
    "message": "Rate limit exceeded. Please wait a few seconds and try again."
  }
  ```

### 3. Real-Time CPU Load Stream (SSE)

- **Path**: `GET /api/cpu/stream`

- **Headers returned**: `Content-Type: text/event-stream`, `Connection: keep-alive`
- **Pushed Events**: Sent every 1 second.

  ```text
  data: {"cpu":8.45,"timestamp":"2026-07-11T16:40:03.000Z"}

  data: {"cpu":11.23,"timestamp":"2026-07-11T16:40:04.000Z"}
  ```

---

## Rate Limiting Specifications

- **Configuration**: Shared rate limiter for `/api/cpu/load` and `/heartbeat`.
- **Rules**: Maximum 15 requests per 15-second window per IP.
- **Demonstrating Rate Limits in the Dashboard**:
  - Click the **Query CPU Load** button repeatedly.
  - The request count meter on the card will decrease.
  - Upon hitting 15 requests within 15 seconds, the server will respond with HTTP 429.
  - The dashboard will display a red alert banner showing "Rate Limit Hit".

---

## Running Integration Tests

Run the automated test runner to verify HTTPS connectivity, route payloads, rate limiting, and Server-Sent Events parsing:

```bash
npm test
```
