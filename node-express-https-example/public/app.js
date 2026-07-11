// State variables
let cpuHistory = [];
const maxHistorySize = 30;
let rateLimitRemaining = 15;
let rateLimitMax = 15;

// DOM Elements
const connectionStatus = document.getElementById('connection-status');
const statusDot = connectionStatus.querySelector('.status-dot');
const statusText = connectionStatus.querySelector('.status-text');

const heartbeatStatus = document.getElementById('heartbeat-status');
const serverUptime = document.getElementById('server-uptime');
const lastPingTime = document.getElementById('last-ping-time');
const pingBtn = document.getElementById('ping-btn');

const cpuGauge = document.getElementById('cpu-gauge');
const cpuQueryVal = document.getElementById('cpu-query-val');
const rateLimitProgress = document.getElementById('rate-limit-progress');
const rateLimitText = document.getElementById('rate-limit-text');
const rateLimitBadge = document.getElementById('rate-limit-badge');
const queryCpuBtn = document.getElementById('query-cpu-btn');
const errorToast = document.getElementById('error-toast');
const errorMsg = document.getElementById('error-msg');

const cpuChart = document.getElementById('cpu-chart');
const ctx = cpuChart.getContext('2d');
const streamCpuVal = document.getElementById('stream-cpu-val');
const streamTimestamp = document.getElementById('stream-timestamp');

const consoleOutput = document.getElementById('console-output');
const clearLogBtn = document.getElementById('clear-log-btn');

// --- Log Utility ---
function logEvent(type, message) {
  const line = document.createElement('div');
  line.className = `console-line ${type}-line`;
  
  const timestamp = new Date().toLocaleTimeString();
  line.textContent = `[${timestamp}] ${message}`;
  
  consoleOutput.appendChild(line);
  consoleOutput.scrollTop = consoleOutput.scrollHeight;
  
  // Keep logs at a reasonable size
  while (consoleOutput.children.length > 50) {
    consoleOutput.removeChild(consoleOutput.firstChild);
  }
}

// --- Connection Status Updater ---
function setConnectedState(isConnected, text = 'Connected') {
  if (isConnected) {
    statusDot.className = 'status-dot connected';
    statusText.textContent = text;
  } else {
    statusDot.className = 'status-dot disconnected';
    statusText.textContent = text;
  }
}

// --- Heartbeat Client ---
async function checkHeartbeat(isManual = false) {
  const modeText = isManual ? 'Manual heartbeat check...' : 'Periodic heartbeat ping...';
  logEvent('system', modeText);
  logEvent('get', 'GET /heartbeat');

  const startTime = Date.now();
  try {
    const response = await fetch('/heartbeat');
    const latency = Date.now() - startTime;
    
    // Track rate limits on heartbeat too
    updateRateLimitInfo(response.headers);

    if (response.ok) {
      const data = await response.json();
      heartbeatStatus.textContent = 'Active';
      heartbeatStatus.className = 'value-data text-green';
      
      // Calculate human-friendly uptime
      const uptimeSec = Math.floor(data.uptime);
      const hours = Math.floor(uptimeSec / 3600);
      const minutes = Math.floor((uptimeSec % 3600) / 60);
      const seconds = uptimeSec % 60;
      serverUptime.textContent = `${hours}h ${minutes}m ${seconds}s`;
      
      lastPingTime.textContent = `${latency}ms ago`;
      logEvent('response', `200 OK (${latency}ms) - Status: ${data.status}, Uptime: ${uptimeSec}s`);
      setConnectedState(true, 'Secure Connection Active');
    } else {
      throw new Error(`HTTP ${response.status}`);
    }
  } catch (error) {
    heartbeatStatus.textContent = 'Offline';
    heartbeatStatus.className = 'value-data text-red';
    logEvent('error', `Heartbeat failed: ${error.message}`);
    setConnectedState(false, 'Disconnected');
  }
}

// --- Rate Limit UI Updater ---
function updateRateLimitInfo(headers) {
  const remaining = headers.get('ratelimit-remaining');
  const limit = headers.get('ratelimit-limit');

  if (remaining !== null && limit !== null) {
    rateLimitRemaining = parseInt(remaining, 10);
    rateLimitMax = parseInt(limit, 10);

    const percentage = (rateLimitRemaining / rateLimitMax) * 100;
    rateLimitProgress.style.width = `${percentage}%`;
    rateLimitText.textContent = `${rateLimitRemaining} / ${rateLimitMax} requests available`;

    // Highlight low limits
    rateLimitBadge.classList.remove('low', 'empty');
    if (rateLimitRemaining === 0) {
      rateLimitBadge.classList.add('empty');
    } else if (rateLimitRemaining < 5) {
      rateLimitBadge.classList.add('low');
    }
  }
}

// --- Manual CPU Query ---
async function queryCpuLoad() {
  logEvent('get', 'GET /api/cpu/load');
  queryCpuBtn.disabled = true;

  try {
    errorToast.classList.add('hide');
    const response = await fetch('/api/cpu/load');
    
    updateRateLimitInfo(response.headers);

    if (response.ok) {
      const data = await response.json();
      const cpu = data.cpu;
      
      // Update Gauge UI
      cpuGauge.style.setProperty('--percent', cpu);
      cpuQueryVal.textContent = `${cpu}%`;
      
      // Set color based on usage
      let gaugeColor = 'var(--accent-secondary)'; // Cyan (low)
      if (cpu > 70) {
        gaugeColor = 'var(--accent-red)'; // Red (high)
      } else if (cpu > 30) {
        gaugeColor = 'var(--accent-orange)'; // Orange (mid)
      }
      cpuGauge.style.setProperty('--gauge-color', gaugeColor);

      logEvent('response', `200 OK - CPU Load retrieved: ${cpu}%`);
    } else if (response.status === 429) {
      const errorData = await response.json();
      logEvent('error', `429 Too Many Requests - ${errorData.message}`);
      
      errorMsg.textContent = errorData.message || 'Too many requests. Please try again later.';
      errorToast.classList.remove('hide');
    } else {
      throw new Error(`HTTP ${response.status}`);
    }
  } catch (error) {
    logEvent('error', `Failed to query CPU: ${error.message}`);
  } finally {
    queryCpuBtn.disabled = false;
  }
}

// --- SSE Event Stream ---
function initEventStream() {
  logEvent('system', 'Opening Server-Sent Events stream...');
  const eventSource = new EventSource('/api/cpu/stream');

  eventSource.onopen = () => {
    logEvent('sse', 'SSE stream connection established.');
    setConnectedState(true, 'SSE Stream Active');
  };

  eventSource.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      const cpu = parseFloat(data.cpu);
      
      streamCpuVal.textContent = `${cpu.toFixed(2)}%`;
      
      // Parse timestamp
      const date = new Date(data.timestamp);
      streamTimestamp.textContent = `Updated: ${date.toLocaleTimeString()}`;

      // Update Chart data
      cpuHistory.push(cpu);
      if (cpuHistory.length > maxHistorySize) {
        cpuHistory.shift();
      }

      drawChart();
    } catch (err) {
      console.error('Error parsing SSE data:', err);
    }
  };

  eventSource.onerror = (error) => {
    logEvent('error', 'SSE stream error occurred. Reconnecting...');
    setConnectedState(false, 'SSE Reconnecting...');
  };
}

// --- Canvas Chart Drawing ---
function drawChart() {
  const width = cpuChart.width;
  const height = cpuChart.height;
  
  // Clear canvas
  ctx.clearRect(0, 0, width, height);

  // Draw Grid Lines
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.03)';
  ctx.lineWidth = 1;
  const gridLines = 4;
  for (let i = 0; i <= gridLines; i++) {
    const y = (height / gridLines) * i;
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(width, y);
    ctx.stroke();

    // Draw grid percentages
    ctx.fillStyle = 'rgba(255, 255, 255, 0.2)';
    ctx.font = '9px Outfit';
    ctx.fillText(`${100 - (100 / gridLines) * i}%`, 8, y - 4);
  }

  if (cpuHistory.length < 2) return;

  // Chart configuration
  const paddingLeft = 40;
  const paddingRight = 10;
  const paddingTop = 10;
  const paddingBottom = 10;
  
  const chartWidth = width - paddingLeft - paddingRight;
  const chartHeight = height - paddingTop - paddingBottom;
  
  const pointsCount = cpuHistory.length;
  const stepX = chartWidth / (maxHistorySize - 1);

  // Function to map data point to coordinates
  const getCoords = (index) => {
    const val = cpuHistory[index];
    const x = paddingLeft + index * stepX;
    const y = paddingTop + chartHeight - (val / 100) * chartHeight;
    return { x, y };
  };

  // Draw gradient area below line
  ctx.beginPath();
  const startPt = getCoords(0);
  ctx.moveTo(startPt.x, paddingTop + chartHeight);
  ctx.lineTo(startPt.x, startPt.y);

  for (let i = 1; i < pointsCount; i++) {
    const pt = getCoords(i);
    ctx.lineTo(pt.x, pt.y);
  }
  ctx.lineTo(getCoords(pointsCount - 1).x, paddingTop + chartHeight);
  ctx.closePath();

  const areaGradient = ctx.createLinearGradient(0, paddingTop, 0, paddingTop + chartHeight);
  areaGradient.addColorStop(0, 'rgba(0, 210, 255, 0.2)');
  areaGradient.addColorStop(1, 'rgba(0, 210, 255, 0.0)');
  ctx.fillStyle = areaGradient;
  ctx.fill();

  // Draw main stroke line
  ctx.beginPath();
  ctx.moveTo(startPt.x, startPt.y);
  for (let i = 1; i < pointsCount; i++) {
    const pt = getCoords(i);
    ctx.lineTo(pt.x, pt.y);
  }

  const lineGradient = ctx.createLinearGradient(paddingLeft, 0, width - paddingRight, 0);
  lineGradient.addColorStop(0, 'var(--accent-primary)');
  lineGradient.addColorStop(1, 'var(--accent-secondary)');

  ctx.strokeStyle = lineGradient;
  ctx.lineWidth = 3;
  ctx.shadowColor = 'rgba(0, 210, 255, 0.4)';
  ctx.shadowBlur = 8;
  ctx.stroke();
  
  // Reset shadow for subsequent drawings
  ctx.shadowBlur = 0;

  // Draw pulse dot at the last point
  const lastPt = getCoords(pointsCount - 1);
  ctx.beginPath();
  ctx.arc(lastPt.x, lastPt.y, 5, 0, 2 * Math.PI);
  ctx.fillStyle = 'var(--accent-secondary)';
  ctx.fill();
  
  ctx.beginPath();
  ctx.arc(lastPt.x, lastPt.y, 8, 0, 2 * Math.PI);
  ctx.strokeStyle = 'rgba(0, 210, 255, 0.4)';
  ctx.lineWidth = 1;
  ctx.stroke();
}

// --- Event Listeners ---
pingBtn.addEventListener('click', () => checkHeartbeat(true));
queryCpuBtn.addEventListener('click', queryCpuLoad);
clearLogBtn.addEventListener('click', () => {
  consoleOutput.innerHTML = '';
  logEvent('system', 'Console cleared.');
});

// Setup resizing for Canvas
function resizeCanvas() {
  // Handle layout scaling if container scales
  const dpr = window.devicePixelRatio || 1;
  const rect = cpuChart.getBoundingClientRect();
  cpuChart.width = rect.width * dpr;
  cpuChart.height = rect.height * dpr;
  ctx.scale(dpr, dpr);
  drawChart();
}

// Initialize on Load
window.addEventListener('resize', resizeCanvas);

// Wait slightly to trigger layout computations
setTimeout(() => {
  resizeCanvas();
  // 1. Initial Heartbeat Check
  checkHeartbeat();
  // 2. Open Server-Sent Events stream
  initEventStream();
  // 3. Set interval for heartbeat checks (every 10 seconds)
  setInterval(() => checkHeartbeat(false), 10000);
}, 200);
