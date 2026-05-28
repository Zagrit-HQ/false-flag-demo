# FalseFlag demo — smoke walkthrough

A step-by-step pass through the demo path. Each step lists the
literal command, the expected output (truncated where verbose), and
what's interesting about it. Designed to be run cold from a clean
clone in ~5 minutes.

Assumes `go`, `pnpm`, `hurl`, `docker`, and `curl` are on `$PATH`.

## 1. Install JS workspace deps

```bash
pnpm --dir js install
```

Expected: pnpm resolves the lockfile and writes `js/node_modules`.
Takes ~30s on a warm cache, 1–2 min cold.

## 2. Boot the full stack via docker compose

```bash
docker compose up -d --build
```

Expected: builds and starts `db`, `api`, `proxy`, `operator`, `mcp`,
and `dashboard` containers. The first build pulls Go and Node base
images and takes 2–3 minutes; subsequent builds reuse cache and
finish in ~30 seconds.

Confirm:

```bash
docker compose ps
# All 6 services should be "Up" / "Up (healthy)".
curl -s http://localhost:8080/healthz
# {"status":"ok","service":"falseflag-api",...}
curl -s http://localhost:3030/projects -o /dev/null -w "%{http_code}\n"
# 200
```

## 3. Seed three projects worth of demo data

```bash
make seed
```

Expected output (truncated):

```
INFO  project ok  slug=acme-web
INFO  flag ok     project=acme-web flag=checkout-redesign strategy=json
INFO  flag ok     project=acme-web flag=max-cart-items   strategy=json
INFO  flag ok     project=acme-web flag=checkout-banner-text strategy=json
INFO  flag ok     project=acme-web flag=proxy-smoke-bool strategy=json
INFO  snapshot ok project=acme-web
INFO  project ok  slug=acme-mobile
INFO  flag ok     project=acme-mobile flag=force-update-required strategy=cel
INFO  flag ok     project=acme-mobile flag=push-notification-cadence strategy=cel
INFO  snapshot ok project=acme-mobile
INFO  project ok  slug=acme-internal
INFO  flag ok     project=acme-internal flag=feature-x          strategy=typescript
INFO  flag ok     project=acme-internal flag=dark-mode-default  strategy=typescript
INFO  snapshot ok project=acme-internal
INFO  seed complete
```

Three projects, seven flags spanning all three strategies (`json`,
`cel`, `typescript`), and a compiled snapshot per project. The seed
is idempotent — re-run it any time.

## 4. Walk the API surface with Hurl

```bash
make smoke
```

Expected: 14 hurl files, 89 requests, all green.

```
Executed files:    14
Executed requests: 89 (~1000/s)
Succeeded files:   14 (100.0%)
Failed files:      0 (0.0%)
```

This covers: health, projects, flags (JSON/CEL/TS publish), evaluate,
environments, segments, snapshots, audit, evaluate-trace, Connect
parity, proxy evaluate, MCP tools, pagination, and the
source_text-backed TS publish path added in slice 8.

## 5. Open the dashboard

```bash
open http://localhost:3030
```

Click into `acme-internal` → `feature-x`. Notice:

- The view route renders the **real TypeScript source** with Shiki
  syntax highlighting — no "compiled IR" fallback caption, because
  slice 9 phase 1 seeds `source_text` for every flag.
- Click **Edit**. Monaco lazy-loads (3.62 kB view route bundle stays
  Monaco-free; Monaco lives in its own chunk).

## 6. Edit a flag and watch it propagate

Inside Monaco, change `default: false` to `default: true`, then
click **Save**.

You land back on the view route with a green toast:

> **v2 published.** Compile a snapshot to propagate this edit to
> SDKs and the proxy.
> [Publish snapshot]

Click **Publish snapshot**. The dashboard navigates to the snapshots
index where the newly compiled row sits at the top.

## 7. Confirm the edit is live in the proxy

The proxy polls the latest snapshot every 10 seconds and serves
evaluations from its in-memory copy. After the snapshot publishes:

```bash
curl -sX POST http://localhost:8081/v1/evaluate \
  -H 'Content-Type: application/json' \
  -d '{"key":"proxy-smoke-bool","context":{"user":{"plan":"pro"}}}'
```

```
{"value":true,"reason":"rule_matched","rule_id":"pro-only","version":3}
```

Note `version` increments each time `make seed` re-runs.

## 8. Tear down

```bash
docker compose down -v       # -v also drops the Postgres volume
```

## What just happened

- **Flag versions are immutable.** Every save creates a new
  `flag_versions` row. The version number is one higher than the
  previous version, scoped per flag.
- **Snapshots are project-wide.** Each snapshot bundles the latest
  version of every flag in the project into a single immutable
  blob. SDKs and the proxy poll this snapshot, not individual flags.
- **Edits don't propagate until you publish a snapshot.** This is
  the architectural beat the toast in step 6 makes visible — it's
  the model the conference talk uses to argue for the snapshot
  abstraction.
- **The server is the authoritative TypeScript compiler.** When you
  save a TS flag, the API runs your `.ts` through esbuild + goja
  inside its own process (no CGO, no subprocess) and stores both
  the source text and the produced IR. The dashboard renders the
  stored source verbatim with Shiki; the proxy evaluates against
  the stored IR.

For the conference-pacing version of this walk, see
[`docs/demo/script.md`](./script.md).
