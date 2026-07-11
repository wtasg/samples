// ============================================================
// gRPC Connect Dashboard — app.js
// Uses the Connect protocol (fetch-based, works from browsers)
// Calls: GetStatus (unary), StreamCPU (server-streaming), Echo (unary)
// ============================================================
'use strict';

// ── Connect protocol helpers ─────────────────────────────────
// We use the Connect HTTP/JSON protocol directly via fetch — no generated client needed.
// POST /monitor.v1.MonitorService/MethodName
// Content-Type: application/json
// Connect-Protocol-Version: 1

const BASE   = window.location.origin;
const SVC    = '/monitor.v1.MonitorService';

async function callUnary(method, body) {
  const res = await fetch(`${BASE}${SVC}/${method}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Connect-Protocol-Version': '1'
    },
    body: JSON.stringify(body)
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${res.statusText}: ${text}`);
  }
  return res.json();
}

// Server-streaming: uses the Connect streaming protocol.
// Each message is a length-prefixed JSON envelope.
// We use ReadableStream + TextDecoder to handle the chunked response.
async function* callServerStream(method, body, signal) {
  const jsonStr = JSON.stringify(body);
  const payload = new TextEncoder().encode(jsonStr);
  const envelope = new Uint8Array(5 + payload.length);
  envelope[0] = 0; // flag: 0 (data)
  envelope[1] = (payload.length >> 24) & 0xff;
  envelope[2] = (payload.length >> 16) & 0xff;
  envelope[3] = (payload.length >> 8) & 0xff;
  envelope[4] = payload.length & 0xff;
  envelope.set(payload, 5);

  const res = await fetch(`${BASE}${SVC}/${method}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/connect+json',
      'Connect-Protocol-Version': '1'
    },
    body: envelope,
    signal
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status}: ${text}`);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = new Uint8Array(0);

  function concat(a, b) {
    const c = new Uint8Array(a.length + b.length);
    c.set(a); c.set(b, a.length);
    return c;
  }

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buf = concat(buf, value);

    // Each Connect streaming frame: 5-byte header + payload
    // Byte 0: flags (0=data, 2=end-of-stream)
    // Bytes 1-4: big-endian uint32 length
    while (buf.length >= 5) {
      const flags = buf[0];
      const length = (buf[1] << 24) | (buf[2] << 16) | (buf[3] << 8) | buf[4];
      if (buf.length < 5 + length) break;

      const payload = buf.slice(5, 5 + length);
      buf = buf.slice(5 + length);

      const text = decoder.decode(payload);

      if (flags & 0x02) {
        // End-of-stream trailer — may contain error
        if (text && text.length > 2) {
          const trailer = JSON.parse(text);
          if (trailer.error) throw new Error(trailer.error.message || 'stream error');
        }
        return;
      }

      if (text.trim()) {
        yield JSON.parse(text);
      }
    }
  }
}

// ── State ─────────────────────────────────────────────────────
let streamController = null; // AbortController for active stream
let cpuHistory = [];
const MAX_HISTORY = 50;
let echoCount = 0;

// ── DOM ───────────────────────────────────────────────────────
const statusDot     = document.getElementById('status-dot');
const statusText    = document.getElementById('status-text');

const srvStatus     = document.getElementById('srv-status');
const srvUptime     = document.getElementById('srv-uptime');
const srvCpuCount   = document.getElementById('srv-cpu-count');
const srvGoVersion  = document.getElementById('srv-go-version');
const srvTimestamp  = document.getElementById('srv-timestamp');
const srvVersion    = document.getElementById('srv-version');
const getStatusBtn  = document.getElementById('get-status-btn');

const cpuCanvas       = document.getElementById('cpu-canvas');
const cpuNow          = document.getElementById('cpu-now');
const goroutines      = document.getElementById('goroutines');
const memUsed         = document.getElementById('mem-used');
const memTotal        = document.getElementById('mem-total');
const streamInterval  = document.getElementById('stream-interval');
const intervalDisplay = document.getElementById('stream-interval-display');
const startStreamBtn  = document.getElementById('start-stream-btn');
const stopStreamBtn   = document.getElementById('stop-stream-btn');
const livePill        = document.getElementById('live-pill');

const echoInput  = document.getElementById('echo-input');
const echoBtn    = document.getElementById('echo-btn');
const echoTerminal = document.getElementById('echo-terminal');
const echoCountEl  = document.getElementById('echo-count');
const echoRtt      = document.getElementById('echo-rtt');

const logOutput  = document.getElementById('log-output');
const clearLogBtn = document.getElementById('clear-log-btn');

// ── Canvas ────────────────────────────────────────────────────
const ctx = cpuCanvas.getContext('2d');

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

  // Grid lines
  const gridLines = 4;
  for (let i = 0; i <= gridLines; i++) {
    const y = (H / gridLines) * i;
    ctx.strokeStyle = 'rgba(255,255,255,0.04)';
    ctx.lineWidth = 1;
    ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(W, y); ctx.stroke();
    ctx.fillStyle = 'rgba(255,255,255,0.18)';
    ctx.font = '9px "JetBrains Mono", monospace';
    ctx.fillText(`${100 - (100 / gridLines) * i}%`, 6, y < 10 ? 12 : y - 3);
  }

  if (cpuHistory.length < 2) return;

  const pL = 36, pR = 10, pT = 8, pB = 8;
  const cW = W - pL - pR, cH = H - pT - pB;
  const step = cW / (MAX_HISTORY - 1);
  const coord = (i) => ({
    x: pL + i * step,
    y: pT + cH - (cpuHistory[i] / 100) * cH
  });

  // Fill
  ctx.beginPath();
  const s0 = coord(0);
  ctx.moveTo(s0.x, pT + cH); ctx.lineTo(s0.x, s0.y);
  for (let i = 1; i < cpuHistory.length; i++) { const p = coord(i); ctx.lineTo(p.x, p.y); }
  const last = coord(cpuHistory.length - 1);
  ctx.lineTo(last.x, pT + cH); ctx.closePath();
  const ag = ctx.createLinearGradient(0, pT, 0, pT + cH);
  ag.addColorStop(0, 'rgba(16,185,129,0.25)');
  ag.addColorStop(1, 'rgba(16,185,129,0)');
  ctx.fillStyle = ag; ctx.fill();

  // Stroke
  ctx.beginPath(); ctx.moveTo(s0.x, s0.y);
  for (let i = 1; i < cpuHistory.length; i++) { const p = coord(i); ctx.lineTo(p.x, p.y); }
  const lg = ctx.createLinearGradient(pL, 0, W - pR, 0);
  lg.addColorStop(0, '#8b5cf6');
  lg.addColorStop(1, '#10b981');
  ctx.strokeStyle = lg; ctx.lineWidth = 2.5;
  ctx.shadowColor = 'rgba(16,185,129,0.4)'; ctx.shadowBlur = 10; ctx.stroke();
  ctx.shadowBlur = 0;

  // Dot
  ctx.beginPath(); ctx.arc(last.x, last.y, 5, 0, Math.PI * 2);
  ctx.fillStyle = '#10b981'; ctx.fill();
  ctx.beginPath(); ctx.arc(last.x, last.y, 9, 0, Math.PI * 2);
  ctx.strokeStyle = 'rgba(16,185,129,0.3)'; ctx.lineWidth = 1; ctx.stroke();
}

// ── Logging ───────────────────────────────────────────────────
function log(type, msg) {
  const line = document.createElement('div');
  line.className = `log-line log-${type}`;
  line.textContent = `[${new Date().toLocaleTimeString()}] ${msg}`;
  logOutput.appendChild(line);
  logOutput.scrollTop = logOutput.scrollHeight;
  while (logOutput.children.length > 100) logOutput.removeChild(logOutput.firstChild);
}

// ── GetStatus ─────────────────────────────────────────────────
async function doGetStatus() {
  getStatusBtn.disabled = true;
  log('send', 'RPC → GetStatus (unary)');
  try {
    const data = await callUnary('GetStatus', {});
    srvStatus.textContent    = data.status    || '—';
    srvUptime.textContent    = formatUptime(parseFloat(data.uptimeS || 0));
    srvCpuCount.textContent  = data.cpuCount  || '—';
    srvGoVersion.textContent = data.goVersion || '—';
    srvTimestamp.textContent = formatTs(data.timestamp);
    srvVersion.textContent   = data.version   || '—';
    log('recv', `GetStatus ← status=${data.status}, uptime=${data.uptimeS?.toFixed(1)}s, cores=${data.cpuCount}`);
  } catch (err) {
    log('error', `GetStatus failed: ${err.message}`);
  } finally {
    getStatusBtn.disabled = false;
  }
}

// ── StreamCPU ─────────────────────────────────────────────────
function setStreamState(active) {
  startStreamBtn.disabled = active;
  stopStreamBtn.disabled  = !active;
  livePill.style.display  = active ? 'flex' : 'none';
  statusDot.className     = `status-dot ${active ? 'streaming' : 'idle'}`;
  statusText.textContent  = active ? 'Streaming' : 'Idle';
}

async function startStream() {
  if (streamController) streamController.abort();

  const intervalMs = parseInt(streamInterval.value, 10);
  streamController = new AbortController();
  setStreamState(true);
  cpuHistory = [];

  log('send', `RPC → StreamCPU (server-streaming), interval=${intervalMs}ms`);

  try {
    for await (const sample of callServerStream('StreamCPU', { intervalMs }, streamController.signal)) {
      const cpu = parseFloat(sample.cpuPercent || 0);
      cpuHistory.push(cpu);
      if (cpuHistory.length > MAX_HISTORY) cpuHistory.shift();

      cpuNow.textContent    = `${cpu.toFixed(1)}%`;
      goroutines.textContent = sample.goroutines || '—';
      memUsed.textContent   = `${parseFloat(sample.memUsedMb  || 0).toFixed(1)} MB`;
      memTotal.textContent  = `${parseFloat(sample.memTotalMb || 0).toFixed(1)} MB`;
      drawChart();
    }
    log('stream', 'StreamCPU ended (server closed stream)');
  } catch (err) {
    if (err.name === 'AbortError') {
      log('stream', 'StreamCPU cancelled by client');
    } else {
      log('error', `StreamCPU error: ${err.message}`);
    }
  } finally {
    setStreamState(false);
    streamController = null;
  }
}

function stopStream() {
  if (streamController) {
    streamController.abort();
    log('send', 'Cancelled StreamCPU');
  }
}

// ── Echo ──────────────────────────────────────────────────────
async function doEcho() {
  const msg = echoInput.value.trim();
  if (!msg) return;

  echoBtn.disabled = true;
  const t0 = Date.now();
  log('send', `RPC → Echo: "${msg}"`);

  const placeholder = echoTerminal.querySelector('.echo-placeholder');
  if (placeholder) placeholder.remove();

  const sentEl = document.createElement('div');
  sentEl.className = 'echo-entry';
  sentEl.innerHTML = `<span class="echo-sent">▶ ${escHtml(msg)}</span>`;
  echoTerminal.appendChild(sentEl);

  try {
    const data = await callUnary('Echo', { message: msg });
    const rtt = Date.now() - t0;
    const recvEl = document.createElement('div');
    recvEl.className = 'echo-entry';
    recvEl.innerHTML =
      `<span class="echo-recv">◀ ${escHtml(data.message)}</span>` +
      `<span class="echo-ts">${rtt}ms | len=${data.length}</span>`;
    echoTerminal.appendChild(recvEl);
    echoTerminal.scrollTop = echoTerminal.scrollHeight;

    echoCount++;
    echoCountEl.textContent = echoCount;
    echoRtt.textContent = `${rtt} ms`;
    log('recv', `Echo ← "${data.message}" (${rtt}ms, len=${data.length})`);
    echoInput.value = '';
  } catch (err) {
    log('error', `Echo failed: ${err.message}`);
  } finally {
    echoBtn.disabled = false;
  }
}

// ── Utilities ─────────────────────────────────────────────────
function formatUptime(s) {
  s = Math.floor(s);
  return `${Math.floor(s/3600)}h ${Math.floor((s%3600)/60)}m ${s%60}s`;
}

function formatTs(ts) {
  if (!ts) return '—';
  return new Date(ts).toLocaleTimeString();
}

function escHtml(s) {
  return String(s)
    .replace(/&/g,'&amp;').replace(/</g,'&lt;')
    .replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function updateSliderTrack() {
  const min = parseFloat(streamInterval.min);
  const max = parseFloat(streamInterval.max);
  const val = parseFloat(streamInterval.value);
  streamInterval.style.setProperty('--pct', `${((val - min) / (max - min)) * 100}%`);
}

// ── Event Listeners ───────────────────────────────────────────
getStatusBtn.addEventListener('click', doGetStatus);
startStreamBtn.addEventListener('click', startStream);
stopStreamBtn.addEventListener('click', stopStream);
echoBtn.addEventListener('click', doEcho);
echoInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') echoBtn.click(); });
streamInterval.addEventListener('input', () => {
  intervalDisplay.textContent = streamInterval.value;
  updateSliderTrack();
});
clearLogBtn.addEventListener('click', () => {
  logOutput.innerHTML = '';
  log('system', 'Console cleared.');
});
window.addEventListener('resize', resizeCanvas);

// ── Init ──────────────────────────────────────────────────────
window.addEventListener('load', () => {
  updateSliderTrack();
  setTimeout(() => {
    resizeCanvas();
    doGetStatus();
  }, 150);
});
