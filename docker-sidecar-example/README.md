# Docker Sidecar Observability Example

This example demonstrates how to implement the **Sidecar Design Pattern** in a multi-container Docker environment. It provides a complete observability, monitoring, logging, and proxy stack wrapped around a Go application.

## Observability Architecture

This project sets up:
1. **Main Go Application**: Listens strictly on `127.0.0.1:8080`, instruments request counts/latencies, and writes JSON logs to `/var/log/app/access.log`.
2. **Nginx Reverse Proxy Sidecar**: Shares the network namespace of the Go application. Listens on port `80` (exposed to host port `60015`) and routes requests to the Go application locally.
3. **Prometheus Scraper Sidecar**: Shares the network namespace of the Go application. Scrapes the metrics from `127.0.0.1:8080/metrics` locally and exposes the Prometheus UI on port `9090` (exposed to host port `60016`).
4. **Fluent Bit Logging Sidecar**: Mounts a shared volume containing the application's log files. It reads, parses, enriches, and outputs the logs to standard output.
5. **Grafana visualization**: A standalone service pre-provisioned to connect to the Prometheus instance (reachable via `http://app:9090`) and visualize performance metrics in a dashboard on host port `60017`.

Refer to the [Architecture Document](arch.md) for full network namespace and volume diagrams.

---

## Exposed Ports Reference

- **`60015`**: Nginx Proxy (Go application HTTP Entrypoint)
- **`60016`**: Prometheus Scraper UI
- **`60017`**: Grafana Observability Dashboard

---

## Quick Start

### 1. Launch the Stack

Run the stack in detached mode:

```bash
docker compose up --build -d
```

Verify that all containers are running successfully:

```bash
docker compose ps
```

### 2. Generate Workload & Inspect Observability

Generate normal traffic, CPU spikes, and errors to populate the monitoring dashboards:

- **Greet (RPS counter)**:
  ```bash
  curl http://localhost:60015/
  ```
- **CPU Workload (Spikes CPU usage metrics)**:
  ```bash
  curl http://localhost:60015/compute
  ```
- **Internal Server Error (Triggers error rate metrics)**:
  ```bash
  curl http://localhost:60015/error
  ```

### 3. Open Dashboards

- **Grafana (Dashboards & Visualization)**: Open [http://localhost:60017](http://localhost:60017) in your browser.
  - **Credentials**: Username `admin`, Password `admin`.
  - Go to **Dashboards** to view the pre-configured *App Observability Dashboard*.
- **Prometheus UI**: Open [http://localhost:60016](http://localhost:60016) in your browser to write Prometheus queries (e.g. `http_requests_total`).

### 4. Inspect Logging Sidecar Streams

Check the logs of the `logging-sidecar` container to see how Fluent Bit tailed, parsed, and enriched the Go application's access logs with sidecar metadata fields:

```bash
docker logs logging-sidecar
```

Expected log output format:
```text
[0] app.access: [1721025752.000000000] {"timestamp"=>"2026-07-15T06:42:32Z", "method"=>"GET", "path"=>"/", "status"=>200, "duration_ms"=>"0.082", "ip"=>"127.0.0.1:45302", "user_agent"=>"curl/7.81.0", "sidecar_name"=>"fluent-bit", "observability_system"=>"sidecar_logging"}
```

---

## Verification & Clean Up

An automated integration script is included to boot, verify endpoints, scrape metrics, inspect logs, and teardown the stack.

Run the test suite:
```bash
./test.sh
```

To stop and clean up volumes manually:
```bash
docker compose down -v
```
