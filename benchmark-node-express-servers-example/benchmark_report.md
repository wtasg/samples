# Express Servers Protocol Benchmark Report

Generated at: `2026-07-11T13:05:31.807Z`
Parameters:
- **Total Requests**: `2000`
- **Concurrency**: `50`

---

## 📊 Comparison Table

| Protocol / Port Configuration | Requests/Sec | Avg Latency | Min Latency | Max Latency | Success Rate |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **HTTP/1.1 (HTTPS Only Project)** | `4090.0 req/s` | `11.8ms` | `2.0ms` | `149.0ms` | `100.0%` |
| **HTTPS Fallback (H2 Dual-Protocol Port)** | `5698.0 req/s` | `8.5ms` | `3.0ms` | `188.0ms` | `100.0%` |
| **HTTP/2 (H2 Dual-Protocol Port)** | `7272.7 req/s` | `6.8ms` | `4.0ms` | `21.0ms` | `100.0%` |
| **HTTPS Fallback (QUIC Dual-Protocol Port)** | `6756.8 req/s` | `7.2ms` | `1.0ms` | `156.0ms` | `100.0%` |
| **HTTP/3 (QUIC Dual-Protocol Port)** | `6968.6 req/s` | `6.9ms` | `3.0ms` | `15.0ms` | `100.0%` |

---

## 💡 Protocol Observations

1. **HTTP/3 (QUIC)**: Runs over UDP, eliminating head-of-line blocking at the transport layer. It exhibits high throughput under load due to single-socket multiplexing over UDP.
2. **HTTP/2**: Uses TCP multiplexing. It performs significantly better than HTTP/1.1 under concurrency by running requests in parallel streams over a single TCP connection.
3. **HTTP/1.1 (HTTPS)**: Suffers from TCP head-of-line blocking. Under high concurrency, it is constrained by connection overhead and handshakes.
