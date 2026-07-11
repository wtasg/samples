// Integration tests for wasm-browser-example
// Tests the HTTP server — WASM logic is tested via cargo test
process.env.NODE_ENV = 'test';

const { app, PORT } = require('./server');
const http = require('http');
const assert = require('assert');
const path = require('path');
const fs = require('fs');

let server;
let testPort;

function request(path) {
  return new Promise((resolve, reject) => {
    http.get(`http://localhost:${testPort}${path}`, (res) => {
      let data = '';
      res.on('data', c => data += c);
      res.on('end', () => resolve({ statusCode: res.statusCode, headers: res.headers, body: data }));
    }).on('error', reject);
  });
}

async function runTests() {
  console.log('--- Starting WASM Server Integration Tests ---');

  server = app.listen(0, () => {
    testPort = server.address().port;
    console.log(`Test server at http://localhost:${testPort}`);
  });

  await new Promise(r => setTimeout(r, 200));

  let passed = 0, failed = 0;

  async function test(name, fn) {
    try { await fn(); console.log(`  ✓ ${name}`); passed++; }
    catch (err) { console.error(`  ✗ ${name}: ${err.message}`); failed++; }
  }

  // ── Server endpoints ──────────────────────────────────────
  console.log('\n[HTTP Server]');

  await test('GET / returns 200 with HTML', async () => {
    const res = await request('/');
    assert.strictEqual(res.statusCode, 200);
    assert.ok(res.headers['content-type'].includes('text/html'));
    assert.ok(res.body.includes('WebAssembly Explorer'));
  });

  await test('GET /heartbeat returns status:alive', async () => {
    const res = await request('/heartbeat');
    assert.strictEqual(res.statusCode, 200);
    const data = JSON.parse(res.body);
    assert.strictEqual(data.status, 'alive');
    assert.ok(typeof data.uptime === 'number');
    assert.ok(data.timestamp);
  });

  await test('GET /styles.css returns CSS', async () => {
    const res = await request('/styles.css');
    assert.strictEqual(res.statusCode, 200);
    assert.ok(res.headers['content-type'].includes('css'));
  });

  await test('GET /app.js returns JavaScript', async () => {
    const res = await request('/app.js');
    assert.strictEqual(res.statusCode, 200);
    assert.ok(res.headers['content-type'].includes('javascript') || res.headers['content-type'].includes('text/'));
  });

  await test('GET /pkg/wasm_monitor_bg.wasm returns application/wasm', async () => {
    const res = await request('/pkg/wasm_monitor_bg.wasm');
    assert.strictEqual(res.statusCode, 200, `Expected 200, got ${res.statusCode} — did you run npm run build first?`);
    assert.strictEqual(res.headers['content-type'], 'application/wasm');
  });

  await test('GET /pkg/wasm_monitor.js returns JS module', async () => {
    const res = await request('/pkg/wasm_monitor.js');
    assert.strictEqual(res.statusCode, 200);
    assert.ok(res.body.includes('export'));
  });

  await test('GET /nonexistent returns 404', async () => {
    const res = await request('/nonexistent-file.xyz');
    assert.strictEqual(res.statusCode, 404);
  });

  // ── WASM build artifacts ──────────────────────────────────
  console.log('\n[WASM Build Artifacts]');

  const pkgDir = path.join(__dirname, 'public', 'pkg');
  await test('public/pkg/ directory exists', async () => {
    assert.ok(fs.existsSync(pkgDir), 'Run "npm run build" first');
  });

  await test('wasm_monitor_bg.wasm file exists and is non-empty', async () => {
    const wasmFile = path.join(pkgDir, 'wasm_monitor_bg.wasm');
    assert.ok(fs.existsSync(wasmFile), 'WASM binary not found');
    const stat = fs.statSync(wasmFile);
    assert.ok(stat.size > 1000, `WASM file too small: ${stat.size} bytes`);
  });

  await test('wasm_monitor.js JS bindings file exists', async () => {
    const jsFile = path.join(pkgDir, 'wasm_monitor.js');
    assert.ok(fs.existsSync(jsFile), 'JS bindings not found');
    const content = fs.readFileSync(jsFile, 'utf8');
    assert.ok(content.includes('fibonacci'), 'fibonacci export not found');
    assert.ok(content.includes('mandelbrot'), 'mandelbrot export not found');
    assert.ok(content.includes('fnv1a_hash'), 'fnv1a_hash export not found');
    assert.ok(content.includes('count_primes'), 'count_primes export not found');
  });

  console.log(`\n${'='.repeat(45)}`);
  if (failed === 0) {
    console.log(`✅ All ${passed} tests passed.`);
  } else {
    console.log(`❌ ${failed} test(s) failed, ${passed} passed.`);
    process.exitCode = 1;
  }

  server.close(() => {
    console.log('Test server closed.');
    process.exit();
  });
}

runTests().catch(err => {
  console.error('Unexpected error:', err);
  process.exitCode = 1;
  process.exit();
});
