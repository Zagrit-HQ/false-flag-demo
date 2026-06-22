# internal/proxy

The FalseFlag evaluation proxy — a thin HTTP server that wraps
`internal/sdkgo` and exposes flag evaluation over a tiny REST surface
for environments that can't link the SDK directly (legacy services,
non-Go runtimes that don't yet have a FalseFlag SDK, sidecar setups).

## Endpoints

| Method | Path             | Purpose |
|--------|------------------|---------|
| GET    | `/healthz`       | Liveness probe — always 200 when the binary is up. Compatible with the slice-1 contract. |
| GET    | `/readyz`        | Readiness — 200 once the first snapshot poll succeeds and any configured required flags are present, 503 while starting. Returns 200 immediately if no project is configured (idle ready). |
| POST   | `/v1/evaluate`   | Evaluate a flag against the cached snapshot. Body: `{key, default_value?, context?}`. Returns `Decision` (value/reason/rule_id/version). |
| GET    | `/v1/snapshot`   | Returns metadata about the loaded snapshot (id, version, flag count). |

## Configuration

| Env var | Default | Purpose |
|---|---|---|
| `FALSEFLAG_PROXY_ADDR` | `:8081` | Listen address. |
| `FALSEFLAG_API_BASE_URL` | `http://localhost:8080` | REST API the proxy polls snapshots from. |
| `FALSEFLAG_PROXY_PROJECT_SLUG` | _(empty)_ | Scopes polling to one project. When empty, the proxy still boots and serves `/healthz`, but `/v1/evaluate` returns 503 — useful for compose liveness checks. |
| `FALSEFLAG_PROXY_READY_FLAGS` | _(empty)_ | Optional comma-separated flag keys that must be present in the loaded snapshot before `/readyz` returns 200. Compose sets this to `proxy-readiness-bool` so E2E tests wait for seeded data instead of only process liveness. |

## Demo-quality posture

- Single-replica only. No leader election, no horizontal scaling.
- No TLS termination — the proxy is meant to live behind another LB.
- No authentication on `/v1/evaluate`. The slice 5 control plane
  itself has no auth; adding it here would be theatre.
- No metrics. Slice 7 owns the CI/Depot timing story.
- Single project per proxy process. Multi-project support is a future
  slice — for the demo, one proxy ↔ one project keeps the surface area
  small.
