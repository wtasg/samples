# Node.js + Express HTTP/3 & HTTPS CPU Monitor Example

This is a secure, high-performance Node.js and Express monitoring application that runs concurrently over **HTTP/3 (QUIC)** and **HTTPS (HTTP/1.1)**. 

The project uses the `@currentspace/http3` library (powered by Cloudflare's Rust-based `quiche` library) to serve requests over UDP using the HTTP/3 protocol, co-existing with standard HTTPS over TCP.

---

## 🚀 Key Features

* **HTTP/3 (QUIC) Port `60007`**: Connects over UDP via `@currentspace/http3` with an Express adapter wrapper.
* **HTTPS Port `60447`**: Serves fallback connections over TCP via native `https` module.
* **Health Endpoint `/heartbeat`**: Returns JSON details containing status, timestamp, and server uptime.
* **Instant CPU Metric `/load`**: Evaluates and returns the CPU load percentage over a brief 100ms window.
* **Live CPU Event Stream `/stream`**: Establishes Server-Sent Events (SSE) pushing real-time CPU measurements to clients every 1s.
* **Rate Limiting**: Rate-limiting protects the `/load` and `/heartbeat` routes (capped at 15 requests per 15 seconds per IP).
* **Premium Dashboard**: Neon dark-themed HTML/CSS/JS frontend dashboard with real-time SVG-like canvas charts, rate-limit meters, and dynamic protocol-badge detection (H3 vs HTTP/1.1).

---

## 🛠️ Installation & Setup

1. **Verify Node.js Version**:
   Ensure you are running Node.js **v24.0.0 or higher** (due to `@currentspace/http3` requirements).
   ```bash
   node -v
   ```

2. **Install Dependencies**:
   ```bash
   npm install
   ```

3. **Build SSL/TLS Certificates**:
   Run the build script to compile self-signed TLS certificates:
   ```bash
   npm run build
   ```

---

## 🖥️ Running the Application

Start the secure HTTP/3 and HTTPS listeners:
```bash
npm start
```

Logs will display listening status:
```text
HTTP/3 secure server listening on https://localhost:60007
HTTPS secure server running at: https://localhost:60447
```

* Open **[https://localhost:60007](https://localhost:60007)** in an HTTP/3-capable browser to view the dashboard over HTTP/3.
* Open **[https://localhost:60447](https://localhost:60447)** to view the dashboard over standard HTTPS.

---

## 🧪 Testing

Run the automated integration test suite:
```bash
npm test
```

The test runner will automatically:
1. Re-generate SSL/TLS credentials if missing.
2. Bind both servers to ephemeral random ports to prevent port clashes.
3. Test all endpoints (`/heartbeat`, `/load`, `/stream`, and rate limiters) over both native HTTP/3 QUIC connection and HTTPS fallback channels.
