# ToyDB Studio — Web UI Client

An interactive web-based database client and dashboard for **ToyDB**. Written in pure Go, communicating with the ToyDB gRPC server using the custom `rdbms-client-lib` client library, and serving a premium web interface.

```
╔══════════════════════════════════════════════╗
║            ToyDB Studio Web UI Client        ║
╚══════════════════════════════════════════════╝
```

## Features

- **SQL Editor**: Execute any supported SQL statement (DDL/DML).
- **Responsive Data Grid**: Renders SELECT query results in a beautiful zebra-striped table.
- **Active Schema Browser**: Displays open tables, their columns, types, and primary key indicators.
- **Quick Templates**: One-click actions to write CREATE TABLE, INSERT, and query templates.
- **Performance Metrics**: Shows query execution time on the server in milliseconds.
- **Interactive Health Indicator**: Live connection ping tracking connection status to the database.

## Port Assignments

- **ToyDB Daemon Server (gRPC)**: `60011` (mapped to container port `9090`)
- **ToyDB Studio (Web UI Client)**: `60012` (mapped to container port `8080`)

## Running Locally

### 1. Start the database server
Ensure the ToyDB gRPC server is listening on port `9090` (or `60011` if mapped):
```bash
cd rdbms-example
go run ./cmd/server --addr :9090 --data ./data
```

### 2. Start the Client Web App
Specify the database address (`TOYDB_ADDR`) and Web UI port:
```bash
cd rdbms-client-example
TOYDB_ADDR=http://localhost:9090 PORT=8080 go run ./cmd/client
```
Open [http://localhost:8080](http://localhost:8080) in your browser.

## Running with Docker Compose (Recommended)

ToyDB Studio is fully dockerized and integrated into the repository's root `docker-compose.yml`.

### 1. Setup environment ports
Configure the ports in `.env` at the root of the repository:
```env
TOYDB_SERVER_PORT=60011
TOYDB_CLIENT_PORT=60012
```

### 2. Launch the services
```bash
docker compose up --build -d toydb-client toydb-server
```

### 3. Open the UI
Go to [http://localhost:60012](http://localhost:60012).

Data is persisted inside a Docker volume named `toydb-data`.

## Project Structure

```
rdbms-client-example/
├── go.mod
├── README.md
├── Dockerfile           Multi-stage docker setup copying local dependencies
├── static/              Web UI static assets
│   ├── index.html       App layout & template triggers
│   ├── styles.css       Custom glassmorphism dark-mode stylesheet
│   └── app.js           AJAX request handler & DOM renderer
└── cmd/client/
    └── main.go          HTTP backend web server talking to ToyDB gRPC
```
