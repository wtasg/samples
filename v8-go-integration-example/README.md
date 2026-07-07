# V8 + Go Integration Example

This project demonstrates a decoupled architecture integrating the V8 JavaScript engine with a Go backend. It features a modern web client interface (V8 + Go Integration Playground) that runs Go-sandboxed Javascript with bidirectional callbacks.

## Features

- **Decoupled Architecture**: Clean interface definitions in the `engine` package allow swapping execution backends easily.
- **V8 Sandboxing**: Executes arbitrary JavaScript inside separate sandboxed V8 isolates and contexts (`rogchap.com/v8go`).
- **Bidirectional Callbacks**:
  - `console.log` / `warn` / `error`: Overridden JavaScript console functions that stream logs back to Go.
  - `goCompute(x, y)`: Exposes a synchronous Go function to JavaScript for mathematical operations.
  - `goFetch(url)`: Simulates a secure HTTP fetch initiated from inside the V8 sandbox but executed on the Go backend.
- **Premium Client Playground Design**:
  - **Full-Width Main Layout**: The columns span 100% viewport width, starting with a 50/50 split layout.
  - **Draggable Splitter Bar**: Responsive resizing of the editor and results panels via a central bar using mouse or touch drag gestures.
  - **Tabbed Scratchpad System**: Support for multiple tabs where users can code from scratch in new tabs (e.g. "Untitled 1") while templates remain available. Edits are cached dynamically when switching tabs to prevent data loss.
  - **Offline Localized Assets**: 100% offline functionality. Google Fonts are downloaded and served directly from disk instead of fetching from external CDNs.
- **Comprehensive Test Coverage**:
  - Go unit tests for both the V8 engine runner and HTTP server handler logic.
  - End-to-end web functional tests using **Playwright (NodeJS + TypeScript)**.

## Directory Structure

```text
v8-go-integration-example/
├── src/
│   ├── client/       # Web UI playground client (HTML, CSS, JS, fonts)
│   ├── engine/       # Definition of the ScriptRunner interfaces & payloads
│   ├── server/       # HTTP Server and endpoints (APIServer)
│   └── v8/           # Implementation of the V8 JavaScript runner
├── tests/            # Playwright E2E functional test suite
├── go.mod            # Go module definitions
├── go.sum            # Go dependencies checksums
├── package.json      # NodeJS dependencies (Playwright, TypeScript)
├── playwright.config.ts # Playwright test runner configuration
└── README.md         # Subproject documentation
```

## Running the Application

### 1. Start the Server

From the `v8-go-integration-example` directory, start the Go server:

```bash
go run src/server/main.go
```

The server will start on port `60001` and serve the playground client. Visit [http://localhost:60001](http://localhost:60001) in your browser.

### 2. Run Go Tests

You can execute the unit and integration tests across the Go codebase:

```bash
go test -v ./...
```

### 3. Run Playwright E2E Tests

The functional web tests are built using Playwright and TypeScript.

To install dependencies and run the tests:

```bash
# Install NodeJS dependencies
npm install

# Run the Playwright functional tests
npx playwright test
```

> **Note**: Playwright is configured to automatically launch the Go server backend during the test execution and shut it down after tests complete. If the server is already running locally on port `60001`, Playwright will automatically reuse it.
