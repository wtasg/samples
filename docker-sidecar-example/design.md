# Docker Sidecar Networking Design

This document explains the networking design of the Docker Sidecar Observability Example and addresses why Prometheus accesses the application via the local loopback address `127.0.0.1:8080/metrics`.

## The Shared Network Namespace Pattern

In standard Docker Compose deployments, each service container gets its own isolated network interface (typically on a bridge network, e.g., `172.18.0.x`).

However, this project implements a **Sidecar Pattern** using a shared network namespace. In [docker-compose.yml](./docker-compose.yml), the `proxy-sidecar` and `prometheus-sidecar` services are configured with:

```yaml
network_mode: "service:app"
```

### Why does Prometheus scrape `127.0.0.1:8080/metrics`?

1. **Shared Loopback Interface**:
   By sharing the network namespace of the parent `app` container, the Prometheus container inherits the exact same network interfaces. The loopback address `127.0.0.1` inside the Prometheus container is identical to the loopback address inside the Go application container.

2. **No DNS Resolution Required**:
   Because they are in the same network stack, they behave as if they are processes running on the same host. The Go application binds to `127.0.0.1:8080`. Prometheus can scrape `/metrics` directly at `127.0.0.1:8080` without resolving the service name `app`.

3. **Secure Scraping**:
   By binding the Go application's metrics server solely to `127.0.0.1:8080`, the metrics endpoint is kept private to the shared network namespace. It is not exposed to the outer Docker network or the host machine directly. Public users accessing the application via the Nginx proxy (which maps host port `60015` to container port `80`) cannot hit the metrics port directly.

## Network Port Layout

```text
                        +---------------------------------------------+
                        |  App Container Network Namespace            |
                        |                                             |
  Host Port: 60015 ---->|  [Port 80] Nginx Proxy Sidecar              |
                        |          |                                  |
                        |          | proxy_pass                       |
                        |          v                                  |
                        |  [Port 8080] Go Application (127.0.0.1)     |
                        |          ^                                  |
                        |          | localhost scrape                 |
                        |          |                                  |
  Host Port: 60016 ---->|  [Port 9090] Prometheus Agent Sidecar       |
                        |                                             |
                        +---------------------------------------------+
```
