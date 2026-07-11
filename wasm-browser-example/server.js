const express = require('express');
const path = require('path');

const app = express();
const PORT = parseInt(process.env.PORT || 60010, 10);

// Serve static files from public/
// Must set correct MIME type for WASM files
app.use((req, res, next) => {
  if (req.path.endsWith('.wasm')) {
    res.setHeader('Content-Type', 'application/wasm');
  }
  next();
});

app.use(express.static(path.join(__dirname, 'public')));

// Health check endpoint
app.get('/heartbeat', (req, res) => {
  res.json({ status: 'alive', timestamp: new Date().toISOString(), uptime: process.uptime() });
});

if (process.env.NODE_ENV !== 'test') {
  app.listen(PORT, () => {
    console.log(`WASM Browser Example server running at: http://localhost:${PORT}`);
    console.log(`Open http://localhost:${PORT} in your browser to see the demo.`);
  });
}

module.exports = { app, PORT };
