# WebAssembly Browser Explorer

A web application demonstrating **Rust compiled to WebAssembly**, running native-speed computation directly in the browser — the browser-side counterpart to the `v8-go-integration-example` (which embeds a JS runtime in Go):

- Rust library crate compiled to WASM via `wasm-pack` + `wasm-bindgen`.
- Exposed WASM functions: `fibonacci(n)`, `mandelbrot(w, h, iters)`, `fnv1a_hash(s)`, `count_primes(n)`.
- Minimal Node.js + Express static file server with correct `application/wasm` MIME type.
- Premium dark-mode dashboard with four interactive panels:
  - **Mandelbrot Renderer** — Renders the full Mandelbrot set on an HTML Canvas using pure WASM pixel computation. Configurable iteration count.
  - **Fibonacci Race** — Benchmarks `fibonacci(n)` called 10,000 times in both JavaScript and WASM. Visualizes relative performance with animated bars.
  - **Prime Sieve Benchmark** — Benchmarks the Sieve of Eratosthenes (count primes up to N) in JS vs WASM, showing real speedup numbers.
  - **FNV-1a Hash Terminal** — Compute the 64-bit FNV-1a hash of arbitrary text in Rust, displayed live.
- Integration tests covering HTTP server endpoints and WASM artifact validity.
- Rust unit tests for all WASM functions via `cargo test`.

## Prerequisites

- **Node.js** ≥ 20
- **Rust** (stable) with the `wasm32-unknown-unknown` target
- **wasm-pack**

## Setup

1. Install Rust and the WASM target (if not already installed):

   ```bash
   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
   rustup target add wasm32-unknown-unknown
   ```

2. Install `wasm-pack`:

   ```bash
   cargo install wasm-pack
   ```

3. Install Node.js dependencies, build the WASM module, and start the server:

   ```bash
   npm install
   npm start
   ```

   The `prestart` hook automatically runs `wasm-pack build` and copies the output to `public/pkg/`.

4. Open the dashboard in your browser:

   ```text
   http://localhost:60010
   ```

## Running Tests

```bash
npm test
```

Tests cover:

- HTTP server static file serving (`/`, `/styles.css`, `/app.js`)
- Correct `application/wasm` MIME type for `.wasm` files
- `/heartbeat` REST endpoint
- Presence of `public/pkg/wasm_monitor_bg.wasm` (non-empty)
- Presence of `public/pkg/wasm_monitor.js` with all four exported functions

### Rust Unit Tests

```bash
cd wasm && cargo test
```

## Port

| Port    | Protocol | Notes               |
| --------- | ---------- | --------------------- |
| `60010` | HTTP     | Static file server  |

## Docker

```bash
docker build -t wasm-browser-example .
docker run -p 60010:60010 wasm-browser-example
```

The Dockerfile uses a multi-stage build: a builder stage installs Rust and wasm-pack to compile the WASM module; the runtime stage is a slim Node.js image.

## Architecture

```text
Browser (ES Module)
  ↓  import init, { fibonacci, mandelbrot, fnv1a_hash, count_primes }
     from './pkg/wasm_monitor.js'
  ↓  init('./pkg/wasm_monitor_bg.wasm')
  ↓  wasm functions run at native speed in the WASM sandbox

Node.js + Express (port 60010)
  └── Serves public/ (index.html, app.js, styles.css, pkg/*.wasm)
```

## Rust WASM Functions

| Function | Signature | Description |
| ---------- | ----------- | ------------- |
| `fibonacci` | `(n: u32) → u64` | Iterative Fibonacci, overflow-safe |
| `mandelbrot` | `(w: u32, h: u32, max_iter: u32) → Vec<u8>` | RGBA pixel buffer of the Mandelbrot set |
| `fnv1a_hash` | `(s: &str) → String` | 64-bit FNV-1a hash as decimal string |
| `count_primes` | `(n: u32) → u32` | Count primes ≤ n via Sieve of Eratosthenes |

## V8-in-Go vs WASM-in-Browser

| | `v8-go-integration-example` | `wasm-browser-example` |
| -- | -- | -- |
| **Concept** | Embed a JS runtime inside Go | Embed compiled Rust inside the browser |
| **Runtime** | V8 JavaScript engine | WebAssembly VM |
| **Language boundary** | Go ↔ JavaScript | JavaScript ↔ Rust |
| **Execution** | Server-side | Client-side (browser) |
| **Sandbox** | V8 isolate | WASM linear memory |
