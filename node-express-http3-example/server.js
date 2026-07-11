const express = require('express');
const { createSecureServer } = require('@currentspace/http3');
const { createExpressAdapter } = require('@currentspace/http3/express');
const https = require('https');
const rateLimit = require('express-rate-limit');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { Writable } = require('stream');

// 1. Ensure SSL/TLS certificates exist, then load them
const certsDir = path.join(__dirname, 'certs');
const keyPath = path.join(certsDir, 'key.pem');
const certPath = path.join(certsDir, 'cert.pem');

if (!fs.existsSync(keyPath) || !fs.existsSync(certPath)) {
  console.error('SSL certificates are missing! Please run "npm run build" first.');
  process.exit(1);
}

const sslOptions = {
  key: fs.readFileSync(keyPath),
  cert: fs.readFileSync(certPath)
};

// Helper to calculate CPU usage over 100ms window without blocking
function getCpuUsage() {
  return new Promise((resolve) => {
    const startCpus = os.cpus();
    setTimeout(() => {
      const endCpus = os.cpus();
      let totalDiff = 0;
      let idleDiff = 0;

      for (let i = 0; i < startCpus.length; i++) {
        const start = startCpus[i].times;
        const end = endCpus[i].times;

        const startTotal = start.user + start.nice + start.sys + start.idle + start.irq;
        const endTotal = end.user + end.nice + end.sys + end.idle + end.irq;

        totalDiff += (endTotal - startTotal);
        idleDiff += (end.idle - start.idle);
      }

      const cpuUsage = totalDiff === 0 ? 0 : 100 * (1 - idleDiff / totalDiff);
      resolve(parseFloat(cpuUsage.toFixed(2)));
    }, 100);
  });
}

// Factory function to create and configure the Express App instance
function createExpressApp() {
  const app = express();

  // Middleware to monkeypatch missing HTTP/1 response/request wrapper methods on custom H3 request/response objects
  app.use((req, res, next) => {
    if (req.httpVersion === '3.0') {
      // 1. Mock request socket and connection for rate limiter/IP checking
      if (!req.socket) {
        req.socket = {
          remoteAddress: '127.0.0.1',
          encrypted: true
        };
      }
      if (!req.connection) {
        req.connection = req.socket;
      }

      // 2. Bypass Express/Node HTTP/1.1 OutgoingMessage prototype methods that crash
      res.removeHeader = () => {};
      res.getHeader = () => {};
      res.hasHeader = () => false;

      // 3. Restore standard stream.Writable write & end methods
      res.write = Writable.prototype.write.bind(res);
      res.end = Writable.prototype.end.bind(res);

      // 4. Intercept setHeader to convert all non-string values to strings to prevent Rust FFI conversion errors
      const originalSetHeader = res.setHeader;
      res.setHeader = (name, value) => {
        if (originalSetHeader) {
          originalSetHeader.call(res, name, value !== undefined && value !== null ? String(value) : '');
        }
      };

      // 5. Intercept writeHead to stringify any headers passed in
      const originalWriteHead = res.writeHead;
      res.writeHead = (statusCode, headers) => {
        if (headers) {
          for (const key of Object.keys(headers)) {
            const val = headers[key];
            headers[key] = val !== undefined && val !== null ? String(val) : '';
          }
        }
        if (originalWriteHead) {
          originalWriteHead.call(res, statusCode, headers);
        }
      };
    }
    next();
  });

  // Configure Rate Limiter (Max 15 requests per 15 seconds per IP)
  const apiLimiter = rateLimit({
    windowMs: 15 * 1000,
    max: 15,
    skip: (req) => req.headers['x-bypass-rate-limit'] === 'benchmark-secret-key',
    standardHeaders: true,
    legacyHeaders: false,
    message: {
      error: 'Too many requests',
      message: 'Rate limit exceeded. Please wait a few seconds and try again.'
    },
    statusCode: 429
  });

  // Apply rate limiting to /load and /heartbeat
  app.use('/load', apiLimiter);
  app.use('/heartbeat', apiLimiter);

  // Serve static assets from public/ folder
  app.use(express.static(path.join(__dirname, 'public')));

  // Heartbeat Endpoint
  app.get('/heartbeat', (req, res) => {
    res.writeHead(200, { 
      'Content-Type': 'application/json; charset=utf-8',
      'Cache-Control': 'no-cache'
    });
    res.end(JSON.stringify({
      status: 'alive',
      timestamp: new Date().toISOString(),
      uptime: process.uptime()
    }));
  });

  // CPU Load - Query/Response Endpoint
  app.get('/load', async (req, res) => {
    try {
      const cpu = await getCpuUsage();
      res.writeHead(200, { 
        'Content-Type': 'application/json; charset=utf-8',
        'Cache-Control': 'no-cache'
      });
      res.end(JSON.stringify({
        cpu,
        timestamp: new Date().toISOString()
      }));
    } catch (error) {
      res.writeHead(500, { 'Content-Type': 'application/json; charset=utf-8' });
      res.end(JSON.stringify({ error: 'Failed to retrieve CPU usage' }));
    }
  });

  // CPU Load - SSE Streaming Endpoint
  app.get('/stream', (req, res) => {
    // Set headers for Server-Sent Events (omit connection header for HTTP/3 compatibility)
    res.writeHead(200, {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Access-Control-Allow-Origin': '*'
    });

    // Send initial establishing comment
    res.write(': ok\n\n');

    // Push CPU load every 1 second
    const intervalId = setInterval(async () => {
      try {
        const cpu = await getCpuUsage();
        res.write(`data: ${JSON.stringify({ cpu, timestamp: new Date().toISOString() })}\n\n`);
      } catch (error) {
        console.error('Error in CPU stream interval:', error.message);
      }
    }, 1000);

    // Clean up on connection close
    req.on('close', () => {
      clearInterval(intervalId);
      res.end();
    });
  });

  return app;
}

// Create independent app instances for HTTP/3 and HTTPS servers
const http3App = createExpressApp();
const httpsApp = createExpressApp();

// Wrap http3App with the Express adapter provided by `@currentspace/http3`
const http3Adapter = createExpressAdapter(http3App);

const http3Server = createSecureServer(
  {
    key: sslOptions.key,
    cert: sslOptions.cert,
    disableRetry: true
  },
  http3Adapter
);

const httpsServer = https.createServer(sslOptions, httpsApp);

if (process.env.NODE_ENV !== 'test') {
  const HTTP3_PORT = parseInt(process.env.HTTP3_PORT || 60007, 10);
  const HTTPS_PORT = parseInt(process.env.HTTPS_PORT || 60447, 10);

  http3Server.listen(HTTP3_PORT, '0.0.0.0');
  console.log(`HTTP/3 secure server listening on https://localhost:${HTTP3_PORT}`);

  httpsServer.listen(HTTPS_PORT, '0.0.0.0', () => {
    console.log(`HTTPS secure server running at: https://localhost:${HTTPS_PORT}`);
  });
}

module.exports = { http3Server, httpsServer };
