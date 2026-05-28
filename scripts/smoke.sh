#!/usr/bin/env bash
# scripts/smoke.sh runs the Hurl test suite against a FalseFlag stack.
# Picks the backend from FALSEFLAG_BACKEND (default: postgres). For
# postgres the DB is reset via psql TRUNCATE; for sqlite the volume is
# wiped via `docker compose down -v && up -d` because the api image is
# distroless and has no sqlite3 client.
#
# Exit status mirrors hurl's.

set -euo pipefail

cd "$(dirname "$0")/.."

API_URL="${FALSEFLAG_SMOKE_BASE_URL:-http://localhost:8080}"
MCP_URL="${FALSEFLAG_MCP_BASE_URL:-http://localhost:8091}"
BACKEND="${FALSEFLAG_BACKEND:-postgres}"

case "${BACKEND}" in
  postgres) COMPOSE_FILE="${FALSEFLAG_COMPOSE_FILE:-compose.yaml}" ;;
  sqlite)   COMPOSE_FILE="${FALSEFLAG_COMPOSE_FILE:-compose.sqlite.yaml}" ;;
  *)        echo "smoke: unknown FALSEFLAG_BACKEND=${BACKEND} (expected postgres or sqlite)"; exit 2 ;;
esac

echo "smoke: backend=${BACKEND} compose=${COMPOSE_FILE}"

case "${BACKEND}" in
  postgres)
    echo "smoke: probing ${API_URL}/healthz"
    if ! curl -fsS -o /dev/null "${API_URL}/healthz"; then
      echo "smoke: API not reachable at ${API_URL}. Run \`make up\` first."
      exit 1
    fi
    echo "smoke: resetting DB state via docker compose exec"
    docker compose -f "${COMPOSE_FILE}" exec -T db \
      psql -U falseflag -d falseflag -v ON_ERROR_STOP=1 \
        -c "TRUNCATE TABLE audit_events, snapshots, segments, environments, flag_versions, flags, projects RESTART IDENTITY CASCADE" \
      >/dev/null
    ;;
  sqlite)
    echo "smoke: wiping SQLite volume and rebooting api"
    docker compose -f "${COMPOSE_FILE}" down -v >/dev/null
    docker compose -f "${COMPOSE_FILE}" up -d --build >/dev/null
    echo "smoke: waiting for ${API_URL}/healthz"
    for i in $(seq 1 60); do
      if curl -fsS -o /dev/null "${API_URL}/healthz"; then
        break
      fi
      sleep 1
      if [ "${i}" -eq 60 ]; then
        echo "smoke: API never came up at ${API_URL}"
        docker compose -f "${COMPOSE_FILE}" logs --tail 50 api
        exit 1
      fi
    done
    ;;
esac

echo "smoke: seeding demo dataset (proxy needs acme-web for /readyz)"
go run ./cmd/falseflag-seed >/dev/null

# The proxy polls the API on a 10s interval; if it started before the
# seed ran (the common case) its cached snapshot is stale. Restart so
# its first poll picks up the seeded acme-web project immediately.
echo "smoke: restarting proxy so it picks up the fresh seed"
docker compose -f "${COMPOSE_FILE}" restart proxy >/dev/null
for i in $(seq 1 30); do
  if curl -fsS -o /dev/null "http://localhost:8081/readyz"; then
    break
  fi
  sleep 1
  if [ "${i}" -eq 30 ]; then
    echo "smoke: proxy /readyz never returned 200"
    docker compose -f "${COMPOSE_FILE}" logs --tail 30 proxy
    exit 1
  fi
done

echo "smoke: running hurl --test --jobs 1 tests/hurl/*.hurl"
# Files depend on state seeded by earlier files in lexical order;
# --jobs 1 keeps Hurl from running them concurrently. mcp_base_url
# is required by 12-mcp-tools.hurl; defaulting it here keeps `make
# smoke` self-contained when the compose mcp service is up.
hurl --test --jobs 1 --variable "mcp_base_url=${MCP_URL}" tests/hurl/*.hurl
