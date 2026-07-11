// Integration tests for node-express-websocket-example
// Uses Node.js built-in assert + ws library — no external test runner needed
process.env.NODE_ENV = 'test';

const { server, startWebSocketServer } = require('./server');
const https = require('https');
const assert = require('assert');
const { WebSocket } = require('ws');

// Ignore self-signed cert in tests
const agent = new https.Agent({ rejectUnauthorized: false });

let testPort = 0;
let baseUrl = '';
let baseWsUrl = '';

function restRequest(path) {
  return new Promise((resolve, reject) => {
    https.get(`${baseUrl}${path}`, { agent }, (res) => {
      let data = '';
      res.on('data', (c) => (data += c));
      res.on('end', () => resolve({ statusCode: res.statusCode, headers: res.headers, body: data }));
    }).on('error', reject);
  });
}

function openWs() {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(`${baseWsUrl}/ws`, {
      rejectUnauthorized: false
    });
    ws.once('open', () => resolve(ws));
    ws.once('error', reject);
  });
}

function nextMessage(ws) {
  return new Promise((resolve, reject) => {
    ws.once('message', (data) => {
      try { resolve(JSON.parse(data.toString())); }
      catch (e) { reject(e); }
    });
    ws.once('error', reject);
  });
}

async function runTests() {
  console.log('--- Starting WebSocket Integration Tests ---');

  await new Promise((resolve) => {
    server.listen(0, () => {
      testPort  = server.address().port;
      baseUrl   = `https://localhost:${testPort}`;
      baseWsUrl = `wss://localhost:${testPort}`;
      console.log(`Test server at ${baseUrl}`);
      resolve();
    });
  });

  let passed = 0;
  let failed = 0;

  async function test(name, fn) {
    try {
      await fn();
      console.log(`  ✓ ${name}`);
      passed++;
    } catch (err) {
      console.error(`  ✗ ${name}: ${err.message}`);
      failed++;
    }
  }

  // ────────────────────────────────────────────
  // REST Tests
  // ────────────────────────────────────────────
  console.log('\n[REST Endpoints]');

  await test('GET /heartbeat returns 200 with status:alive', async () => {
    const res = await restRequest('/heartbeat');
    assert.strictEqual(res.statusCode, 200);
    const data = JSON.parse(res.body);
    assert.strictEqual(data.status, 'alive');
    assert.ok(typeof data.uptime === 'number');
    assert.ok(typeof data.connections === 'number');
  });

  await test('GET /heartbeat includes timestamp', async () => {
    const res = await restRequest('/heartbeat');
    const data = JSON.parse(res.body);
    assert.ok(data.timestamp, 'Should include timestamp');
    assert.ok(!isNaN(new Date(data.timestamp).getTime()), 'Timestamp should be a valid ISO date');
  });

  // ────────────────────────────────────────────
  // WebSocket Connection Tests
  // ────────────────────────────────────────────
  console.log('\n[WebSocket Connection]');

  await test('WebSocket connects and receives welcome message', async () => {
    const ws = await openWs();
    const msg = await nextMessage(ws);
    assert.strictEqual(msg.type, 'welcome');
    assert.ok(typeof msg.clientId === 'string' && msg.clientId.length > 0);
    ws.close();
  });

  await test('Server sends cpu messages periodically', async () => {
    const ws = await openWs();
    // Skip welcome
    await nextMessage(ws);

    // Wait for a cpu message
    const msg = await new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timed out waiting for cpu message')), 3000);
      function onMsg(data) {
        const parsed = JSON.parse(data.toString());
        if (parsed.type === 'cpu') {
          clearTimeout(timeout);
          ws.off('message', onMsg);
          resolve(parsed);
        }
      }
      ws.on('message', onMsg);
    });

    assert.strictEqual(msg.type, 'cpu');
    assert.ok(typeof msg.cpu === 'number');
    assert.ok(msg.cpu >= 0 && msg.cpu <= 100);
    assert.ok(typeof msg.clients === 'number');
    ws.close();
  });

  // ────────────────────────────────────────────
  // Ping / Pong
  // ────────────────────────────────────────────
  console.log('\n[Ping / Pong]');

  await test('Sends ping and receives pong', async () => {
    const ws = await openWs();
    await nextMessage(ws); // consume welcome

    ws.send(JSON.stringify({ type: 'ping', payload: 'hello-test' }));

    const pong = await new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timed out waiting for pong')), 3000);
      function onMsg(data) {
        const msg = JSON.parse(data.toString());
        if (msg.type === 'pong') {
          clearTimeout(timeout);
          ws.off('message', onMsg);
          resolve(msg);
        }
      }
      ws.on('message', onMsg);
    });

    assert.strictEqual(pong.type, 'pong');
    assert.strictEqual(pong.echo, 'hello-test');
    assert.ok(pong.serverTime);
    ws.close();
  });

  // ────────────────────────────────────────────
  // Interval Control
  // ────────────────────────────────────────────
  console.log('\n[Interval Control]');

  await test('set-interval command receives interval-ack', async () => {
    const ws = await openWs();
    await nextMessage(ws); // consume welcome

    ws.send(JSON.stringify({ type: 'set-interval', intervalMs: 500 }));

    const ack = await new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timed out waiting for interval-ack')), 3000);
      function onMsg(data) {
        const msg = JSON.parse(data.toString());
        if (msg.type === 'interval-ack') {
          clearTimeout(timeout);
          ws.off('message', onMsg);
          resolve(msg);
        }
      }
      ws.on('message', onMsg);
    });

    assert.strictEqual(ack.type, 'interval-ack');
    assert.strictEqual(ack.intervalMs, 500);
    ws.close();
  });

  await test('set-interval clamps values below 250ms to 250ms', async () => {
    const ws = await openWs();
    await nextMessage(ws);

    ws.send(JSON.stringify({ type: 'set-interval', intervalMs: 10 }));

    const ack = await new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 3000);
      function onMsg(data) {
        const msg = JSON.parse(data.toString());
        if (msg.type === 'interval-ack') {
          clearTimeout(timeout); ws.off('message', onMsg); resolve(msg);
        }
      }
      ws.on('message', onMsg);
    });
    assert.strictEqual(ack.intervalMs, 250, 'Should clamp to 250ms minimum');
    ws.close();
  });

  // ────────────────────────────────────────────
  // Broadcast
  // ────────────────────────────────────────────
  console.log('\n[Broadcast]');

  await test('Broadcast message is received by all connected clients', async () => {
    const ws1 = await openWs();
    const ws2 = await openWs();
    await nextMessage(ws1); // welcome
    await nextMessage(ws2); // welcome

    ws1.send(JSON.stringify({ type: 'broadcast', text: 'hello-all' }));

    // Both ws1 and ws2 should receive the broadcast
    async function waitForBroadcast(ws) {
      return new Promise((resolve, reject) => {
        const timeout = setTimeout(() => reject(new Error('Timed out waiting for broadcast')), 3000);
        function onMsg(data) {
          const msg = JSON.parse(data.toString());
          if (msg.type === 'broadcast') {
            clearTimeout(timeout); ws.off('message', onMsg); resolve(msg);
          }
        }
        ws.on('message', onMsg);
      });
    }

    const [b1, b2] = await Promise.all([waitForBroadcast(ws1), waitForBroadcast(ws2)]);
    assert.strictEqual(b1.type, 'broadcast');
    assert.strictEqual(b1.text, 'hello-all');
    assert.strictEqual(b2.text, 'hello-all');
    ws1.close(); ws2.close();
  });

  // ────────────────────────────────────────────
  // Unknown message type
  // ────────────────────────────────────────────
  console.log('\n[Error Handling]');

  await test('Unknown message type receives error response', async () => {
    const ws = await openWs();
    await nextMessage(ws);

    ws.send(JSON.stringify({ type: 'unknown-type' }));

    const errMsg = await new Promise((resolve, reject) => {
      const timeout = setTimeout(() => reject(new Error('Timeout')), 3000);
      function onMsg(data) {
        const msg = JSON.parse(data.toString());
        if (msg.type === 'error') {
          clearTimeout(timeout); ws.off('message', onMsg); resolve(msg);
        }
      }
      ws.on('message', onMsg);
    });

    assert.strictEqual(errMsg.type, 'error');
    assert.ok(errMsg.message.includes('unknown-type'));
    ws.close();
  });

  // ────────────────────────────────────────────
  // Summary
  // ────────────────────────────────────────────
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

runTests().catch((err) => {
  console.error('Unexpected error:', err);
  process.exitCode = 1;
  process.exit();
});
