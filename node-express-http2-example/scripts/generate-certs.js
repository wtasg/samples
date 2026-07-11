const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const certsDir = path.join(__dirname, '..', 'certs');
const keyPath = path.join(certsDir, 'key.pem');
const certPath = path.join(certsDir, 'cert.pem');

console.log('Building: Generating self-signed SSL/TLS certificates for HTTP/2 example...');

if (!fs.existsSync(certsDir)) {
  fs.mkdirSync(certsDir, { recursive: true });
}

try {
  const cmd = `openssl req -x509 -newkey rsa:2048 -keyout "${keyPath}" -out "${certPath}" -sha256 -days 365 -nodes -subj "/CN=localhost"`;
  execSync(cmd, { stdio: 'inherit' });
  console.log('SSL certificates successfully generated in:', certsDir);
} catch (error) {
  console.error('Failed to generate SSL certificates:', error.message);
  process.exit(1);
}
