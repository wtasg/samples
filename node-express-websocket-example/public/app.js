// ============================================================
// WebSocket Dashboard — app.js
// Full-duplex WebSocket client: server-push CPU + client commands
// ============================================================

'use strict';

// --- State ---
let ws = null;
let cpuHistory = [];
const MAX_HISTORY = 50;
let peakCpu = 0;
let cpuSum = 0;
let cpuCount = 0;
let pingCount = 0;
let pingMin = Infinity;
let pingMax = -Infinity;
let pingPendingTime = null;
let reconnectTimer = null;
let reconnectDelay = 1000;
const MAX_RECONNECT_DELAY = 16000;

// --- DOM ---
const connDot    = document.getElementById('conn-dot');
const connText   = document.getElementById('conn-text');
const clientChip = document.getElementById('client-chip');

const cpuCanvas   = document.getElementById('cpu-canvas');
const cpuCurrent  = document.getElementById('cpu-current');
const cpuPeak     = document.getElementById('cpu-peak');
const cpuAvg      = document.getElementById('cpu-avg');
const clientCount = document.getElementById('client-count');

const intervalSlider  = document.getElementById('interval-slider');
const intervalDisplay = document.getElementById('interval-display');
const applyIntervalBtn = document.getElementById('apply-interval-btn');
const intervalAck     = document.getElementById('interval-ack');

const pingBtn    = document.getElementById('ping-btn');
const pingRtt    = document.getElementById('ping-rtt');
const pingMinMax = document.getElementById('ping-minmax');
const pingCountEl = document.getElementById('ping-count');

const broadcastInput = document.getElementById('broadcast-input');
const broadcastBtn   = document.getElementById('broadcast-btn');
const broadcastLog   = document.getElementById('broadcast-log');

const logOutput   = document.getElementById('log-output');
const clearLogBtn = document.getElementById('clear-log-btn');

const hbBtn     = document.getElementById('hb-btn');
const hbStatus  = document.getElementById('hb-status');
const hbUptime  = document.getElementById('hb-uptime');
const hbClients = document.getElementById('hb-clients');

// --- Canvas context ---
const ctx = cpuCanvas.getContext('2d');

// ============================================================
// Logging
// ============================================================
function log(type, msg) {
  const line = document.createElement('div');
  line.className = `log-line log-${type}`;
  const ts = new Date().toLocaleTimeString();
  line.textContent = `[${ts}] ${msg}`;
  logOutput.appendChild(line);
  logOutput.scrollTop = logOutput.scrollHeight;
  while (logOutput.children.length > 80) logOutput.removeChild(logOutput.firstChild);
}

// ============================================================
// Connection State UI
// ============================================================
function setConnState(state, text) {
  connDot.className = `conn-dot ${state}`;
  connText.textContent = text;
}

// ============================================================
// WebSocket Connection
// ============================================================
function connect() {
  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
  const url = `${proto}://${window.location.host}/ws`;

  setConnState('connecting', 'Connecting…');
  log('system', `Opening WebSocket connection to ${url}`);

  ws = new WebSocket(url);

  ws.addEventListener('open', () => {
    setConnState('connected', 'Connected (WSS)');
    log('sse', 'WebSocket connection established over WSS');
    reconnectDelay = 1000; // reset backoff
    clearTimeout(reconnectTimer);
  });

  ws.addEventListener('message', (event) => {
    let msg;
    try { msg = JSON.parse(event.data); }
    catch { log('error', `Unparseable message: ${event.data}`); return; }

    handleMessage(msg);
  });

  ws.addEventListener('close', (event) => {
    setConnState('error', `Disconnected (${event.code})`);
    log('error', `WebSocket closed — code ${event.code}. Reconnecting in ${reconnectDelay / 1000}s…`);
    scheduleReconnect();
  });

  ws.addEventListener('error', () => {
    log('error', 'WebSocket error');
  });
}

function scheduleReconnect() {
  clearTimeout(reconnectTimer);
  reconnectTimer = setTimeout(() => {
    reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_DELAY);
    connect();
  }, reconnectDelay);
}

// ============================================================
// Message Handling
// ============================================================
function handleMessage(msg) {
  switch (msg.type) {
    case 'welcome':
      log('sse', `Server: ${msg.message} — assigned ID: ${msg.clientId}`);
      clientChip.textContent = msg.clientId;
      break;

    case 'cpu': {
      const cpu = parseFloat(msg.cpu);
      cpuHistory.push(cpu);
      if (cpuHistory.length > MAX_HISTORY) cpuHistory.shift();

      peakCpu = Math.max(peakCpu, cpu);
      cpuSum += cpu; cpuCount++;

      cpuCurrent.textContent = `${cpu.toFixed(1)}%`;
      cpuPeak.textContent    = `${peakCpu.toFixed(1)}%`;
      cpuAvg.textContent     = `${(cpuSum / cpuCount).toFixed(1)}%`;
      clientCount.textContent = msg.clients;

      drawChart();
      break;
    }

    case 'pong': {
      if (pingPendingTime !== null) {
        const rtt = Date.now() - pingPendingTime;
        pingPendingTime = null;
        pingMin = Math.min(pingMin, rtt);
        pingMax = Math.max(pingMax, rtt);
        pingRtt.textContent    = `${rtt} ms`;
        pingMinMax.textContent = `${pingMin} / ${pingMax} ms`;
        log('receive', `Pong received — RTT: ${rtt}ms`);
      }
      break;
    }

    case 'interval-ack':
      log('receive', `Interval acknowledged: ${msg.intervalMs}ms`);
      showIntervalAck();
      break;

    case 'broadcast': {
      const entry = document.createElement('div');
      entry.className = 'broadcast-entry';
      const ts = new Date(msg.timestamp).toLocaleTimeString();
      entry.innerHTML =
        `<span class="bc-from">${escHtml(msg.from)}</span>` +
        `<span class="bc-text">${escHtml(msg.text)}</span>` +
        `<span class="bc-time">${ts}</span>`;
      // Remove placeholder if present
      const placeholder = broadcastLog.querySelector('.log-placeholder');
      if (placeholder) placeholder.remove();
      broadcastLog.appendChild(entry);
      broadcastLog.scrollTop = broadcastLog.scrollHeight;
      log('bc', `Broadcast from ${msg.from}: "${msg.text}"`);
      break;
    }

    case 'error':
      log('error', `Server error: ${msg.message}`);
      break;

    default:
      log('receive', `Unknown message type: ${msg.type}`);
  }
}

// ============================================================
// Send Helpers
// ============================================================
function send(obj) {
  if (!ws || ws.readyState !== WebSocket.OPEN) {
    log('error', 'Cannot send — WebSocket not connected');
    return false;
  }
  const raw = JSON.stringify(obj);
  ws.send(raw);
  return true;
}

// ============================================================
// Canvas Chart
// ============================================================
function resizeCanvas() {
  const dpr = window.devicePixelRatio || 1;
  const rect = cpuCanvas.getBoundingClientRect();
  cpuCanvas.width  = rect.width  * dpr;
  cpuCanvas.height = rect.height * dpr;
  ctx.scale(dpr, dpr);
  drawChart();
}

function drawChart() {
  const W = cpuCanvas.width  / (window.devicePixelRatio || 1);
  const H = cpuCanvas.height / (window.devicePixelRatio || 1);

  ctx.clearRect(0, 0, W, H);

  // Grid
  const gridLines = 4;
  ctx.lineWidth = 1;
  for (let i = 0; i <= gridLines; i++) {
    const y = (H / gridLines) * i;
    ctx.strokeStyle = 'rgba(255,255,255,0.04)';
    ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke();
    ctx.fillStyle = 'rgba(255,255,255,0.18)';
    ctx.font = `9px "JetBrains Mono", monospace`;
    ctx.fillText(`${100 - (100 / gridLines) * i}%`, 6, y < 10 ? 12 : y - 3);
  }

  if (cpuHistory.length < 2) return;

  const pL = 36, pR = 10, pT = 8, pB = 8;
  const cW = W - pL - pR;
  const cH = H - pT - pB;
  const step = cW / (MAX_HISTORY - 1);

  const coord = (i) => ({
    x: pL + i * step,
    y: pT + cH - (cpuHistory[i] / 100) * cH
  });

  // Fill area
  ctx.beginPath();
  const s0 = coord(0);
  ctx.moveTo(s0.x, pT + cH);
  ctx.lineTo(s0.x, s0.y);
  for (let i = 1; i < cpuHistory.length; i++) {
    const p = coord(i);
    ctx.lineTo(p.x, p.y);
  }
  const last = coord(cpuHistory.length - 1);
  ctx.lineTo(last.x, pT + cH);
  ctx.closePath();
  const areaGrad = ctx.createLinearGradient(0, pT, 0, pT + cH);
  areaGrad.addColorStop(0, 'rgba(99,102,241,0.25)');
  areaGrad.addColorStop(1, 'rgba(99,102,241,0.0)');
  ctx.fillStyle = areaGrad;
  ctx.fill();

  // Stroke line
  ctx.beginPath();
  ctx.moveTo(s0.x, s0.y);
  for (let i = 1; i < cpuHistory.length; i++) {
    const p = coord(i);
    ctx.lineTo(p.x, p.y);
  }
  const lineGrad = ctx.createLinearGradient(pL, 0, W - pR, 0);
  lineGrad.addColorStop(0, '#6366f1');
  lineGrad.addColorStop(1, '#22d3ee');
  ctx.strokeStyle = lineGrad;
  ctx.lineWidth = 2.5;
  ctx.shadowColor = 'rgba(99,102,241,0.5)';
  ctx.shadowBlur = 10;
  ctx.stroke();
  ctx.shadowBlur = 0;

  // Pulse dot at latest
  ctx.beginPath();
  ctx.arc(last.x, last.y, 5, 0, 2 * Math.PI);
  ctx.fillStyle = '#22d3ee';
  ctx.fill();
  ctx.beginPath();
  ctx.arc(last.x, last.y, 9, 0, 2 * Math.PI);
  ctx.strokeStyle = 'rgba(34,211,238,0.35)';
  ctx.lineWidth = 1;
  ctx.stroke();
}

// ============================================================
// Interval Ack Animation
// ============================================================
let ackTimer = null;
function showIntervalAck() {
  intervalAck.classList.remove('hide');
  clearTimeout(ackTimer);
  ackTimer = setTimeout(() => intervalAck.classList.add('hide'), 2500);
}

// ============================================================
// Heartbeat REST
// ============================================================
async function checkHeartbeat() {
  hbBtn.disabled = true;
  log('send', 'GET /heartbeat (REST)');
  try {
    const res = await fetch('/heartbeat');
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = await res.json();
    hbStatus.textContent = data.status;
    hbStatus.className = 'stat-value text-green';
    const u = Math.floor(data.uptime);
    hbUptime.textContent  = `${Math.floor(u/3600)}h ${Math.floor((u%3600)/60)}m ${u%60}s`;
    hbClients.textContent = data.connections;
    log('receive', `200 OK — uptime: ${u}s, WS clients: ${data.connections}`);
  } catch (err) {
    hbStatus.textContent = 'Error';
    hbStatus.className = 'stat-value text-red';
    log('error', `Heartbeat failed: ${err.message}`);
  } finally {
    hbBtn.disabled = false;
  }
}

// ============================================================
// Utilities
// ============================================================
function escHtml(str) {
  return String(str)
    .replace(/&/g,'&amp;').replace(/</g,'&lt;')
    .replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function updateSliderTrack() {
  const min = parseFloat(intervalSlider.min);
  const max = parseFloat(intervalSlider.max);
  const val = parseFloat(intervalSlider.value);
  const pct = ((val - min) / (max - min)) * 100;
  intervalSlider.style.setProperty('--pct', `${pct}%`);
}

// ============================================================
// Event Listeners
// ============================================================
intervalSlider.addEventListener('input', () => {
  intervalDisplay.textContent = intervalSlider.value;
  updateSliderTrack();
});

applyIntervalBtn.addEventListener('click', () => {
  const ms = parseInt(intervalSlider.value, 10);
  if (send({ type: 'set-interval', intervalMs: ms })) {
    log('send', `Sent set-interval: ${ms}ms`);
  }
});

pingBtn.addEventListener('click', () => {
  pingPendingTime = Date.now();
  pingCount++;
  pingCountEl.textContent = pingCount;
  if (send({ type: 'ping', payload: `ping-${pingCount}` })) {
    log('send', `Sent ping #${pingCount}`);
  }
});

broadcastBtn.addEventListener('click', () => {
  const text = broadcastInput.value.trim();
  if (!text) return;
  if (send({ type: 'broadcast', text })) {
    log('send', `Sent broadcast: "${text}"`);
    broadcastInput.value = '';
  }
});

broadcastInput.addEventListener('keydown', (e) => {
  if (e.key === 'Enter') broadcastBtn.click();
});

clearLogBtn.addEventListener('click', () => {
  logOutput.innerHTML = '';
  log('system', 'Console cleared.');
});

hbBtn.addEventListener('click', checkHeartbeat);

// ============================================================
// Init
// ============================================================
window.addEventListener('resize', resizeCanvas);

window.addEventListener('load', () => {
  updateSliderTrack();
  setTimeout(() => {
    resizeCanvas();
    connect();
    checkHeartbeat();
    setInterval(checkHeartbeat, 30000);
  }, 150);
});
