# V8 + Go Integration Example

This project demonstrates a decoupled architecture integrating the V8 JavaScript engine with a Go backend. It features a modern web client interface (V8 + Go Integration Playground) that runs Go-sandboxed Javascript with bidirectional callbacks.

## Features

- **Decoupled Architecture**: Clean interface definitions in the engine package allow swapping execution backends easily.
- **V8 Sandboxing**: Executes arbitrary JavaScript inside separate sandboxed V8 isolates and contexts (`rogchap.com/v8go`).
- **Bidirectional Callbacks**:
  - `console.log` / `warn` / `error`: Overridden JavaScript console functions that stream logs back to Go.
  - `goCompute(x, y)`: Exposes a synchronous Go function to JavaScript for mathematical operations.
  - `goFetch(url)`: Simulates a secure HTTP fetch initiated from inside the V8 sandbox but executed on the Go backend.
- **Comprehensive Test Coverage**: Unit tests for both the V8 engine runner and HTTP server handler logic, along with full end-to-end integration tests.

## Directory Structure

```text
v8-go-integration-example/
├── src/
│   ├── client/       # Web UI playground client (HTML, CSS, JS)
│   ├── engine/       # Definition of the ScriptRunner interfaces & payloads
│   ├── server/       # HTTP Server and endpoints (APIServer)
│   └── v8/           # Implementation of the V8 JavaScript runner
├── go.mod            # Go module definitions
├── go.sum            # Go dependencies checksums
└── README.md         # Project documentation
```

## Running the Application

### 1. Start the Server

From the `v8-go-integration-example` directory, start the Go server:

```bash
go run src/server/main.go
```

The server will start on port `60001` and serve the playground client. Visit [http://localhost:60001](http://localhost:60001) in your browser.

### 2. Run Tests

You can execute the unit and integration tests across the codebase:

```bash
go test -v ./...
```

To run tests specifically for the server component:

```bash
go test -v ./src/server/...
```

And for the V8 engine runner:

```bash
go test -v ./src/v8/...
```
