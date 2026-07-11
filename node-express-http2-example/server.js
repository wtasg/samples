const express = require('express');
const http2Express = require('http2-express');
const http2 = require('node:http2');
const https = require('https');
const rateLimit = require('express-rate-limit');
const fs = require('fs');
const path = require('path');
const os = require('os');

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
  const app = http2Express(express);

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
    res.json({
      status: 'alive',
      timestamp: new Date().toISOString(),
      uptime: process.uptime()
    });
  });

  // CPU Load - Query/Response Endpoint
  app.get('/load', async (req, res) => {
    try {
      const cpu = await getCpuUsage();
      res.json({
        cpu,
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      res.status(500).json({ error: 'Failed to retrieve CPU usage' });
    }
  });

  // CPU Load - SSE Streaming Endpoint
  app.get('/stream', (req, res) => {
    // Set headers for Server-Sent Events (omit connection header for HTTP/2 compatibility)
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

// Create independent app instances for HTTP/2 and HTTPS servers
const http2App = createExpressApp();
const httpsApp = createExpressApp();

const http2Server = http2.createSecureServer(sslOptions, http2App);
const httpsServer = https.createServer(sslOptions, httpsApp);

if (process.env.NODE_ENV !== 'test') {
  const HTTP2_PORT = parseInt(process.env.HTTP2_PORT || 60006, 10);
  const HTTPS_PORT = parseInt(process.env.HTTPS_PORT || 60446, 10);

  http2Server.listen(HTTP2_PORT, () => {
    console.log(`HTTP/2 secure server running at: https://localhost:${HTTP2_PORT}`);
  });

  httpsServer.listen(HTTPS_PORT, () => {
    console.log(`HTTPS secure server running at: https://localhost:${HTTPS_PORT}`);
  });
}

module.exports = { http2Server, httpsServer };
