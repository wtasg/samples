#!/usr/bin/env bash
set -euo pipefail

# Colors for log statements
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Starting Docker Sidecar Example Verification ===${NC}"

# Ensure we clean up on exit
cleanup() {
  echo -e "\n${GREEN}=== Cleaning up Docker resources... ===${NC}"
  docker compose down -v
}
trap cleanup EXIT

# 1. Build and boot containers
echo -e "\n${GREEN}Booting up the Docker stack...${NC}"
docker compose up --build -d

# 2. Wait for services to be ready
wait_for_url() {
  local url=$1
  local name=$2
  local max_attempts=30
  local attempt=1
  echo -n "Waiting for $name to be ready at $url..."
  until curl -s -f "$url" > /dev/null || [ $attempt -eq $max_attempts ]; do
    echo -n "."
    sleep 1
    attempt=$((attempt + 1))
  done
  if [ $attempt -eq $max_attempts ]; then
    echo -e "\n${RED}Error: $name failed to start!${NC}"
    exit 1
  fi
  echo -e " [READY]"
}

wait_for_url "http://localhost:60015/proxy-health" "Nginx Proxy Sidecar"
wait_for_url "http://localhost:60015/" "Go Application (via Nginx)"
wait_for_url "http://localhost:60016/-/healthy" "Prometheus Sidecar"
wait_for_url "http://localhost:60017/api/health" "Grafana Observability Dashboard"

# 3. Simulate client requests
echo -e "\n${GREEN}Sending requests to populate metrics and logs...${NC}"
echo "Sending standard requests..."
curl -s -f http://localhost:60015/ > /dev/null
curl -s -f http://localhost:60015/ > /dev/null

echo "Sending compute request (triggers CPU load)..."
curl -s -f http://localhost:60015/compute > /dev/null

echo "Sending error request (triggers HTTP 500 error)..."
# Expect status 500, ignore exit code
curl -s http://localhost:60015/error > /dev/null || true

# 4. Wait for scrape interval (2s)
echo -e "\n${GREEN}Waiting for Prometheus scrape loop (4 seconds)...${NC}"
sleep 4

# 5. Query Prometheus for metrics
echo -e "\n${GREEN}Verifying Prometheus metrics collection...${NC}"
PROMETHEUS_METRICS=$(curl -s "http://localhost:60016/api/v1/query?query=http_requests_total")
echo "Prometheus API response: $PROMETHEUS_METRICS"

if [[ "$PROMETHEUS_METRICS" == *"http_requests_total"* ]]; then
  echo -e "${GREEN}✓ Success: Found http_requests_total metrics in Prometheus!${NC}"
else
  echo -e "${RED}✗ Error: http_requests_total metrics not found in Prometheus!${NC}"
  exit 1
fi

# 6. Verify logging sidecar
echo -e "\n${GREEN}Verifying Logging Sidecar output...${NC}"
sleep 2 # Let Fluent Bit flush
LOGS=$(docker compose logs logging-sidecar 2>&1)
echo "Tailed sidecar logs snippet:"
echo "$LOGS" | tail -n 10

if [[ "$LOGS" == *"fluent-bit"* && "$LOGS" == *"/error"* ]]; then
  echo -e "${GREEN}✓ Success: Logging sidecar is tailing and enriching Go access logs!${NC}"
else
  echo -e "${RED}✗ Error: Logging sidecar does not show processed logs!${NC}"
  exit 1
fi

echo -e "\n${GREEN}=== All Verifications Passed Successfully! ===${NC}"
