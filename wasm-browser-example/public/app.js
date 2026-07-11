// ============================================================
// WASM Browser Example — app.js
// Loads the Rust WASM module and drives all UI interactions:
//   - Mandelbrot renderer (mandelbrot)
//   - Fibonacci race (fibonacci)
//   - Prime sieve benchmark (count_primes)
//   - FNV-1a hash terminal (fnv1a_hash)
// ============================================================
'use strict';

// ── Load the WASM module ──────────────────────────────────────
// wasm-bindgen --target web generates an ES module in public/pkg/
import init, {
  fibonacci,
  mandelbrot,
  fnv1a_hash,
  count_primes
} from './pkg/wasm_monitor.js';

// ── DOM ───────────────────────────────────────────────────────
const wasmLoading = document.getElementById('wasm-loading');
const appEl       = document.getElementById('app');

// Mandelbrot
const mandelbrotCanvas = document.getElementById('mandelbrot-canvas');
const renderOverlay    = document.getElementById('render-overlay');
const iterSlider       = document.getElementById('iter-slider');
const iterDisplay      = document.getElementById('iter-display');
const renderBtn        = document.getElementById('render-btn');
const renderStats      = document.getElementById('render-stats');

// Fibonacci race
const fibNSlider       = document.getElementById('fib-n-slider');
const fibNDisplay      = document.getElementById('fib-n-display');
const raceBtn          = document.getElementById('race-btn');
const jsBar            = document.getElementById('js-bar');
const wasmBar          = document.getElementById('wasm-bar');
const jsTime           = document.getElementById('js-time');
const wasmTime         = document.getElementById('wasm-time');
const speedupBadge     = document.getElementById('speedup-badge');
const fibResult        = document.getElementById('fib-result');

// Prime sieve
const primeNSlider     = document.getElementById('prime-n-slider');
const primeNDisplay    = document.getElementById('prime-n-display');
const primeBtn         = document.getElementById('prime-btn');
const primeJsBar       = document.getElementById('prime-js-bar');
const primeWasmBar     = document.getElementById('prime-wasm-bar');
const primeJsTime      = document.getElementById('prime-js-time');
const primeWasmTime    = document.getElementById('prime-wasm-time');
const primeSpeedupBadge = document.getElementById('prime-speedup-badge');
const primeResult      = document.getElementById('prime-result');

// Hash
const hashInput        = document.getElementById('hash-input');
const hashBtn          = document.getElementById('hash-btn');
const hashTerminal     = document.getElementById('hash-terminal');
const hashLiveValue    = document.getElementById('hash-live-value');

// ── Mandelbrot ────────────────────────────────────────────────
const ctx = mandelbrotCanvas.getContext('2d');

async function renderMandelbrot() {
  const maxIter = parseInt(iterSlider.value, 10);
  const W = mandelbrotCanvas.width;
  const H = mandelbrotCanvas.height;

  renderOverlay.classList.add('active');
  renderBtn.disabled = true;

  // Yield to browser for repaint
  await new Promise(r => setTimeout(r, 50));

  const t0 = performance.now();
  const pixels = mandelbrot(W, H, maxIter);
  const t1 = performance.now();

  const imageData = ctx.createImageData(W, H);
  imageData.data.set(pixels);
  ctx.putImageData(imageData, 0, 0);

  const elapsed = (t1 - t0).toFixed(1);
  renderStats.textContent =
    `Rendered ${W}×${H} = ${(W*H).toLocaleString()} pixels in ${elapsed}ms ` +
    `(${maxIter} iterations) — ${((W * H) / (t1 - t0) * 1000 / 1e6).toFixed(1)} Mpx/s`;

  renderOverlay.classList.remove('active');
  renderBtn.disabled = false;
}

// ── Fibonacci race ────────────────────────────────────────────
// JS implementation to compare against
function fibJS(n) {
  if (n <= 1) return n;
  let a = 0, b = 1;
  for (let i = 2; i <= n; i++) { const t = a + b; a = b; b = t; }
  return b;
}

async function runFibRace() {
  const n = parseInt(fibNSlider.value, 10);
  const CALLS = 10_000;
  raceBtn.disabled = true;
  speedupBadge.style.display = 'none';
  jsBar.style.width = '0%';
  wasmBar.style.width = '0%';
  jsTime.textContent = '…';
  wasmTime.textContent = '…';
  fibResult.textContent = '';

  await new Promise(r => setTimeout(r, 30));

  // JS benchmark
  const t0js = performance.now();
  let jsVal;
  for (let i = 0; i < CALLS; i++) { jsVal = fibJS(n); }
  const jsMs = performance.now() - t0js;

  await new Promise(r => setTimeout(r, 10));

  // WASM benchmark
  const t0w = performance.now();
  let wasmVal;
  for (let i = 0; i < CALLS; i++) { wasmVal = fibonacci(n); }
  const wasmMs = performance.now() - t0w;

  const maxMs = Math.max(jsMs, wasmMs);
  jsBar.style.width   = `${(jsMs / maxMs) * 100}%`;
  wasmBar.style.width = `${(wasmMs / maxMs) * 100}%`;
  jsTime.textContent   = `${jsMs.toFixed(1)} ms`;
  wasmTime.textContent = `${wasmMs.toFixed(1)} ms`;

  const ratio = jsMs / wasmMs;
  if (ratio >= 1.05) {
    speedupBadge.textContent = `🦀 WASM is ${ratio.toFixed(1)}× faster`;
    speedupBadge.style.display = 'flex';
  } else if (ratio < 0.95) {
    speedupBadge.textContent = `JS is ${(1/ratio).toFixed(1)}× faster (JIT wins here!)`;
    speedupBadge.style.display = 'flex';
  } else {
    speedupBadge.textContent = 'Roughly equal performance';
    speedupBadge.style.display = 'flex';
  }

  fibResult.textContent =
    `fibonacci(${n}) = ${wasmVal.toString()} — verified with JS (${CALLS.toLocaleString()} calls each)`;

  raceBtn.disabled = false;
}

// ── Prime sieve benchmark ─────────────────────────────────────
// JS Sieve of Eratosthenes
function countPrimesJS(n) {
  if (n < 2) return 0;
  const sieve = new Uint8Array(n + 1).fill(1);
  sieve[0] = sieve[1] = 0;
  for (let i = 2; i * i <= n; i++) {
    if (sieve[i]) {
      for (let j = i * i; j <= n; j += i) sieve[j] = 0;
    }
  }
  let count = 0;
  for (let i = 2; i <= n; i++) { if (sieve[i]) count++; }
  return count;
}

async function runPrimeBenchmark() {
  const n = parseInt(primeNSlider.value, 10);
  primeBtn.disabled = true;
  primeSpeedupBadge.style.display = 'none';
  primeJsBar.style.width = '0%';
  primeWasmBar.style.width = '0%';
  primeJsTime.textContent = '…';
  primeWasmTime.textContent = '…';
  primeResult.textContent = '';

  await new Promise(r => setTimeout(r, 30));

  const t0js = performance.now();
  const jsCount = countPrimesJS(n);
  const jsMs = performance.now() - t0js;

  await new Promise(r => setTimeout(r, 10));

  const t0w = performance.now();
  const wasmCount = count_primes(n);
  const wasmMs = performance.now() - t0w;

  const maxMs = Math.max(jsMs, wasmMs);
  primeJsBar.style.width   = `${(jsMs / maxMs) * 100}%`;
  primeWasmBar.style.width = `${(wasmMs / maxMs) * 100}%`;
  primeJsTime.textContent   = `${jsMs.toFixed(1)} ms`;
  primeWasmTime.textContent = `${wasmMs.toFixed(1)} ms`;

  const ratio = jsMs / wasmMs;
  if (ratio >= 1.05) {
    primeSpeedupBadge.textContent = `🦀 WASM is ${ratio.toFixed(1)}× faster`;
  } else if (ratio < 0.95) {
    primeSpeedupBadge.textContent = `JS is ${(1/ratio).toFixed(1)}× faster (typed arrays are fast!)`;
  } else {
    primeSpeedupBadge.textContent = 'Roughly equal performance';
  }
  primeSpeedupBadge.style.display = 'flex';

  primeResult.textContent =
    `Primes up to ${n.toLocaleString()}: ${wasmCount.toLocaleString()} ` +
    `(JS=${jsCount.toLocaleString()} — match: ${jsCount === wasmCount ? '✓' : '✗'})`;

  primeBtn.disabled = false;
}

// ── FNV Hash ─────────────────────────────────────────────────
function doHash(text) {
  if (!text) { hashLiveValue.textContent = '—'; return; }
  const hash = fnv1a_hash(text);
  hashLiveValue.textContent = `0x${BigInt(hash).toString(16).padStart(16, '0')} (${hash})`;
  return hash;
}

function addHashEntry(text) {
  const placeholder = hashTerminal.querySelector('.hash-placeholder');
  if (placeholder) placeholder.remove();

  const hash = doHash(text);
  const entry = document.createElement('div');
  entry.className = 'hash-entry';
  const ts = new Date().toLocaleTimeString();
  entry.innerHTML =
    `<span class="hash-in">▶ ${escHtml(text)}</span><br>` +
    `<span class="hash-out">◀ ${hash}</span>` +
    `<span class="hash-ts">${ts}</span>`;
  hashTerminal.appendChild(entry);
  hashTerminal.scrollTop = hashTerminal.scrollHeight;
}

// ── Slider track helpers ──────────────────────────────────────
function updateSlider(el) {
  const min = parseFloat(el.min), max = parseFloat(el.max), val = parseFloat(el.value);
  el.style.setProperty('--pct', `${((val - min) / (max - min)) * 100}%`);
}

function escHtml(s) {
  return String(s)
    .replace(/&/g,'&amp;').replace(/</g,'&lt;')
    .replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function formatN(n) {
  return parseInt(n).toLocaleString();
}

// ── Event Listeners ───────────────────────────────────────────
iterSlider.addEventListener('input', () => {
  iterDisplay.textContent = iterSlider.value;
  updateSlider(iterSlider);
});
renderBtn.addEventListener('click', renderMandelbrot);

fibNSlider.addEventListener('input', () => {
  fibNDisplay.textContent = fibNSlider.value;
  updateSlider(fibNSlider);
});
raceBtn.addEventListener('click', runFibRace);

primeNSlider.addEventListener('input', () => {
  primeNDisplay.textContent = formatN(primeNSlider.value);
  updateSlider(primeNSlider);
});
primeBtn.addEventListener('click', runPrimeBenchmark);

hashInput.addEventListener('input', () => doHash(hashInput.value));
hashBtn.addEventListener('click', () => {
  const text = hashInput.value.trim();
  if (text) { addHashEntry(text); hashInput.value = ''; hashLiveValue.textContent = '—'; }
});
hashInput.addEventListener('keydown', (e) => { if (e.key === 'Enter') hashBtn.click(); });

// ── Init ──────────────────────────────────────────────────────
async function main() {
  try {
    await init('./pkg/wasm_monitor_bg.wasm');

    // Hide loading overlay, show app
    wasmLoading.style.display = 'none';
    appEl.style.display = 'block';

    // Initialise slider tracks
    updateSlider(iterSlider);
    updateSlider(fibNSlider);
    updateSlider(primeNSlider);

    // Render initial Mandelbrot
    await renderMandelbrot();

  } catch (err) {
    wasmLoading.querySelector('.loader-title').textContent = 'Failed to load WASM';
    wasmLoading.querySelector('.loader-sub').textContent = err.message;
    console.error('WASM init error:', err);
  }
}

main();
