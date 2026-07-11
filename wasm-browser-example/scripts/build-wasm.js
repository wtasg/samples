#!/usr/bin/env node
/**
 * build-wasm.js
 * Builds the Rust WASM crate using wasm-pack and copies the output
 * into public/pkg/ for serving from the web server.
 */
const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

const wasmDir = path.join(__dirname, '..', 'wasm');
const outDir  = path.join(__dirname, '..', 'public', 'pkg');
const pkgDir  = path.join(wasmDir, 'pkg');

// Ensure wasm-pack is available
function findWasmPack() {
  const candidates = [
    'wasm-pack',
    path.join(process.env.HOME || '', '.cargo', 'bin', 'wasm-pack')
  ];
  for (const bin of candidates) {
    try {
      execSync(`${bin} --version`, { stdio: 'ignore' });
      return bin;
    } catch { /* not found */ }
  }
  throw new Error(
    'wasm-pack not found. Install it with: cargo install wasm-pack\n' +
    'Then re-run: npm run build'
  );
}

console.log('[WASM Build] Building Rust crate with wasm-pack…');

const wasmPack = findWasmPack();
console.log(`[WASM Build] Using: ${wasmPack}`);

try {
  execSync(
    `${wasmPack} build --target web --out-dir pkg --release`,
    { cwd: wasmDir, stdio: 'inherit' }
  );
} catch (err) {
  console.error('[WASM Build] wasm-pack build failed:', err.message);
  process.exit(1);
}

// Copy pkg/ output to public/pkg/
if (fs.existsSync(outDir)) {
  fs.rmSync(outDir, { recursive: true });
}
fs.mkdirSync(outDir, { recursive: true });

const files = fs.readdirSync(pkgDir);
for (const file of files) {
  if (file.startsWith('.') || file === 'package.json') continue;
  fs.copyFileSync(path.join(pkgDir, file), path.join(outDir, file));
}

console.log(`[WASM Build] Output copied to: ${outDir}`);
console.log('[WASM Build] Done.');
