const express = require('express');
const rateLimit = require('express-rate-limit');
const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { WebSocketServer } = require('ws');

// 1. Load SSL/TLS certificates
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
const PORT = parseInt(process.env.PORT || 60008, 10);

// 2. Rate limiter for REST endpoints
const apiLimiter = rateLimit({
  windowMs: 15 * 1000,
  max: 15,
  standardHeaders: true,
  legacyHeaders: false,
  message: {
    error: 'Too many requests',
    message: 'Rate limit exceeded. Please wait a few seconds and try again.'
  },
  statusCode: 429
});

app.use('/heartbeat', apiLimiter);
app.use(express.static(path.join(__dirname, 'public')));

// Helper: calculate CPU usage over a 100ms sampling window
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
        totalDiff += endTotal - startTotal;
        idleDiff += end.idle - start.idle;
      }
      const cpuUsage = totalDiff === 0 ? 0 : 100 * (1 - idleDiff / totalDiff);
      resolve(parseFloat(cpuUsage.toFixed(2)));
    }, 100);
  });
}

// 3. REST Endpoints
app.get('/heartbeat', (req, res) => {
  res.json({
    status: 'alive',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    connections: wss ? wss.clients.size : 0
  });
});

// 4. HTTPS Server + WebSocket Server
const server = https.createServer(sslOptions, app);

// Attach WebSocket server to the same HTTPS server (wss://)
let wss;

// Track per-client state
const clientState = new Map();

function startWebSocketServer() {
  wss = new WebSocketServer({ server, path: '/ws' });

  wss.on('connection', (ws, req) => {
    const clientId = `client-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;
    const remoteIp = req.socket.remoteAddress || 'unknown';

    // Default: push CPU every 1000ms
    let pushIntervalMs = 1000;
    let intervalId = null;

    const state = { clientId, remoteIp, pushIntervalMs, connectedAt: new Date().toISOString() };
    clientState.set(ws, state);

    console.log(`[WebSocket] Client connected: ${clientId} from ${remoteIp}`);

    // Send a welcome message with the client's ID
    ws.send(JSON.stringify({
      type: 'welcome',
      clientId,
      message: 'WebSocket connection established over WSS',
      timestamp: new Date().toISOString()
    }));

    // Start pushing CPU metrics at the current interval
    function startPushing() {
      if (intervalId) clearInterval(intervalId);
      intervalId = setInterval(async () => {
        if (ws.readyState !== ws.OPEN) return;
        try {
          const cpu = await getCpuUsage();
          ws.send(JSON.stringify({
            type: 'cpu',
            cpu,
            clients: wss.clients.size,
            timestamp: new Date().toISOString()
          }));
        } catch (err) {
          console.error(`[WebSocket] Error pushing CPU to ${clientId}:`, err.message);
        }
      }, state.pushIntervalMs);
    }

    startPushing();

    // Handle incoming messages from the client
    ws.on('message', (rawData) => {
      let msg;
      try {
        msg = JSON.parse(rawData.toString());
      } catch {
        ws.send(JSON.stringify({ type: 'error', message: 'Invalid JSON message' }));
        return;
      }

      console.log(`[WebSocket] Message from ${clientId}:`, msg);

      switch (msg.type) {
        case 'ping': {
          // Client → server ping; server replies with pong
          ws.send(JSON.stringify({
            type: 'pong',
            echo: msg.payload || null,
            serverTime: new Date().toISOString()
          }));
          break;
        }
        case 'set-interval': {
          // Client requests a different push interval (clamp: 250ms–5000ms)
          const requested = parseInt(msg.intervalMs, 10);
          if (!isNaN(requested)) {
            state.pushIntervalMs = Math.min(5000, Math.max(250, requested));
            startPushing();
            ws.send(JSON.stringify({
              type: 'interval-ack',
              intervalMs: state.pushIntervalMs,
              timestamp: new Date().toISOString()
            }));
            console.log(`[WebSocket] ${clientId} set interval to ${state.pushIntervalMs}ms`);
          }
          break;
        }
        case 'broadcast': {
          // Client sends a message to all connected clients
          const broadcastMsg = JSON.stringify({
            type: 'broadcast',
            from: clientId,
            text: (msg.text || '').toString().slice(0, 256),
            timestamp: new Date().toISOString()
          });
          wss.clients.forEach((client) => {
            if (client.readyState === client.OPEN) {
              client.send(broadcastMsg);
            }
          });
          break;
        }
        default:
          ws.send(JSON.stringify({ type: 'error', message: `Unknown message type: ${msg.type}` }));
      }
    });

    ws.on('close', (code, reason) => {
      clearInterval(intervalId);
      clientState.delete(ws);
      console.log(`[WebSocket] Client disconnected: ${clientId} (code ${code})`);
    });

    ws.on('error', (err) => {
      console.error(`[WebSocket] Error on ${clientId}:`, err.message);
    });
  });

  console.log(`[WebSocket] WSS server attached to HTTPS server at wss://localhost:${PORT}/ws`);
}

if (process.env.NODE_ENV !== 'test') {
  server.listen(PORT, () => {
    console.log(`HTTPS server running at: https://localhost:${PORT}`);
    startWebSocketServer();
  });
} else {
  // In test mode, caller starts the server manually
  startWebSocketServer();
}

module.exports = { server, app, wss: () => wss, PORT, startWebSocketServer };
