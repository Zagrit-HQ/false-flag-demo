#!/usr/bin/env bash
# scripts/mcp-smoke.sh runs the MCP Hurl smoke against the
# compose-managed falseflag-mcp service. Requires `make up` (which
# now includes the mcp service) and `make seed` so the project list
# is non-empty.
#
# Best-effort: if hurl isn't installed, the script exits 0 with a
# warning so CI without hurl doesn't break. Matches the pattern in
# scripts/kind-smoke.sh.

set -euo pipefail

cd "$(dirname "$0")/.."

if ! command -v hurl >/dev/null 2>&1; then
  echo "mcp-smoke: hurl not on \$PATH; skipping. Install with \`brew install hurl\`."
  exit 0
fi

MCP_URL="${FALSEFLAG_MCP_BASE_URL:-http://localhost:8091}"
MCP_HEALTH_URL="${FALSEFLAG_MCP_HEALTH_URL:-http://localhost:8092}"

echo "mcp-smoke: probing ${MCP_HEALTH_URL}/healthz"
if ! curl -fsS -o /dev/null "${MCP_HEALTH_URL}/healthz"; then
  echo "mcp-smoke: MCP not reachable at ${MCP_HEALTH_URL}. Run \`make up\` first."
  exit 1
fi

echo "mcp-smoke: running hurl --test tests/hurl/12-mcp-tools.hurl"
hurl --test --variable "mcp_base_url=${MCP_URL}" tests/hurl/12-mcp-tools.hurl
