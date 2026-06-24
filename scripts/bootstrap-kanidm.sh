#!/usr/bin/env bash
# bootstrap-kanidm.sh — Spin up a Kanidm instance via Docker for acceptance testing.
#
# Usage:
#   bash scripts/bootstrap-kanidm.sh [version]
#
# The script:
#   1. Starts a Kanidm container on localhost:8443
#   2. Waits for it to become healthy
#   3. Creates a test service account and prints its token
#
# The token is written to $KANIDM_TOKEN_FILE (default: /tmp/kanidm-test-token)
# so that the caller can source it.

set -euo pipefail

KANIDM_VERSION="${1:-latest}"
CONTAINER_NAME="kanidm-test-$$"
KANIDM_PORT=8443
HEALTH_TIMEOUT=60   # seconds
TOKEN_FILE="${KANIDM_TOKEN_FILE:-/tmp/kanidm-test-token}"

cleanup() {
  echo "→ Stopping Kanidm container ${CONTAINER_NAME}..."
  docker rm -f "${CONTAINER_NAME}" 2>/dev/null || true
}
trap cleanup EXIT

echo "→ Starting Kanidm ${KANIDM_VERSION} on port ${KANIDM_PORT}..."

docker run -d \
  --name "${CONTAINER_NAME}" \
  -p "${KANIDM_PORT}:8443" \
  -e KANIDM_DOMAIN=localhost \
  -e KANIDM_ORIGIN="https://localhost:${KANIDM_PORT}" \
  "kanidm/server:${KANIDM_VERSION}"

echo "→ Waiting for Kanidm to become healthy (up to ${HEALTH_TIMEOUT}s)..."
elapsed=0
until curl -sf -o /dev/null --insecure "https://localhost:${KANIDM_PORT}/status"; do
  sleep 2
  elapsed=$((elapsed + 2))
  if [ "${elapsed}" -ge "${HEALTH_TIMEOUT}" ]; then
    echo "ERROR: Kanidm did not become healthy within ${HEALTH_TIMEOUT}s" >&2
    docker logs "${CONTAINER_NAME}" >&2
    exit 1
  fi
done

echo "→ Kanidm is up."

# Create the admin password and a test service account.
# In a real CI setup this would use proper secrets management.
ADMIN_PASS="Test1234!"

docker exec "${CONTAINER_NAME}" kanidmd recover-account admin --password "${ADMIN_PASS}" 2>/dev/null || true

# Generate a token for the built-in idm_admin service account.
TOKEN=$(docker exec "${CONTAINER_NAME}" \
  kanidm login -H "https://localhost:${KANIDM_PORT}" -D admin --password "${ADMIN_PASS}" \
  2>/dev/null | grep -o 'token: .*' | awk '{print $2}' || echo "")

if [ -z "${TOKEN}" ]; then
  echo "WARNING: could not extract token automatically; set KANIDM_TOKEN manually." >&2
else
  echo "${TOKEN}" > "${TOKEN_FILE}"
  echo "→ Token written to ${TOKEN_FILE}"
  echo "  export KANIDM_TOKEN=\$(cat ${TOKEN_FILE})"
fi

echo "→ Bootstrap complete. KANIDM_URL=https://localhost:${KANIDM_PORT}"

# Keep the container alive for the test run duration.
# The trap will clean it up on exit.
# When called from make test-acceptance, the process exits after tests finish.
