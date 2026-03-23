#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="${ROOT_DIR}/.dev-runtime"
PID_FILE="${LOG_DIR}/client.pid"

echo "============================================"
echo "  Restarting Development Environment"
echo "============================================"

# Stop client dev server if running
if [[ -f "${PID_FILE}" ]]; then
  EXISTING_PID="$(cat "${PID_FILE}")"
  if kill -0 "${EXISTING_PID}" >/dev/null 2>&1; then
    echo "Stopping client dev server (PID: ${EXISTING_PID})..."
    kill "${EXISTING_PID}" 2>/dev/null || true
    sleep 1
  fi
  rm -f "${PID_FILE}"
fi

# Restart docker containers
echo "Restarting docker compose services..."
cd "${ROOT_DIR}"
export HOST_IP=$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo "localhost")
echo "Detected HOST_IP: ${HOST_IP}"
docker compose down
docker compose up -d

echo ""
echo "Docker services restarted!"
echo "MQTT Broker: ${HOST_IP}:1883"
echo ""

# Restart using the start script
echo "Starting client dev server..."
./scripts/dev-start.sh
