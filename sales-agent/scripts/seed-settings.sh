#!/bin/sh
# Seed business rule settings into the engine via the Settings API.
# Run this after the engine is up and configuration is imported.
#
# Usage: ./scripts/seed-settings.sh [ENGINE_URL]
#   ENGINE_URL defaults to http://localhost:8443

set -e

ENGINE_URL="${1:-http://localhost:8443}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-changeme}"

echo "[seed-settings] Seeding business rule settings..."

put_setting() {
  KEY="$1"
  VALUE="$2"
  DESCRIPTION="$3"

  RESPONSE=$(curl -sf -X PUT \
    -H "Content-Type: application/json" \
    -u "${ADMIN_USER}:${ADMIN_PASSWORD}" \
    -d "{\"value\": \"${VALUE}\", \"description\": \"${DESCRIPTION}\"}" \
    "${ENGINE_URL}/api/v1/settings/${KEY}" 2>&1) || true

  echo "[seed-settings] ${KEY} = ${VALUE} -- ${RESPONSE}"
}

put_setting "max_discount_percent" "15" "Maximum discount percentage allowed on quotes"
put_setting "free_shipping_min" "100" "Minimum order amount for free shipping (USD)"
put_setting "quote_validity_days" "7" "Number of days a quote remains valid"
put_setting "sales_tax_rate" "8.25" "Sales tax rate percentage"

echo "[seed-settings] Done."
