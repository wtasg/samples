const express = require('express');
const rateLimit = require('express-rate-limit');
const https = require('https');
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

const app = express();
const PORT = process.env.PORT || 60005;

// 2. Configure Rate Limiters
// We configure a rate limiter for the API/heartbeat endpoints.
// To make it easy to demonstrate in the UI, we limit to 15 requests per 15 seconds.
const apiLimiter = rateLimit({
  windowMs: 15 * 1000, // 15 seconds
  max: 15, // Limit each IP to 15 requests per window
  standardHeaders: true, // Return rate limit info in the `RateLimit-*` headers
  legacyHeaders: false, // Disable the `X-RateLimit-*` headers
  message: {
    error: 'Too many requests',
    message: 'Rate limit exceeded. Please wait a few seconds and try again.'
  },
  statusCode: 429
});

// Apply rate limiting to /api/cpu/load and /heartbeat
app.use('/api/cpu/load', apiLimiter);
app.use('/heartbeat', apiLimiter);

// Serve static assets from public/ folder
app.use(express.static(path.join(__dirname, 'public')));

// Helper to calculate CPU usage over a short window (100ms) to avoid blocking the event loop
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

// 3. Define Endpoints

// Heartbeat Endpoint
app.get('/heartbeat', (req, res) => {
  res.json({
    status: 'alive',
    timestamp: new Date().toISOString(),
    uptime: process.uptime()
  });
});

// CPU Load - Query/Response Endpoint
app.get('/api/cpu/load', async (req, res) => {
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
app.get('/api/cpu/stream', (req, res) => {
  // Set headers for Server-Sent Events
  res.writeHead(200, {
    'Content-Type': 'text/event-stream',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive',
    'Access-Control-Allow-Origin': '*'
  });

  // Send an initial comment to establish the connection
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

// 4. Start the HTTPS Server
const server = https.createServer(sslOptions, app);

if (process.env.NODE_ENV !== 'test') {
  server.listen(PORT, () => {
    console.log(`HTTPS Secure server running at: https://localhost:${PORT}`);
  });
}

module.exports = { server, app, PORT };
