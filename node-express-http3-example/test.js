process.env.NODE_ENV = 'test';

const { http3Server, httpsServer } = require('./server');
const { connect } = require('@currentspace/http3');
const https = require('https');
const assert = require('assert');

// Global HTTPS Agent to ignore self-signed certificate warnings in HTTP/1.1 tests
const agent = new https.Agent({ rejectUnauthorized: false });

let testHttp3Port = 0;
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

// Helper to make HTTP/3 requests using native @currentspace/http3 client
function http3Request(port, path) {
  return new Promise((resolve, reject) => {
    const authority = `localhost:${port}`;
    const session = connect(authority, {
      rejectUnauthorized: false
    });

    session.on('error', (err) => {
      reject(err);
    });

    session.on('connect', () => {
      const stream = session.request(
        {
          ':method': 'GET',
          ':path': path,
          ':authority': 'localhost',
          ':scheme': 'https',
        },
        { endStream: true }
      );

      let status = 0;
      let responseHeaders = {};
      const chunks = [];

      stream.on('response', (headers) => {
        status = parseInt(headers[':status'], 10);
        responseHeaders = headers;
      });

      stream.on('data', (chunk) => {
        chunks.push(Buffer.from(chunk));
      });

      stream.on('end', async () => {
        await session.close();
        resolve({
          statusCode: status,
          headers: responseHeaders,
          body: Buffer.concat(chunks).toString()
        });
      });

      stream.on('error', async (err) => {
        await session.close();
        reject(err);
      });
    });
  });
}

async function runTests() {
  console.log('--- Starting HTTP/3 & HTTPS Integration Tests ---');

  // Start HTTP/3 server on random free port
  http3Server.listen(0, '127.0.0.1');
  const h3Addr = http3Server.address();
  testHttp3Port = h3Addr.port;
  console.log(`Test HTTP/3 Server running at https://localhost:${testHttp3Port}`);

  // Start HTTPS server on random free port
  await new Promise((resolve) => {
    httpsServer.listen(0, '127.0.0.1', () => {
      testHttpsPort = httpsServer.address().port;
      console.log(`Test HTTPS Server running at https://localhost:${testHttpsPort}`);
      resolve();
    });
  });

  try {
    // ==========================================
    // PHASE 1: TESTING HTTP/3 PROTOCOL SERVER
    // ==========================================
    console.log('\n--- PHASE 1: Testing HTTP/3 (Port ' + testHttp3Port + ') ---');
    
    console.log('Test 1.1: GET /heartbeat over HTTP/3');
    const h3Heartbeat = await http3Request(testHttp3Port, '/heartbeat');
    assert.strictEqual(h3Heartbeat.statusCode, 200);
    const h3HeartbeatData = JSON.parse(h3Heartbeat.body);
    assert.strictEqual(h3HeartbeatData.status, 'alive');
    console.log('✓ Heartbeat test passed on HTTP/3');

    console.log('Test 1.2: GET /load over HTTP/3');
    const h3Load = await http3Request(testHttp3Port, '/load');
    assert.strictEqual(h3Load.statusCode, 200);
    const h3LoadData = JSON.parse(h3Load.body);
    assert.ok(typeof h3LoadData.cpu === 'number');
    console.log(`✓ CPU query test passed on HTTP/3. CPU: ${h3LoadData.cpu}%`);

    console.log('Test 1.3: Rate Limiting on HTTP/3');
    let h3RateLimitHit = false;
    for (let i = 0; i < 20; i++) {
      const res = await http3Request(testHttp3Port, '/load');
      if (res.statusCode === 429) {
        h3RateLimitHit = true;
        const errData = JSON.parse(res.body);
        assert.strictEqual(errData.error, 'Too many requests');
        console.log(`✓ HTTP/3 Rate limit hit as expected on request #${i + 1}. Status 429.`);
        break;
      }
    }
    assert.ok(h3RateLimitHit, 'Should have triggered HTTP/3 rate limiter');

    console.log('Test 1.4: SSE Stream over HTTP/3');
    const h3StreamPromise = new Promise((resolve, reject) => {
      const session = connect(`localhost:${testHttp3Port}`, {
        rejectUnauthorized: false
      });
      session.on('error', reject);
      session.on('connect', () => {
        const stream = session.request({
          ':method': 'GET',
          ':path': '/stream',
          ':authority': 'localhost',
          ':scheme': 'https',
        });
        stream.on('response', (headers) => {
          assert.strictEqual(parseInt(headers[':status'], 10), 200);
          assert.strictEqual(headers['content-type'], 'text/event-stream');
        });
        stream.on('data', (chunk) => {
          const chunkStr = chunk.toString();
          if (chunkStr.trim().startsWith(':')) return;
          if (chunkStr.includes('data:')) {
            const jsonStr = chunkStr.substring(chunkStr.indexOf('data:') + 5).trim();
            try {
              const data = JSON.parse(jsonStr);
              assert.ok(typeof data.cpu === 'number');
              stream.destroy();
              session.close().then(resolve);
            } catch (e) {
              reject(new Error(`H3 SSE Parse failure: ${e.message}`));
            }
          }
        });
        stream.on('error', (err) => {
          if (stream.destroyed) return;
          reject(err);
        });
      });
    });
    await h3StreamPromise;
    console.log('✓ SSE Stream passed on HTTP/3');

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
    console.log('\nShutting down HTTP/3 and HTTPS test servers...');
    await http3Server.close();
    await new Promise((resolve) => httpsServer.close(resolve));
    console.log('Test servers closed.');
    process.exit();
  }
}

runTests();
