#!/usr/bin/env bash
# kind-smoke.sh — end-to-end smoke test for the FalseFlag operator.
#
# Spins up a kind cluster, installs the CRDs + operator via Kustomize,
# applies the sample manifests, asserts the API received the project
# via curl, then tears down. Best-effort: kind must be installed and
# docker reachable; the FalseFlag API must already be running on the
# host (e.g. via `make up`) so the operator can reach it through the
# host.docker.internal alias.
#
# Exit codes:
#   0 — smoke passed
#   1 — assertion failed (operator did not reconcile in time)
#   2 — missing prerequisite (kind, docker, curl)
set -euo pipefail

CLUSTER_NAME=${CLUSTER_NAME:-falseflag-smoke}
API_BASE_URL=${FALSEFLAG_API_BASE_URL:-http://localhost:8080}
TIMEOUT=${TIMEOUT:-120}

cleanup() {
  if [[ "${SKIP_TEARDOWN:-}" == "1" ]]; then
    echo "skipping teardown (SKIP_TEARDOWN=1)"
    return
  fi
  echo "==> tearing down kind cluster $CLUSTER_NAME"
  kind delete cluster --name "$CLUSTER_NAME" || true
}

if ! command -v kind >/dev/null 2>&1; then
  echo "kind not installed; install from https://kind.sigs.k8s.io and re-run"
  exit 2
fi
if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl not installed"
  exit 2
fi
if ! command -v curl >/dev/null 2>&1; then
  echo "curl not installed"
  exit 2
fi

trap cleanup EXIT

echo "==> ensuring API is reachable at $API_BASE_URL"
if ! curl --silent --fail "$API_BASE_URL/healthz" >/dev/null; then
  echo "FalseFlag API not running on $API_BASE_URL — run 'make up' first"
  exit 2
fi

if [[ "${SKIP_CREATE:-}" == "1" ]]; then
  echo "==> reusing existing kind cluster $CLUSTER_NAME (SKIP_CREATE=1)"
else
  echo "==> creating kind cluster $CLUSTER_NAME"
  kind create cluster --name "$CLUSTER_NAME"
fi

echo "==> applying operator overlay"
kubectl apply -k deploy/kustomize/overlays/dev

echo "==> waiting for operator deployment to become ready"
kubectl -n falseflag-system rollout status deployment/falseflag-operator --timeout="${TIMEOUT}s"

echo "==> applying sample manifests"
kubectl apply -k deploy/samples

echo "==> polling API for project 'demo'"
deadline=$(( $(date +%s) + TIMEOUT ))
while true; do
  if curl --silent --fail "$API_BASE_URL/v1/projects/demo" >/dev/null; then
    echo "==> ✓ API has project 'demo'"
    break
  fi
  if [[ $(date +%s) -ge $deadline ]]; then
    echo "==> ✗ project did not appear within ${TIMEOUT}s"
    kubectl -n falseflag-system logs deployment/falseflag-operator --tail=200 || true
    exit 1
  fi
  sleep 3
done

echo "==> polling API for flag 'banner'"
deadline=$(( $(date +%s) + TIMEOUT ))
while true; do
  if curl --silent --fail "$API_BASE_URL/v1/projects/demo/flags/banner" >/dev/null; then
    echo "==> ✓ API has flag 'banner'"
    break
  fi
  if [[ $(date +%s) -ge $deadline ]]; then
    echo "==> ✗ flag did not appear within ${TIMEOUT}s"
    kubectl -n falseflag-system logs deployment/falseflag-operator --tail=200 || true
    exit 1
  fi
  sleep 3
done

echo "==> ✓ smoke passed"
