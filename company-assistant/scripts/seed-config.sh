#!/bin/sh
# Seed the engine with agent configuration on first startup.
# This script is mounted into /docker-entrypoint.d/ and runs before the engine starts.
#
# It waits for the engine API to become available, then imports agents.yaml
# via the config import endpoint.

set -e

CONFIG_FILE="/etc/bytebrew/agents.yaml"
ENGINE_URL="http://localhost:8443"
MAX_RETRIES=30
RETRY_INTERVAL=2

if [ ! -f "$CONFIG_FILE" ]; then
  echo "[seed-config] No config file found at $CONFIG_FILE, skipping."
  exit 0
fi

echo "[seed-config] Waiting for engine API to become ready..."

for i in $(seq 1 $MAX_RETRIES); do
  if curl -sf "${ENGINE_URL}/api/v1/health" > /dev/null 2>&1; then
    echo "[seed-config] Engine is ready. Importing configuration..."

    RESPONSE=$(curl -sf -X POST \
      -H "Content-Type: application/x-yaml" \
      -u "${ADMIN_USER:-admin}:${ADMIN_PASSWORD:-changeme}" \
      -d @"$CONFIG_FILE" \
      "${ENGINE_URL}/api/v1/config/import" 2>&1) || true

    echo "[seed-config] Import response: $RESPONSE"
    echo "[seed-config] Configuration imported successfully."
    exit 0
  fi

  echo "[seed-config] Attempt $i/$MAX_RETRIES -- engine not ready yet, retrying in ${RETRY_INTERVAL}s..."
  sleep $RETRY_INTERVAL
done

echo "[seed-config] WARNING: Engine did not become ready in time. Config was NOT imported."
echo "[seed-config] You can import manually: curl -X POST -H 'Content-Type: application/x-yaml' -d @config/agents.yaml http://localhost:8443/api/v1/config/import"
exit 1
