# DocDB Studio — Web UI Client

An interactive web-based database client and dashboard for **DocDB**. Written in pure Go,
communicating with the DocDB gRPC server using the custom `github.com/docdb/client/docdb`
client library, and serving a premium web interface.

```
╔══════════════════════════════════════════════╗
║            DocDB Studio Web UI Client        ║
╚══════════════════════════════════════════════╝
```

## Features

- **NoSQL Command Editor**: Execute any supported javascript-style command (insert, find, update, delete).
- **Responsive Views (JSON & Table)**: Dual view modes to display raw pretty-printed JSON documents or a dynamic flat schema-less grid table.
- **Active Collection Browser**: Displays list of collections, document count, and database size.
- **Quick Templates**: One-click actions to write createCollection, insert, and query templates.
- **Performance Metrics**: Shows query execution time on the server in milliseconds.
- **Interactive Health Indicator**: Live connection ping tracking connection status to the database.

## Port Assignments

- **DocDB Daemon Server (gRPC)**: `60013` (mapped to container port `60013`)
- **DocDB Studio (Web UI Client)**: `60014` (mapped to container port `8080`)

## Running Locally

### 1. Start the database server
Ensure the DocDB gRPC server is listening on port `60013`:
```bash
cd docdb-example
go run ./cmd/server --addr :60013 --data ./data
```

### 2. Start the Client Web App
Specify the database address (`DOCDB_ADDR`) and Web UI port:
```bash
cd docdb-client-example
DOCDB_ADDR=http://localhost:60013 PORT=8080 go run ./cmd/client
```
Open [http://localhost:8080](http://localhost:8080) in your browser.

## Running with Docker Compose (Recommended)

DocDB Studio is fully dockerized and integrated into the repository's root `docker-compose.yml`.

### 1. Setup environment ports
Configure the ports in `.env` at the root of the repository:
```env
DOCDB_SERVER_PORT=60013
DOCDB_CLIENT_PORT=60014
```

### 2. Launch the services
```bash
docker compose up --build -d docdb-client docdb-server
```

### 3. Open the UI
Go to [http://localhost:60014](http://localhost:60014).

Data is persisted inside a Docker volume named `docdb-data`.

## Project Structure

```
docdb-client-example/
├── go.mod
├── README.md
├── Dockerfile           Multi-stage docker setup copying local dependencies
├── static/              Web UI static assets
│   ├── index.html       App layout & template triggers
│   ├── styles.css       Custom glassmorphism dark-mode stylesheet
│   └── app.js           AJAX request handler & DOM renderer
└── cmd/client/
    └── main.go          HTTP backend web server talking to DocDB gRPC
```
