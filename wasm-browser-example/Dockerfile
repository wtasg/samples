FROM node:24-slim AS builder

# Install Rust + wasm-pack
RUN apt-get update && apt-get install -y curl build-essential && rm -rf /var/lib/apt/lists/*
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --default-toolchain stable
ENV PATH="/root/.cargo/bin:${PATH}"
RUN rustup target add wasm32-unknown-unknown && cargo install wasm-pack

WORKDIR /app

# Copy everything
COPY . .

# Build WASM
RUN npm install && npm run build

# ── Runtime stage ─────────────────────────────────────────────
FROM node:24-slim

WORKDIR /app
USER node

COPY --chown=node:node package*.json ./
RUN npm install --production

# Copy built artifacts
COPY --chown=node:node --from=builder /app/server.js ./
COPY --chown=node:node --from=builder /app/public ./public

EXPOSE 60010

CMD ["npm", "start"]
