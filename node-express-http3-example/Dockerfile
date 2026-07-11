FROM node:24-slim

# Install openssl for certificate generation
RUN apt-get update && apt-get install -y openssl && rm -rf /var/lib/apt/lists/*

# Set up working directory owned by node user
RUN mkdir -p /app && chown -R node:node /app

WORKDIR /app

USER node

COPY --chown=node:node package*.json ./
RUN npm install

COPY --chown=node:node . .

# Generate SSL certs before start
RUN npm run build

# Expose HTTP/3 UDP and HTTPS TCP
EXPOSE 60007/udp 60447/tcp

CMD ["npm", "start"]
