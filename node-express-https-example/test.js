// Set testing environment
process.env.NODE_ENV = 'test';

const { server } = require('./server');
const https = require('https');
const assert = require('assert');

// Global HTTPS Agent to ignore self-signed certificate warnings in tests
const agent = new https.Agent({ rejectUnauthorized: false });

let testPort = 0;
let baseUrl = '';

// Helper to make HTTPS requests
function request(path) {
  return new Promise((resolve, reject) => {
    https.get(`${baseUrl}${path}`, { agent }, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        resolve({
          statusCode: res.statusCode,
          headers: res.headers,
          body: data
        });
      });
    }).on('error', (err) => {
      reject(err);
    });
  });
}

async function runTests() {
  console.log('--- Starting Integration Tests ---');

  // Start server on a random free port for testing
  await new Promise((resolve) => {
    server.listen(0, () => {
      testPort = server.address().port;
      baseUrl = `https://localhost:${testPort}`;
      console.log(`Test server running at ${baseUrl}`);
      resolve();
    });
  });

  try {
    // Test 1: Heartbeat Endpoint
    console.log('\nTest 1: GET /heartbeat');
    const resHeartbeat = await request('/heartbeat');
    assert.strictEqual(resHeartbeat.statusCode, 200, 'Heartbeat should return 200');
    const heartbeatData = JSON.parse(resHeartbeat.body);
    assert.strictEqual(heartbeatData.status, 'alive', 'Heartbeat status should be alive');
    assert.ok(heartbeatData.timestamp, 'Heartbeat should include timestamp');
    assert.ok(typeof heartbeatData.uptime === 'number', 'Heartbeat should include uptime number');
    console.log('✓ Heartbeat test passed.');

    // Test 2: CPU Query Endpoint
    console.log('\nTest 2: GET /api/cpu/load');
    const resCpu = await request('/api/cpu/load');
    assert.strictEqual(resCpu.statusCode, 200, 'CPU query should return 200');
    const cpuData = JSON.parse(resCpu.body);
    assert.ok(typeof cpuData.cpu === 'number', 'CPU load should be a number');
    assert.ok(cpuData.cpu >= 0 && cpuData.cpu <= 100, 'CPU load should be between 0 and 100');
    assert.ok(cpuData.timestamp, 'CPU load response should include timestamp');
    console.log(`✓ CPU query test passed. CPU Load: ${cpuData.cpu}%`);

    // Test 3: Rate Limiting
    // The rate limiter is set to 15 requests in 15 seconds.
    // Let's send 16 requests and verify that the 16th one fails with 429.
    console.log('\nTest 3: Rate Limiting on /api/cpu/load (15 requests/15s limit)');
    let rateLimitHit = false;
    for (let i = 0; i < 20; i++) {
      const res = await request('/api/cpu/load');
      if (res.statusCode === 429) {
        rateLimitHit = true;
        const errData = JSON.parse(res.body);
        assert.strictEqual(errData.error, 'Too many requests', 'Error message structure should match');
        console.log(`✓ Rate limit hit as expected on request #${i + 1}. HTTP Status 429.`);
        break;
      }
    }
    assert.ok(rateLimitHit, 'Should have hit rate limit (HTTP 429) after sending multiple requests');

    // Test 4: SSE Streaming Endpoint
    console.log('\nTest 4: GET /api/cpu/stream (Server-Sent Events)');
    const streamPromise = new Promise((resolve, reject) => {
      const req = https.get(`${baseUrl}/api/cpu/stream`, { agent }, (res) => {
        assert.strictEqual(res.statusCode, 200, 'SSE stream should return 200');
        assert.strictEqual(res.headers['content-type'], 'text/event-stream', 'Content type should be text/event-stream');
        assert.strictEqual(res.headers['connection'], 'keep-alive', 'Connection should be keep-alive');

        let messageCount = 0;
        res.on('data', (chunk) => {
          const chunkStr = chunk.toString();
          // Filter out comments/heartbeats (lines starting with colon)
          if (chunkStr.trim().startsWith(':')) return;

          // Expect: data: {"cpu":..., "timestamp":...}
          const lines = chunkStr.split('\n');
          for (const line of lines) {
            if (line.startsWith('data:')) {
              const jsonStr = line.substring(5).trim();
              try {
                const data = JSON.parse(jsonStr);
                assert.ok(typeof data.cpu === 'number', 'SSE data should contain a CPU number');
                messageCount++;
                if (messageCount >= 1) {
                  req.destroy(); // Terminate the client request
                  resolve();
                }
              } catch (err) {
                reject(new Error(`Failed to parse SSE JSON message: ${err.message}`));
              }
            }
          }
        });
      });
      req.on('error', (err) => {
        // Since we call req.destroy(), an ECONNRESET/aborted error is expected and should not fail the test
        if (req.destroyed) return;
        reject(err);
      });
    });

    await streamPromise;
    console.log('✓ SSE streaming test passed.');

    console.log('\n=== All Tests Passed Successfully! ===');
  } catch (error) {
    console.error('\n❌ Test failure:', error);
    process.exitCode = 1;
  } finally {
    // Shut down server
    console.log('\nShutting down test server...');
    server.close(() => {
      console.log('Test server closed.');
      process.exit();
    });
  }
}

runTests();
