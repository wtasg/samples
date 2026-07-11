FROM node:24-slim

# Install apache benchmark (ab) and h2load
RUN apt-get update && apt-get install -y \
    apache2-utils \
    nghttp2-client \
    && rm -rf /var/lib/apt/lists/*

# Set up working directory owned by node user
RUN mkdir -p /app && chown -R node:node /app

WORKDIR /app

USER node

COPY --chown=node:node package*.json ./
RUN npm install

COPY --chown=node:node . .

ENTRYPOINT ["node", "benchmark.js"]
