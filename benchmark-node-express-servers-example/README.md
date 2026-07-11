# Dockerized Express Servers Benchmark Suite

This directory contains a standardized load testing framework to compare performance characteristics across different HTTP protocols:

- **HTTP/1.1 (HTTPS Only Port `60005`)**
- **HTTP/2 (H2 Multiplexed Port `60006`)**
- **HTTP/3 (QUIC Multiplexed Port `60007`)**
- **HTTPS fallbacks (Ports `60446`, `60447`)**

Testing is entirely Dockerized to ensure benchmarks are isolated, reproducible, and do not compete with host system resources.

---

## 🛠️ Usage

### 1. Build Server and Runner Images

Compile the Docker containers for the targets and the benchmark CLI:

```bash
docker compose build
```

### 2. Launch the Target Servers

Spin up the target servers in the background:

```bash
docker compose up -d https-server http2-server http3-server
```

### 3. Run the Benchmarks

Trigger the load tests against all protocols (e.g., 5,000 total requests at 100 concurrent clients):

```bash
docker compose run --rm benchmark-runner --target all --requests 5000 --concurrency 100
```

#### CLI Target Options

- `--target all`: Benchmarks all protocol variants (Default).

- `--target http1`: Benchmarks standard HTTPS (HTTP/1.1 on `60005`).
- `--target http2`: Benchmarks HTTP/2 (H2 on `60006`).
- `--target http3`: Benchmarks HTTP/3 (QUIC on `60007`).
- `--requests <count>`: Total request count (e.g. 10000).
- `--concurrency <count>`: Concurrent client count (e.g. 100).

---

## 📊 Outputs

The run generates a comparative Markdown report containing latency percentiles, throughput (RPS), and success rates in **`benchmark_report.md`**.
