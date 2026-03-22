#!/bin/sh
set -e

ENGINE_URL="http://localhost:8443"
MAX_WAIT=30
WAITED=0

echo "Waiting for Engine..."
while ! wget -q --spider "$ENGINE_URL/api/v1/health" 2>/dev/null; do
  sleep 1
  WAITED=$((WAITED + 1))
  if [ "$WAITED" -ge "$MAX_WAIT" ]; then
    echo "Engine not ready after ${MAX_WAIT}s"
    exit 1
  fi
done
echo "Engine ready"

# Login
TOKEN=$(wget -qO- --post-data='{"username":"'"${ADMIN_USER:-admin}"'","password":"'"${ADMIN_PASSWORD:-changeme}"'"}' \
  --header='Content-Type: application/json' \
  "$ENGINE_URL/api/v1/auth/login" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "Login failed"
  exit 1
fi

# Import config
CONFIG_FILE="/etc/bytebrew/agents.yaml"
if [ -f "$CONFIG_FILE" ]; then
  echo "Importing config..."
  wget -qO- --post-file="$CONFIG_FILE" \
    --header="Authorization: Bearer $TOKEN" \
    --header="Content-Type: application/x-yaml" \
    "$ENGINE_URL/api/v1/config/import"
  echo "Config imported"
fi

# Seed sales settings
echo "Seeding settings..."
for setting in "max_discount_percent:15" "free_shipping_min:100" "quote_validity_days:7" "sales_tax_rate:8.25"; do
  KEY="${setting%%:*}"
  VALUE="${setting#*:}"
  wget -qO- --method=PUT \
    --body-data="{\"value\": \"${VALUE}\"}" \
    --header="Authorization: Bearer $TOKEN" \
    --header="Content-Type: application/json" \
    "$ENGINE_URL/api/v1/settings/${KEY}" 2>/dev/null || true
done
echo "Settings seeded"

echo "All examples configured"
