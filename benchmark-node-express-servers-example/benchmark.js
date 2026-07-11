const { connect } = require('@currentspace/http3');
const https = require('https');
const http2 = require('http2');
const dns = require('dns').promises;
const fs = require('fs');
const path = require('path');
const minimist = require('minimist');

// Parse CLI Arguments
const argv = minimist(process.argv.slice(2), {
  string: ['target'],
  integer: ['requests', 'concurrency'],
  alias: { t: 'target', n: 'requests', c: 'concurrency' },
  default: { target: 'all', requests: 1000, concurrency: 50 }
});

const target = argv.target;
const requests = parseInt(argv.requests, 10);
const concurrency = parseInt(argv.concurrency, 10);

console.log('========================================================');
console.log('             Express Protocol Benchmarker               ');
console.log('========================================================');
console.log(`Target:      ${target}`);
console.log(`Requests:    ${requests}`);
console.log(`Concurrency: ${concurrency}`);
console.log('========================================================\n');

// Standard configurations for target endpoints in Docker network
const endpoints = {
  'http1': {
    name: 'HTTP/1.1 (HTTPS Only Project)',
    host: 'https-server',
    port: 60005,
    path: '/heartbeat'
  },
  'http2': {
    name: 'HTTP/2 (H2 Dual-Protocol Port)',
    host: 'http2-server',
    port: 60006,
    path: '/heartbeat'
  },
  'https-fallback': {
    name: 'HTTPS Fallback (H2 Dual-Protocol Port)',
    host: 'http2-server',
    port: 60446,
    path: '/heartbeat'
  },
  'http3': {
    name: 'HTTP/3 (QUIC Dual-Protocol Port)',
    host: 'http3-server',
    port: 60007,
    path: '/heartbeat'
  },
  'http3-fallback': {
    name: 'HTTPS Fallback (QUIC Dual-Protocol Port)',
    host: 'http3-server',
    port: 60447,
    path: '/heartbeat'
  }
};

// DNS Resolver Helper to map Docker container hostnames to IP addresses
function resolveHostToIp(hostname) {
  return new Promise((resolve) => {
    require('dns').lookup(hostname, (err, address) => {
      if (err) {
        console.warn(`DNS lookup failed for hostname "${hostname}": ${err.message}. Defaulting to hostname.`);
        resolve(hostname);
      } else {
        resolve(address);
      }
    });
  });
}

// 1. HTTP/1.1 HTTPS Benchmarker
function runHttp1Benchmark(requests, concurrency, ipAddress, port, reqPath) {
  return new Promise((resolve) => {
    const agent = new https.Agent({ rejectUnauthorized: false, keepAlive: true });
    const startTime = Date.now();
    let success = 0;
    let failed = 0;
    const latencies = [];
    let requestsSent = 0;
    let activeRequests = 0;

    const sendNext = () => {
      if (requestsSent >= requests) {
        if (activeRequests === 0) {
          const duration = (Date.now() - startTime) / 1000;
          const rps = parseFloat((success / duration).toFixed(2));
          latencies.sort((a, b) => a - b);
          const avgLatency = latencies.length ? latencies.reduce((a, b) => a + b, 0) / latencies.length : 0;
          const minLatency = latencies.length ? latencies[0] : 0;
          const maxLatency = latencies.length ? latencies[latencies.length - 1] : 0;
          resolve({
            rps,
            minLatency,
            maxLatency,
            avgLatency: parseFloat(avgLatency.toFixed(2)),
            success,
            failed
          });
        }
        return;
      }

      while (activeRequests < concurrency && requestsSent < requests) {
        requestsSent++;
        activeRequests++;
        const reqStartTime = Date.now();

        const req = https.get({
          hostname: ipAddress,
          port,
          path: reqPath,
          agent,
          rejectUnauthorized: false,
          headers: { 
            host: 'localhost',
            'x-bypass-rate-limit': 'benchmark-secret-key'
          }
        }, (res) => {
          res.on('data', () => {});
          res.on('end', () => {
            if (res.statusCode === 200) {
              success++;
            } else {
              failed++;
            }
            latencies.push(Date.now() - reqStartTime);
            activeRequests--;
            sendNext();
          });
        });

        req.on('error', () => {
          failed++;
          activeRequests--;
          sendNext();
        });
      }
    };

    sendNext();
  });
}

// 2. HTTP/2 Benchmarker
function runHttp2Benchmark(requests, concurrency, ipAddress, port, reqPath) {
  return new Promise((resolve) => {
    const startTime = Date.now();
    let success = 0;
    let failed = 0;
    const latencies = [];
    let requestsSent = 0;
    let activeRequests = 0;

    const client = http2.connect(`https://${ipAddress}:${port}`, {
      rejectUnauthorized: false
    });

    client.on('error', (err) => {
      console.error('HTTP/2 Connection Error:', err.message);
      resolve({ rps: 0, minLatency: 0, maxLatency: 0, avgLatency: 0, success: 0, failed: requests });
    });

    const sendNext = () => {
      if (requestsSent >= requests) {
        if (activeRequests === 0) {
          const duration = (Date.now() - startTime) / 1000;
          const rps = parseFloat((success / duration).toFixed(2));
          latencies.sort((a, b) => a - b);
          const avgLatency = latencies.length ? latencies.reduce((a, b) => a + b, 0) / latencies.length : 0;
          const minLatency = latencies.length ? latencies[0] : 0;
          const maxLatency = latencies.length ? latencies[latencies.length - 1] : 0;

          client.close();
          resolve({
            rps,
            minLatency,
            maxLatency,
            avgLatency: parseFloat(avgLatency.toFixed(2)),
            success,
            failed
          });
        }
        return;
      }

      while (activeRequests < concurrency && requestsSent < requests) {
        requestsSent++;
        activeRequests++;
        const reqStartTime = Date.now();

        const req = client.request({
          [http2.constants.HTTP2_HEADER_PATH]: reqPath,
          [http2.constants.HTTP2_HEADER_AUTHORITY]: 'localhost',
          'x-bypass-rate-limit': 'benchmark-secret-key'
        });

        req.on('response', (headers) => {
          const status = parseInt(headers[':status'], 10);
          if (status === 200) {
            success++;
          } else {
            failed++;
          }
        });

        req.on('data', () => {});
        req.on('end', () => {
          latencies.push(Date.now() - reqStartTime);
          activeRequests--;
          sendNext();
        });

        req.on('error', () => {
          failed++;
          activeRequests--;
          sendNext();
        });
      }
    };

    sendNext();
  });
}

// 3. HTTP/3 Benchmarker with Connection Recycling
function runHttp3Benchmark(requests, concurrency, ipAddress, port, reqPath) {
  return new Promise((resolve) => {
    const startTime = Date.now();
    let success = 0;
    let failed = 0;
    const latencies = [];
    let requestsSent = 0;
    let activeRequests = 0;

    let session = null;
    let connecting = false;
    const pendingStreams = [];

    const getSession = () => {
      return new Promise((resolveSession, rejectSession) => {
        if (session) {
          return resolveSession(session);
        }
        if (connecting) {
          pendingStreams.push({ resolve: resolveSession, reject: rejectSession });
          return;
        }

        connecting = true;
        const newSession = connect(`${ipAddress}:${port}`, {
          rejectUnauthorized: false
        });

        newSession.on('error', (err) => {
          connecting = false;
          rejectSession(err);
          while (pendingStreams.length > 0) {
            const p = pendingStreams.shift();
            p.reject(err);
          }
        });

        newSession.on('close', () => {
          if (session === newSession) session = null;
        });

        newSession.on('connect', () => {
          session = newSession;
          connecting = false;
          resolveSession(session);

          // Flush pending requests
          while (pendingStreams.length > 0) {
            const p = pendingStreams.shift();
            p.resolve(session);
          }
        });
      });
    };

    const sendNext = async () => {
      if (requestsSent >= requests) {
        if (activeRequests === 0) {
          const duration = (Date.now() - startTime) / 1000;
          const rps = parseFloat((success / duration).toFixed(2));
          latencies.sort((a, b) => a - b);
          const avgLatency = latencies.length ? latencies.reduce((a, b) => a + b, 0) / latencies.length : 0;
          const minLatency = latencies.length ? latencies[0] : 0;
          const maxLatency = latencies.length ? latencies[latencies.length - 1] : 0;

          if (session) {
            try {
              await session.close();
            } catch (e) {}
          }
          resolve({
            rps,
            minLatency,
            maxLatency,
            avgLatency: parseFloat(avgLatency.toFixed(2)),
            success,
            failed
          });
        }
        return;
      }

      while (activeRequests < concurrency && requestsSent < requests) {
        requestsSent++;
        activeRequests++;
        const reqStartTime = Date.now();

        try {
          const activeSession = await getSession();

          const stream = activeSession.request({
            ':method': 'GET',
            ':path': reqPath,
            ':authority': 'localhost',
            ':scheme': 'https',
            'x-bypass-rate-limit': 'benchmark-secret-key'
          }, { endStream: true });

          stream.on('response', (headers) => {
            const status = parseInt(headers[':status'], 10);
            if (status === 200) {
              success++;
            } else {
              failed++;
            }
          });

          stream.on('data', () => {});
          stream.on('end', () => {
            latencies.push(Date.now() - reqStartTime);
            activeRequests--;
            sendNext();
          });

          stream.on('error', () => {
            failed++;
            activeRequests--;
            sendNext();
          });
        } catch (err) {
          // If session connection failed to establish
          failed++;
          activeRequests--;
          sendNext();
        }
      }
    };

    sendNext();
  });
}

// Generate Markdown Comparison Report
function writeMarkdownReport(results) {
  const reportPath = path.join(__dirname, 'benchmark_report.md');
  const timestamp = new Date().toISOString();
  const dateStr = timestamp.split('T')[0];
  const reportsDir = path.join(__dirname, 'reports');
  
  // Ensure the reports directory exists
  fs.mkdirSync(reportsDir, { recursive: true });
  const timestampedReportPath = path.join(reportsDir, `report-${dateStr}.md`);

  let md = `# Express Servers Protocol Benchmark Report

Generated at: \`${timestamp}\`
Parameters:
- **Total Requests**: \`${requests}\`
- **Concurrency**: \`${concurrency}\`

---

## 📊 Comparison Table

| Protocol / Port Configuration | Requests/Sec | Avg Latency | Min Latency | Max Latency | Success Rate |
| :--- | :--- | :--- | :--- | :--- | :--- |
`;

  results.forEach(res => {
    if (res.error) {
      md += `| **${res.name}** | *Error* | *Error* | *Error* | *Error* | \`0.0%\` |\n`;
    } else {
      const rate = ((res.success / (res.success + res.failed)) * 100).toFixed(1);
      md += `| **${res.name}** | \`${res.rps.toFixed(1)} req/s\` | \`${res.avgLatency.toFixed(1)}ms\` | \`${res.minLatency.toFixed(1)}ms\` | \`${res.maxLatency.toFixed(1)}ms\` | \`${rate}%\` |\n`;
    }
  });

  md += `
---

## 💡 Protocol Observations

1. **HTTP/3 (QUIC)**: Runs over UDP, eliminating head-of-line blocking at the transport layer. It exhibits high throughput under load due to single-socket multiplexing over UDP.
2. **HTTP/2**: Uses TCP multiplexing. It performs significantly better than HTTP/1.1 under concurrency by running requests in parallel streams over a single TCP connection.
3. **HTTP/1.1 (HTTPS)**: Suffers from TCP head-of-line blocking. Under high concurrency, it is constrained by connection overhead and handshakes.
`;

  fs.writeFileSync(reportPath, md);
  fs.writeFileSync(timestampedReportPath, md);
  console.log(`Markdown report written to: ${reportPath}`);
  console.log(`Historical markdown report written to: ${timestampedReportPath}`);
}

// Main runner execution
async function main() {
  const results = [];
  const targetsToRun = target === 'all' 
    ? ['http1', 'https-fallback', 'http2', 'http3-fallback', 'http3'] 
    : [target];

  for (const t of targetsToRun) {
    const config = endpoints[t];
    if (!config) {
      console.error(`Unknown benchmark target: ${t}`);
      continue;
    }

    console.log(`Running benchmark for: ${config.name}...`);
    
    // Resolve DNS hostname to container IP
    const ipAddress = await resolveHostToIp(config.host);
    let result;

    try {
      if (t === 'http3') {
        result = await runHttp3Benchmark(requests, concurrency, ipAddress, config.port, config.path);
      } else if (t === 'http2') {
        result = await runHttp2Benchmark(requests, concurrency, ipAddress, config.port, config.path);
      } else {
        result = await runHttp1Benchmark(requests, concurrency, ipAddress, config.port, config.path);
      }
      console.log(`-> Completed: ${result.rps} req/s, avg latency: ${result.avgLatency}ms\n`);
      results.push({ name: config.name, ...result });
    } catch (err) {
      console.error(`Failed running ${config.name} benchmark:`, err.message);
      results.push({ name: config.name, error: err.message });
    }
  }

  writeMarkdownReport(results);
}

main();
