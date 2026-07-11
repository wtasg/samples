process.env.NODE_ENV = 'test';

const { http2Server, httpsServer } = require('./server');
const http2 = require('node:http2');
const https = require('https');
const assert = require('assert');

// Global HTTPS Agent to ignore self-signed certificate warnings in tests for HTTP/1.1
const agent = new https.Agent({ rejectUnauthorized: false });

let testHttp2Port = 0;
let testHttpsPort = 0;

// Helper to make standard HTTPS (HTTP/1.1) requests
function httpsRequest(port, path) {
  return new Promise((resolve, reject) => {
    https.get(`https://localhost:${port}${path}`, { agent }, (res) => {
      let data = '';
      res.on('data', (chunk) => data += chunk);
      res.on('end', () => {
        resolve({
          statusCode: res.statusCode,
          headers: res.headers,
          body: data
        });
      });
    }).on('error', reject);
  });
}

// Helper to make HTTP/2 requests using native client
function http2Request(port, path) {
  return new Promise((resolve, reject) => {
    const client = http2.connect(`https://localhost:${port}`, {
      rejectUnauthorized: false
    });

    client.on('error', (err) => {
      reject(err);
    });

    const req = client.request({ ':path': path });
    let data = '';
    let status = 0;
    let responseHeaders = {};

    req.on('response', (headers) => {
      status = parseInt(headers[':status'], 10);
      responseHeaders = headers;
    });

    req.on('data', (chunk) => {
      data += chunk;
    });

    req.on('end', () => {
      client.close();
      resolve({
        statusCode: status,
        headers: responseHeaders,
        body: data
      });
    });

    req.on('error', (err) => {
      client.close();
      reject(err);
    });
  });
}

async function runTests() {
  console.log('--- Starting HTTP/2 & HTTPS Integration Tests ---');

  // Start HTTP/2 server on random free port
  await new Promise((resolve) => {
    http2Server.listen(0, () => {
      testHttp2Port = http2Server.address().port;
      console.log(`Test HTTP/2 Server running at https://localhost:${testHttp2Port}`);
      resolve();
    });
  });

  // Start HTTPS server on random free port
  await new Promise((resolve) => {
    httpsServer.listen(0, () => {
      testHttpsPort = httpsServer.address().port;
      console.log(`Test HTTPS Server running at https://localhost:${testHttpsPort}`);
      resolve();
    });
  });

  try {
    // ==========================================
    // PHASE 1: TESTING HTTP/2 PROTOCOL SERVER
    // ==========================================
    console.log('\n--- PHASE 1: Testing HTTP/2 (Port ' + testHttp2Port + ') ---');
    
    console.log('Test 1.1: GET /heartbeat over HTTP/2');
    const h2Heartbeat = await http2Request(testHttp2Port, '/heartbeat');
    assert.strictEqual(h2Heartbeat.statusCode, 200);
    const h2HeartbeatData = JSON.parse(h2Heartbeat.body);
    assert.strictEqual(h2HeartbeatData.status, 'alive');
    console.log('✓ Heartbeat test passed on HTTP/2');

    console.log('Test 1.2: GET /load over HTTP/2');
    const h2Load = await http2Request(testHttp2Port, '/load');
    assert.strictEqual(h2Load.statusCode, 200);
    const h2LoadData = JSON.parse(h2Load.body);
    assert.ok(typeof h2LoadData.cpu === 'number');
    console.log(`✓ CPU query test passed on HTTP/2. CPU: ${h2LoadData.cpu}%`);

    console.log('Test 1.3: Rate Limiting on HTTP/2');
    let h2RateLimitHit = false;
    for (let i = 0; i < 20; i++) {
      const res = await http2Request(testHttp2Port, '/load');
      if (res.statusCode === 429) {
        h2RateLimitHit = true;
        const errData = JSON.parse(res.body);
        assert.strictEqual(errData.error, 'Too many requests');
        console.log(`✓ HTTP/2 Rate limit hit as expected on request #${i + 1}. Status 429.`);
        break;
      }
    }
    assert.ok(h2RateLimitHit, 'Should have triggered HTTP/2 rate limiter');

    console.log('Test 1.4: SSE Stream over HTTP/2');
    const h2StreamPromise = new Promise((resolve, reject) => {
      const client = http2.connect(`https://localhost:${testHttp2Port}`, {
        rejectUnauthorized: false
      });
      const req = client.request({ ':path': '/stream' });
      req.on('response', (headers) => {
        assert.strictEqual(headers[':status'], 200);
        assert.strictEqual(headers['content-type'], 'text/event-stream');
      });
      req.on('data', (chunk) => {
        const chunkStr = chunk.toString();
        if (chunkStr.trim().startsWith(':')) return; // ignore comments
        if (chunkStr.includes('data:')) {
          const jsonStr = chunkStr.substring(chunkStr.indexOf('data:') + 5).trim();
          try {
            const data = JSON.parse(jsonStr);
            assert.ok(typeof data.cpu === 'number');
            req.destroy();
            client.close();
            resolve();
          } catch (e) {
            reject(new Error(`H2 SSE Parse failure: ${e.message}`));
          }
        }
      });
      req.on('error', (err) => {
        if (req.destroyed) return;
        reject(err);
      });
    });
    await h2StreamPromise;
    console.log('✓ SSE Stream passed on HTTP/2');

    // ==========================================
    // PHASE 2: TESTING STANDARD HTTPS SERVER
    // ==========================================
    console.log('\n--- PHASE 2: Testing HTTPS (Port ' + testHttpsPort + ') ---');

    console.log('Test 2.1: GET /heartbeat over HTTPS');
    const httpsHeartbeat = await httpsRequest(testHttpsPort, '/heartbeat');
    assert.strictEqual(httpsHeartbeat.statusCode, 200);
    const httpsHeartbeatData = JSON.parse(httpsHeartbeat.body);
    assert.strictEqual(httpsHeartbeatData.status, 'alive');
    console.log('✓ Heartbeat test passed on HTTPS');

    console.log('Test 2.2: GET /load over HTTPS');
    const httpsLoad = await httpsRequest(testHttpsPort, '/load');
    assert.strictEqual(httpsLoad.statusCode, 200);
    const httpsLoadData = JSON.parse(httpsLoad.body);
    assert.ok(typeof httpsLoadData.cpu === 'number');
    console.log(`✓ CPU query test passed on HTTPS. CPU: ${httpsLoadData.cpu}%`);

    console.log('Test 2.3: Rate Limiting on HTTPS');
    let httpsRateLimitHit = false;
    for (let i = 0; i < 20; i++) {
      const res = await httpsRequest(testHttpsPort, '/load');
      if (res.statusCode === 429) {
        httpsRateLimitHit = true;
        const errData = JSON.parse(res.body);
        assert.strictEqual(errData.error, 'Too many requests');
        console.log(`✓ HTTPS Rate limit hit as expected on request #${i + 1}. Status 429.`);
        break;
      }
    }
    assert.ok(httpsRateLimitHit, 'Should have triggered HTTPS rate limiter');

    console.log('Test 2.4: SSE Stream over HTTPS');
    const httpsStreamPromise = new Promise((resolve, reject) => {
      const req = https.get(`https://localhost:${testHttpsPort}/stream`, { agent }, (res) => {
        assert.strictEqual(res.statusCode, 200);
        assert.strictEqual(res.headers['content-type'], 'text/event-stream');

        res.on('data', (chunk) => {
          const chunkStr = chunk.toString();
          if (chunkStr.trim().startsWith(':')) return;
          if (chunkStr.includes('data:')) {
            const jsonStr = chunkStr.substring(chunkStr.indexOf('data:') + 5).trim();
            try {
              const data = JSON.parse(jsonStr);
              assert.ok(typeof data.cpu === 'number');
              req.destroy();
              resolve();
            } catch (e) {
              reject(new Error(`HTTPS SSE Parse failure: ${e.message}`));
            }
          }
        });
      });
      req.on('error', (err) => {
        if (req.destroyed) return;
        reject(err);
      });
    });
    await httpsStreamPromise;
    console.log('✓ SSE Stream passed on HTTPS');

    console.log('\n=== All Dual-Protocol Tests Passed Successfully! ===');
  } catch (error) {
    console.error('\n❌ Integration Test Failure:', error);
    process.exitCode = 1;
  } finally {
    console.log('\nShutting down HTTP/2 and HTTPS test servers...');
    await new Promise((resolve) => http2Server.close(resolve));
    await new Promise((resolve) => httpsServer.close(resolve));
    console.log('Test servers closed.');
    process.exit();
  }
}

runTests();
