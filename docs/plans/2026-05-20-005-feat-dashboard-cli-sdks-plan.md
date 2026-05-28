---
title: "feat: Dashboard, CLI, and SDKs for FalseFlag"
type: feat
status: completed
date: 2026-05-20
slice: 5
---

# Dashboard, CLI, and SDKs

## Overview

Slice 5 turns FalseFlag from a "things-talking-to-an-API" backend into
something a person can actually use. Slices 1–4 produced a Go control
plane (REST on `:8080`, ConnectRPC on `:8090`), a Postgres-backed
store, three configuration strategies (JSON / CEL / TypeScript), a
generated TypeScript client, and a Kubernetes operator with seven
CRDs. Slice 5 wraps those surfaces in the human- and SDK-facing
artifacts the conference audience will actually see: a Remix
dashboard, a Commander CLI, a Go SDK, and a TypeScript SDK — plus a
tiny edge proxy and a hardened cross-runtime conformance harness.

The slice fans out cleanly. Phase 1 lands the small amount of shared
foundation work that every other phase needs (snapshot export
format, OpenFeature-shaped provider conventions, generated-client
re-exports). After that, **TypeScript SDK**, **Go SDK**, **Proxy**,
**CLI**, **Dashboard**, **Golden corpus**, and **Playwright** can be
worked on in parallel along non-overlapping write paths. The Main
thread reconciles generated artifacts, root files (`Makefile`,
`js/package.json`), and runs the final validation ladder.

This is still demo-quality. The CLI `login` is a stub that writes a
credentials JSON file. OpenFeature compliance is "OpenFeature-shaped",
not full-spec. The proxy is single-binary, single-replica, no leader
election. The dashboard ships pages, not polish. What matters: every
visible workflow described in `docs/ideation/2026-05-20-synthetic-feature-flag-platform-depot-demo-ideation.md`
has a credible end-to-end path by the end of this slice, and the
slice-7 Depot demo gets six new slow-CI surfaces (Vitest, Playwright,
extended Hurl, SDK conformance on two runtimes, dashboard SSR build,
and proxy image build) that benefit visibly from Depot Cache and
Depot container builds.

## Problem Statement / Motivation

The metaplan's quality bar is "broad credible surface area, visible
workflows, real build/test complexity." Slices 1–4 are correct but
invisible — every interaction so far has been `curl`, `hurl`, or
`kubectl apply`. At PlatformCon, the audience will see:

1. A **dashboard** that lists projects/flags, shows config source,
   compiles snapshots, surfaces audit history, and explains an
   evaluation trace.
2. A **CLI** that runs the full "validate → save → snapshot → evaluate"
   loop a developer would run in CI.
3. **SDKs** in Go and TypeScript that fetch a snapshot and evaluate
   locally — with identical decisions across runtimes, proven by a
   shared golden corpus.
4. A **proxy** that serves the same snapshot evaluation over HTTP for
   environments that can't speak Connect or run an SDK directly.

Without slice 5 the demo has nothing to point a camera at. With it,
the CI/CD acceleration story in slice 7 has eight more meaningfully
slow targets to compare baseline-vs-Depot against.

This slice also exercises three surfaces the slice-7 CI demo cares
about:

1. **Vitest matrix.** Six pnpm packages × multiple test files; cold
   `node_modules` install is the headline slow path Depot Cache
   targets.
2. **Playwright end-to-end.** Real browser, real Chromium download,
   real Remix SSR build — perfect Depot Cache target.
3. **Go SDK + corpus tests.** Adding `internal/sdkgo` adds another
   `go test ./...` package that runs the full conformance corpus on
   every CI run.

## Proposed Solution

Ship slice 5 as **nine committable phases** across **five
parallelizable worker tracks**. Phase 1 must land first; phases 2–4
and 7 may run in parallel after Phase 1; phases 5 and 6 may run in
parallel after Phase 2; phase 8 follows phase 6; phase 9 is the
main-thread integration. Each phase corresponds to a phase-bundling
commit set in line with slices 1–4 (which used 6–22 commits per
slice).

### Worker tracks (per METAPLAN Fan-Out C)

| Worker | Owned write paths | Phases |
|---|---|---|
| TypeScript SDK worker | `js/packages/sdk-js/**` | 2 |
| Go SDK / Proxy worker | `internal/sdkgo/**`, `internal/proxy/**`, `cmd/falseflag-proxy/**`, `cmd/falseflag-sdkgo-demo/**` | 3, 4 |
| CLI worker | `js/apps/cli/**` | 5 |
| Dashboard worker | `js/apps/dashboard/**` (excl. `playwright/`) | 6 |
| Golden corpus worker | `tests/eval-corpus/**`, `js/packages/shared-eval-corpus/**`, `internal/sdkgo/conformance_test.go`, `js/packages/sdk-js/tests/conformance.test.ts` | 7 |
| Playwright owner (dashboard worker continues) | `js/apps/dashboard/playwright/**`, `js/apps/dashboard/playwright.config.ts` | 8 |
| Main thread | `Makefile`, `js/package.json`, `js/turbo.json`, `js/pnpm-workspace.yaml`, `go.mod`, `go.sum`, `infra/compose.yaml`, `infra/docker-bake.hcl`, `db/seed/**`, `cmd/falseflag-seed/**`, `docs/**`, generated artifacts | 1, 9 |

### Cross-cutting contracts

- **Snapshot wire shape** (Phase 1) — both SDKs and the proxy consume
  the same `/v1/projects/{slug}/snapshots/latest` response. Already
  defined in slice 3; Phase 1 only adds an `application/x-falseflag-snapshot`
  cousin of the JSON shape suitable for `falseflag snapshot export`
  output (a stable canonical form ordered for diff readability).
- **OpenFeature-shaped provider** (Phase 1) — `docs/sdk-openfeature.md`
  enumerates the four method signatures both SDKs implement
  (`booleanValue`, `stringValue`, `numberValue`, `objectValue`),
  including the agreed `EvaluationContext` shape and the agreed
  `Decision` shape. Both SDKs already produce identical Decisions in
  slice 2; Phase 1 promotes that into a documented contract.
- **CLI ↔ API** — CLI uses `@falseflag/generated-client`. Phase 1 adds
  `validateConfig` and `publishVersion` re-exports if Orval misses any.
- **Dashboard ↔ API** — Remix server-side loaders use
  `@falseflag/generated-client`. Phase 1 wraps the global-fetch hack
  the existing `_index.tsx` loader uses into a single
  `apiFetch(baseUrl, init)` helper at
  `js/apps/dashboard/app/lib/api.server.ts` so every new loader
  imports it cleanly.

## Technical Considerations

### Architecture choices

- **Dashboard is SSR-only for slice 5.** Every loader runs server-side
  and calls the API directly. No client-side data fetching, no
  client-side router state beyond the Remix defaults. This keeps the
  Playwright story simple (just hit the URL, assert HTML).
- **TypeScript SDK is isomorphic but tested under Node.** The
  `createClient` already shipped in slice 2 runs in both. Slice 5
  doesn't change runtime targets — just extends the surface.
- **Go SDK uses the existing IR evaluator.** `internal/sdkgo` imports
  `internal/config` and `internal/eval` rather than reimplementing the
  evaluator. The headline value is a clean `Client` + `Provider`
  facade with snapshot polling, cache, and OpenFeature-shaped types —
  not a second evaluator.
- **Proxy is a thin HTTP wrapper around the Go SDK.** `internal/proxy`
  embeds `internal/sdkgo.Client` and exposes `POST /v1/evaluate` and
  `GET /healthz`. Single binary at `cmd/falseflag-proxy`. The slice-1
  scaffold already has a stub `cmd/falseflag-proxy/main.go`; slice 5
  fills it in.
- **Snapshot polling cadence is 10s by default**, configurable via
  `FALSEFLAG_SNAPSHOT_POLL_INTERVAL`. The dashboard does NOT poll;
  Remix loaders fetch on each request. The CLI does NOT poll; each
  command is one-shot.

### What slice 5 deliberately does not do

- No real authentication. `falseflag auth login` writes a stub JSON
  file at `~/.config/falseflag/credentials.json` with a `bearer` of
  `demo-token` (per the slice 3 `X-Actor` header convention) and the
  CLI reads it back as the `X-Actor` value.
- No client-side React framework beyond Remix's built-in. No Redux,
  no React Query. Server-side loaders only.
- No realtime SSE/streaming snapshot push. The METAPLAN status notes
  mark this as a slice 5+ gap; slice 5 keeps it deferred.
- No full OpenFeature spec compliance. We ship the four key
  resolution methods and `EvaluationContext`, not hooks, providers
  registry, finally-stage logic, or telemetry.
- No real auth, RBAC, or per-project access control on the dashboard.

### Demo dataset

The slice 1 + 2 `make smoke` Hurl files seed a small project. Slice 5
extends this into a richer demo dataset under
`db/seed/2026-05-20-demo-seed.sql` invoked by `cmd/falseflag-seed`:

- 3 projects (`acme-web`, `acme-mobile`, `acme-internal`)
- 4 environments per project (`production`, `staging`, `qa`, `dev`)
- 6 segments (`enterprise-tier`, `beta-cohort`, `internal-employees`, etc.)
- ~25 flags spread across the three strategies (~9 JSON, ~9 CEL, ~7 TS)
- Pre-published snapshots so the SDK conformance tests have something
  to evaluate against without a "first write" race.
- A trailing 200-row audit log so the dashboard audit page has
  pagination to show off.

## System-Wide Impact

### Interaction Graph

```
Dashboard (Remix SSR) ─┐
CLI (Commander/Node)  ─┼──> REST :8080  ──> store (pgxpool, sqlc)
TS SDK (browser/Node) ─┤      └ Connect :8090 (slice 3, unchanged)
                       │
Go SDK (in-process) ───┘──> REST :8080 (snapshot polling only)
        │
        └──> internal/eval.Evaluate (in-process IR eval)

Proxy ──> Go SDK ──> REST :8080 (snapshot poll)
  │
  └── exposes /v1/evaluate HTTP for clients that don't run the SDK

Operator (slice 4) ─── unchanged; still Connect-only.
```

Concretely:

1. A `falseflag config save` CLI invocation triggers a sequence: CLI
   `validateConfig` → API validate handler → CLI `publishVersion` →
   API mutation + audit-event txn → CLI `compileSnapshot` → API
   snapshot endpoint → store writes a `snapshots` row → CLI prints
   the resulting snapshot id and flag count.
2. Each subsequent `falseflag-proxy` snapshot poll (every 10s) calls
   `GET /v1/projects/{slug}/snapshots/latest`; if the `etag` is
   unchanged the API returns 304, the proxy keeps its cache.
3. Each dashboard page request triggers fresh Remix loaders that
   call the API server-side; HTML lands in the user's browser.

### Error & Failure Propagation

| Failure | Surface impact |
|---|---|
| API down on dashboard load | Loader catches, returns `{ projects: [], error }`; route renders an empty-state with a banner. Already implemented in `_index.tsx`; new loaders follow the same pattern. |
| API down on CLI command | CLI prints the error to stderr, exits non-zero. `falseflag smoke-test` aborts on first non-zero step. |
| API down on SDK poll | SDK keeps serving from cached snapshot; logs to `slog`/`console.warn`. `defaultValue` is returned only if there is no snapshot at all. |
| Stale snapshot served | Proxy/SDK transparently use last-good. Documented behavior; matches OpenFeature spirit. |
| Playwright Chromium download failure | `make dashboard-e2e` records as a known infra failure; not blocking for CI grade in this slice. |
| Generated-client drift | `make generate-check` already catches it. Slice 5 extends to include `pnpm orval` (added in slice 3) — already covered. |

### State Lifecycle Risks

- **CLI credentials file.** `falseflag auth login` writes `~/.config/falseflag/credentials.json`
  with mode `0600`. If the directory doesn't exist, the CLI creates
  it. If the file exists, the CLI overwrites it without a confirm
  prompt (demo-quality). Documented in the CLI README.
- **Proxy snapshot cache.** Single in-memory cache per process. On
  restart, first request blocks on the initial poll (or returns
  `503` with `Retry-After: 1` if the API is unreachable).
- **Dashboard SSR caching.** None. Remix loaders fetch on every
  request. This is fine for the demo and avoids stale-data confusion
  during a live presentation.

### API Surface Parity

Every surface added in slice 5 must consume only endpoints that
already exist at the start of slice 5 (the slice 3 OpenAPI + Connect
contracts). Slice 5 changes **zero** Go API handlers. The only
backend code slice 5 ships outside `internal/sdkgo` and
`internal/proxy` is:

- `cmd/falseflag-seed/main.go` — uses the existing API (or store
  directly) to populate the demo dataset.
- `internal/server/handlers/snapshots.go` — extends the snapshot list
  endpoint to support `?since=<rfc3339>` filtering, **only if**
  Phase 6 (dashboard) needs it. If not strictly required, skip.

### Integration Test Scenarios

These cross-layer scenarios are NOT covered by individual worker unit
tests and must run as part of the final validation:

1. **CLI smoke-test e2e** — Run `falseflag smoke-test --project acme-web`
   against the live compose stack; expect zero-exit and a printed
   evaluation result.
2. **Playwright happy path** — Visit `/`, click into `acme-web`, click
   into a flag, click "Evaluate", assert the trace UI shows at least
   one rule node.
3. **SDK conformance — Go** — `go test ./internal/sdkgo/...` runs all
   25 fixtures, asserts byte-identical Decision JSON.
4. **SDK conformance — TS** — `pnpm --filter @falseflag/sdk test` runs
   the same 25 fixtures through the TS evaluator, asserts byte-identical
   Decision JSON.
5. **Proxy snapshot survives API restart** — Hurl test: poll proxy
   /v1/evaluate, restart API container, poll again, assert same
   decision served from cache.
6. **Dashboard SSR fetches API** — `pnpm --filter @falseflag/dashboard build`
   produces a Vite SSR bundle, and Playwright (with a real compose API)
   confirms the rendered HTML contains seeded project names.

## Acceptance Criteria

### Functional Requirements

#### TypeScript SDK (`@falseflag/sdk`)
- [ ] `createClient({ baseUrl, projectSlug, pollIntervalMs? })` returns
      a long-lived client with `start()`, `stop()`, `getSnapshot()`,
      and OpenFeature-shaped `boolean(key, ctx, defaultValue)`,
      `string`, `number`, `object` methods.
- [ ] `FalseFlagProvider` wraps `createClient` to expose the OpenFeature
      provider interface (`resolveBooleanEvaluation`, `resolveStringEvaluation`,
      `resolveNumberEvaluation`, `resolveObjectEvaluation`).
- [ ] Local snapshot evaluation: when the client has a snapshot, no
      network call happens per evaluation; on snapshot miss returns
      `defaultValue` with `reason: "ERROR" | "DEFAULT"`.
- [ ] Vitest tests for: snapshot polling, snapshot caching, provider
      interface conformance, default-value fallback.

#### Go SDK (`internal/sdkgo`)
- [ ] `sdkgo.NewClient(ctx, opts)` returns a `*Client` with
      `Start(ctx)`, `Stop()`, `Snapshot()`, and OpenFeature-shaped
      `BooleanValue(ctx, key, def, EvalContext)`, `StringValue`,
      `NumberValue`, `ObjectValue` methods.
- [ ] `sdkgo.Provider` exposes the OpenFeature-inspired Provider
      interface with `Metadata()`, `BooleanEvaluation()`,
      `StringEvaluation()`, `NumberEvaluation()`, `ObjectEvaluation()`.
- [ ] Local IR evaluation reuses `internal/eval.Evaluate` (no
      duplicated evaluator code).
- [ ] Unit tests for client lifecycle (start/stop, cache hit/miss).
- [ ] Conformance test reading `tests/eval-corpus/*.json` (lives in
      Phase 7).

#### Proxy (`internal/proxy`, `cmd/falseflag-proxy`)
- [ ] Binary boots, polls a configured project's snapshot via the SDK,
      exposes `POST /v1/evaluate {key, defaultValue, context}` returning
      a `Decision`.
- [ ] `GET /healthz` returns OK once at least one snapshot has been
      loaded; before that, 503.
- [ ] `infra/compose.yaml` adds a `proxy` service exposed on `:8081`.
- [ ] `infra/docker-bake.hcl` already has a `proxy` target from slice 1.
- [ ] Hurl test under `tests/hurl/11-proxy-evaluate.hurl` runs at least
      4 evaluations against the live proxy.

#### CLI (`@falseflag/cli`)
- [ ] `falseflag auth login --token <token>` writes
      `~/.config/falseflag/credentials.json` (mode 0600).
- [ ] `falseflag auth whoami` reads the credentials file and prints
      the stored actor.
- [ ] `falseflag project list` — already shipped in slice 3, unchanged.
- [ ] `falseflag flag list --project <slug>` — already shipped, unchanged.
- [ ] `falseflag config validate --project <slug> --strategy <s> --file <path>`
      sends to the API validate endpoint and prints validation result.
- [ ] `falseflag config save --project <slug> --flag <key> --file <path>`
      validates then publishes a new version. Prints the new version id.
- [ ] `falseflag snapshot latest --project <slug>` — already shipped.
- [ ] `falseflag snapshot export --project <slug> [--format json|yaml] [--out <path>]`
      writes the canonical snapshot output to stdout or a file.
- [ ] `falseflag smoke-test --project <slug>` chains validate → save →
      snapshot → evaluate and exits zero on success.
- [ ] Each new command has at least one Vitest unit test stubbing the
      fetch layer.

#### Dashboard (`@falseflag/dashboard`)
- [ ] `/` — replace with a redirect to `/projects`.
- [ ] `/projects` — table of projects with links into detail pages.
- [ ] `/projects/$slug` — project detail: environments, flag count,
      latest snapshot version, "Edit config" CTA.
- [ ] `/projects/$slug/flags` — flag list table with strategy badges.
- [ ] `/projects/$slug/flags/$key` — flag detail showing the latest
      version source, value type, rule count.
- [ ] `/projects/$slug/flags/$key/edit` — config-as-code editor (plain
      `<textarea>` for slice 5, no Monaco) with a Validate button that
      calls the API validate endpoint via a form action.
- [ ] `/projects/$slug/snapshots` — list of snapshot versions with a
      "Diff against previous" CTA (diff renders as a server-side
      computed JSON-diff list).
- [ ] `/projects/$slug/audit` — paginated audit list, filterable by
      `action` and `actor`.
- [ ] `/projects/$slug/flags/$key/trace` — invokes
      `EvaluateWithTrace`, renders the rule-by-rule trace as a
      collapsible tree.
- [ ] Tailwind for styling. At least three Radix UI primitives in use
      (recommended: Dialog for the publish-confirm modal, Tabs for the
      strategy switcher, Toast for save-result feedback).
- [ ] Vitest snapshot tests for each route's component-level rendering
      (loader stubbed).

#### Golden corpus & SDK conformance
- [ ] `tests/eval-corpus/` expanded from 15 → 25 fixtures, covering
      at least: string-eq, string-in, number-gt, number-lt, boolean,
      matches-regex, rollout-0%, rollout-25%, rollout-50%, rollout-100%,
      AND-of-N, OR-of-N, NOT, deeply-nested predicate trees, and an
      explicit "default-fallback" case where no rule matches.
- [ ] Go conformance test at `internal/sdkgo/conformance_test.go` reads
      every fixture and asserts the SDK's Decision is byte-identical
      to the expected Decision in each fixture file.
- [ ] TS conformance test at `js/packages/sdk-js/tests/conformance.test.ts`
      does the same via `@falseflag/shared-eval-corpus`.
- [ ] `make conformance` (new Makefile target) runs both.

#### Playwright (`js/apps/dashboard/playwright/**`)
- [ ] `playwright.config.ts` sets `webServer` to `pnpm --filter @falseflag/dashboard dev`,
      and `baseURL` to `http://127.0.0.1:3000`.
- [ ] At least one spec at `playwright/dashboard.spec.ts` covers:
      navigate to `/`, follow redirect, see project list, click into
      a project, see flag list, click into a flag, click into "Trace",
      assert at least one rule row visible.
- [ ] `make dashboard-e2e` runs Playwright against the compose stack
      (compose must already be `up`).

#### Demo seed
- [ ] `cmd/falseflag-seed/main.go` populates the demo dataset.
- [ ] `make seed` runs the binary against compose Postgres.
- [ ] `make smoke` continues to pass after `make seed` (idempotent —
      seed is a no-op on second run).

### Non-Functional Requirements

- **Performance.** Dashboard SSR page render < 250ms with a warm
  `node_modules`. SDK in-process evaluation < 100µs for the median
  fixture.
- **Build size.** `pnpm --filter @falseflag/dashboard build` produces
  a Vite SSR bundle under 1.5MB gzipped. (Not a hard cap — informational.)
- **Demo-quality posture.** No real auth, no rate limiting, no CSRF
  protection, no TLS in the proxy. All called out in READMEs.
- **Cross-runtime parity.** The 25-fixture conformance suite must
  produce byte-identical Decision JSON in Go and TS.

### Quality Gates

- [ ] `go build ./cmd/... && go vet ./... && go test ./...` green.
- [ ] `make generate-check` green (idempotent codegen on a clean tree).
- [ ] `make contract-test` green (slice 3 REST↔Connect parity preserved).
- [ ] `pnpm --dir js -r typecheck && pnpm --dir js -r test && pnpm --dir js -r build && pnpm --dir js lint` green.
- [ ] `make smoke` green with both new Hurl files included.
- [ ] `make conformance` green (Go + TS, 25 fixtures each, all match).
- [ ] `make dashboard-e2e` green against running compose stack.
- [ ] `make bake-print` green (proxy + dashboard images in default group).

## Technical Approach

### Architecture

```
                              ┌─────────────────────────┐
                              │  Postgres (slice 1-3)   │
                              └────────────┬────────────┘
                                           │
                              ┌────────────▼────────────┐
                              │   falseflag-api (Go)    │
                              │  REST :8080  Connect :8090  │
                              └─────┬─────┬─────┬───────┘
                                    │     │     │
                ┌───────────────────┘     │     └────────────────────┐
                │                         │                          │
        ┌───────▼──────┐         ┌────────▼──────────┐      ┌────────▼────────┐
        │   CLI (TS)   │         │ Dashboard (Remix) │      │  Operator (Go)  │
        │  Commander   │         │  SSR-only loaders │      │   slice 4       │
        └───────┬──────┘         └───────────────────┘      └─────────────────┘
                │
                ▼
        ┌───────────────┐
        │  ~/.config/   │
        │  credentials  │
        └───────────────┘

                              ┌─────────────────────────┐
                              │  falseflag-api  (REST)  │
                              └────────────┬────────────┘
                                           │ /v1/projects/{slug}/snapshots/latest
                  ┌────────────────────────┼────────────────────────┐
                  │                        │                        │
        ┌─────────▼─────────┐    ┌─────────▼──────────┐    ┌────────▼─────────┐
        │ TS SDK (sdk-js)   │    │ Go SDK (sdkgo)     │    │ Proxy (Go)       │
        │ in-process eval   │    │ in-process eval    │    │ wraps Go SDK     │
        └───────────────────┘    └────────────────────┘    │ HTTP :8081       │
                                                          └──────────────────┘
```

### Implementation Phases

Each phase below corresponds to one phase-bundling commit set on
`main`. Phase boundaries are commit boundaries (not branch
boundaries) — slices 1–4 all shipped directly to `main` via phase
bundles, and slice 5 does the same.

#### Phase 1 — Shared Foundation (main thread, blocks all others)

Write paths: `docs/sdk-openfeature.md`, `js/packages/generated-client-ts/src/index.ts`
(re-exports only), `js/apps/dashboard/app/lib/api.server.ts`,
`tests/eval-corpus/README.md`, `Makefile` (new targets:
`conformance`, `dashboard-e2e`, `seed`).

Tasks:

1. Author `docs/sdk-openfeature.md` describing the four-method
   provider contract, the `EvaluationContext` shape, and the
   `Decision`/`Reason` shape. Linked from both SDK READMEs.
2. Re-export `validateConfig`, `publishVersion`, `compileSnapshot`,
   `evaluateFlag`, `evaluateWithTrace`, `listSnapshots`,
   `listAuditEvents`, `listSegments`, `listEnvironments` from
   `@falseflag/generated-client` (Orval already generates them;
   confirm/extend the barrel export).
3. Extract the global-fetch hack from `js/apps/dashboard/app/routes/_index.tsx`
   into `js/apps/dashboard/app/lib/api.server.ts` so every new
   dashboard loader uses one helper.
4. Add empty Makefile targets: `make conformance`, `make dashboard-e2e`,
   `make seed`. Each prints "TODO" + exits 0. Phases 5–9 fill them in.
5. Document the snapshot canonical output format (key ordering,
   trailing newline, RFC 7396 alignment) in `docs/snapshot-format.md`
   so phases 2 / 3 / 5 don't drift.

Estimated commits: **2–3**.
Acceptance: docs land, generated-client re-exports compile,
`pnpm --dir js -r build && pnpm --dir js -r test` still green.

#### Phase 2 — TypeScript SDK (sdk-js worker)

Write paths: `js/packages/sdk-js/src/**`, `js/packages/sdk-js/tests/**`,
`js/packages/sdk-js/package.json`, `js/packages/sdk-js/README.md`.

Tasks:

1. Add `src/provider.ts` exporting `FalseFlagProvider` implementing the
   OpenFeature-shaped interface from `docs/sdk-openfeature.md`. Wraps
   the existing `createClient`.
2. Extend `src/client.ts` to add `start()`, `stop()`, `getSnapshot()`,
   and the four resolution methods (`boolean`, `string`, `number`,
   `object`). Polls `/v1/projects/{slug}/snapshots/latest`.
3. Add ETag handling to the polling loop. The slice-3 snapshot
   endpoint returns an `ETag`; on 304 the cache is preserved.
4. Update `src/index.ts` barrel to re-export `FalseFlagProvider` and
   `createClient` (no breaking changes).
5. Vitest tests:
   - `tests/provider.test.ts` — provider returns correct values for a
     stubbed snapshot.
   - `tests/poll.test.ts` — polling backoff and ETag cache behavior
     with a fake fetch.
   - Existing tests stay green.
6. `README.md` documents the OpenFeature usage path.

Estimated commits: **3–4**.

#### Phase 3 — Go SDK (sdkgo worker)

Write paths: `internal/sdkgo/**`, `cmd/falseflag-sdkgo-demo/**`
(optional 30-line example binary), `internal/sdkgo/README.md`.

Tasks:

1. New package `internal/sdkgo`:
   - `client.go` — `Client` struct with HTTP poll loop using
     `net/http` and `context.Context`.
   - `provider.go` — `Provider` interface and `FalseFlagProvider`
     implementing it; methods call `Client.Evaluate`.
   - `evaluate.go` — small wrapper around `internal/eval.Evaluate`
     translating the SDK's `EvalContext` into the IR's expected
     attribute map.
   - `types.go` — `Decision`, `Reason`, `EvalContext`,
     `ProviderMetadata`. Match the documented shape from
     Phase 1's `docs/sdk-openfeature.md`.
2. Unit tests:
   - `client_test.go` — client start/stop with a `httptest.Server`
     stubbing the API.
   - `provider_test.go` — four resolution methods returning
     expected types.
3. Optional `cmd/falseflag-sdkgo-demo/main.go` — 30-line binary that
   creates a client against `http://localhost:8080`, evaluates one
   flag, prints the decision. Acts as documentation + smoke check.

Estimated commits: **3–4**.

#### Phase 4 — Proxy (sdkgo/proxy worker, depends on Phase 3)

Write paths: `internal/proxy/**`, `cmd/falseflag-proxy/main.go`,
`tests/hurl/11-proxy-evaluate.hurl`, `infra/compose.yaml` (add proxy
service block — main thread reconciles).

Tasks:

1. `internal/proxy/server.go` — HTTP server with `POST /v1/evaluate`
   and `GET /healthz`. Wraps an `*sdkgo.Client`.
2. `cmd/falseflag-proxy/main.go` — already a slice-1 stub; replace
   the placeholder with a real entrypoint that constructs the
   `sdkgo.Client`, starts polling, mounts the proxy server, listens
   on `:8081`. Honors `FALSEFLAG_API_BASE_URL` and
   `FALSEFLAG_PROXY_PROJECT_SLUG` env vars.
3. Unit tests:
   - `internal/proxy/server_test.go` — `httptest.NewServer` covering
     happy path, missing-snapshot 503, malformed body 400.
4. `tests/hurl/11-proxy-evaluate.hurl` — boots-after-seed proxy
   evaluations: 4 requests against `:8081`, all 200.
5. `infra/compose.yaml` adds a `proxy` service block (Main thread
   reconciles in Phase 9).
6. `make smoke` extended to include the new Hurl file. Phase 9
   integration owns the `Makefile` edit.

Estimated commits: **2–3**.

#### Phase 5 — CLI (CLI worker, depends on Phase 2 only if it imports the SDK)

Write paths: `js/apps/cli/src/**`, `js/apps/cli/tests/**`,
`js/apps/cli/package.json`, `js/apps/cli/README.md`.

Note: the CLI does NOT need to import `@falseflag/sdk`. Every CLI
command can hit the API directly via `@falseflag/generated-client`.
This makes Phase 5 effectively parallel with Phase 2.

Tasks:

1. Split `src/index.ts` (currently a single 200-line file) into
   `src/commands/{auth,project,flag,config,snapshot,smoke}.ts`. The
   `createProgram` factory stays in `src/index.ts` and wires the
   subcommands.
2. Add `auth.ts` — `auth login --token <t>` writes
   `~/.config/falseflag/credentials.json` with mode 0600;
   `auth whoami` reads and prints.
3. Add `config.ts` — `config validate` and `config save` call the API
   validate/publish endpoints, accepting `--strategy json|cel|typescript`
   and `--file <path>`.
4. Extend `snapshot.ts` — `snapshot export` writes canonical JSON or
   YAML (yaml dependency: `yaml` package).
5. Add `smoke.ts` — `smoke-test --project <slug>` chains
   validate→save→snapshot→evaluate using a built-in trivial flag
   spec; prints each step.
6. Vitest tests for each new command using `nock`-style fetch stubs
   (or hand-rolled `vi.spyOn(global, 'fetch')`).

Estimated commits: **4–5**.

#### Phase 6 — Dashboard (Dashboard worker, depends on Phase 1)

Write paths: `js/apps/dashboard/app/routes/**`,
`js/apps/dashboard/app/components/**`, `js/apps/dashboard/app/lib/**`,
`js/apps/dashboard/tests/**`, `js/apps/dashboard/package.json`,
`js/apps/dashboard/app/tailwind.css`.

Tasks:

1. Replace `_index.tsx` with a redirect to `/projects`.
2. Add the eight routes listed in Acceptance Criteria → Dashboard.
   Each loader uses the `api.server.ts` helper from Phase 1.
3. Add Radix-based components under `app/components/`:
   - `<StrategyBadge />` — Tailwind pill colored per strategy.
   - `<Tabs />` — Radix Tabs wrapping the strategy switcher on the
     edit page.
   - `<Dialog />` — Radix Dialog wrapping the publish-confirm modal.
   - `<Toast />` — Radix Toast for save feedback.
   - `<TraceTree />` — recursive tree renderer for the EvaluateWithTrace
     response.
4. Tailwind theme: extend the slice-1 `falseflag-*` color palette
   with a `strategy-json`, `strategy-cel`, `strategy-typescript`
   accent set.
5. Vitest snapshot tests for each route's main component using
   `@remix-run/testing` to render with a stubbed loader.

Estimated commits: **6–8**.

#### Phase 7 — Golden corpus & SDK conformance (corpus worker, depends on Phases 2 + 3)

Write paths: `tests/eval-corpus/**`,
`js/packages/shared-eval-corpus/src/**`,
`js/packages/shared-eval-corpus/tests/**`,
`internal/sdkgo/conformance_test.go`,
`js/packages/sdk-js/tests/conformance.test.ts`.

Tasks:

1. Expand `tests/eval-corpus/` from 15 → 25 fixtures. New fixtures
   target SDK-specific cases:
   - `16-default-no-rules.json`
   - `17-default-all-rules-miss.json`
   - `18-rollout-25-percent.json`
   - `19-rollout-50-percent.json`
   - `20-and-of-3.json`
   - `21-or-of-3.json`
   - `22-not-around-and.json`
   - `23-deeply-nested.json`
   - `24-string-list-membership.json`
   - `25-number-comparison-edge.json`
2. Each fixture is a self-contained JSON file:
   `{spec, context, expected: {value, reason, ruleId?}}`.
3. `js/packages/shared-eval-corpus/src/index.ts` exports a `loadCorpus()`
   helper that returns all fixtures with stable ordering.
4. `internal/sdkgo/conformance_test.go` loads every fixture under
   `tests/eval-corpus/` (relative path resolution via `_test.go`
   `runtime.Caller`), runs `sdkgo.Client.Evaluate`, asserts
   byte-identical Decision JSON.
5. `js/packages/sdk-js/tests/conformance.test.ts` does the same.
6. `make conformance` runs both as one target.

Estimated commits: **2–3**.

#### Phase 8 — Playwright (Dashboard worker again, depends on Phase 6)

Write paths: `js/apps/dashboard/playwright/**`,
`js/apps/dashboard/playwright.config.ts`,
`js/apps/dashboard/package.json` (test scripts only).

Tasks:

1. `playwright.config.ts` — uses `webServer` to launch
   `pnpm --filter @falseflag/dashboard dev` against `FALSEFLAG_API_BASE_URL`.
2. `playwright/dashboard.spec.ts` — the happy-path spec covering
   projects → flag → trace.
3. `Makefile` `dashboard-e2e` target wires Playwright runner.

Estimated commits: **2**.

#### Phase 9 — Demo seed & integration (main thread)

Write paths: `cmd/falseflag-seed/**`, `db/seed/**`,
`infra/compose.yaml`, `Makefile`, root READMEs.

Tasks:

1. `cmd/falseflag-seed/main.go` — small Go binary that POSTs the
   3-project / 25-flag dataset to the API. Idempotent via API
   `409 ALREADY_EXISTS` handling.
2. `db/seed/2026-05-20-demo-seed.sql` — alternative path that runs
   directly against Postgres (preferred for speed; main path).
3. `infra/compose.yaml` — add the `proxy` service (Phase 4 deferred
   this); add `falseflag-seed` as a one-shot init service that runs
   after the API healthcheck succeeds.
4. `make seed` runs the binary or the SQL (one-shot).
5. Run the full validation ladder, fix any cross-phase friction.
6. Update `docs/METAPLAN.md` Status Notes with the slice 5 close-out.

Estimated commits: **1–2**.

### Parallel Fan-Out Plan

```
                          Phase 1 (main)
                              │
        ┌─────────────────┬───┴───┬─────────────────┐
        │                 │       │                 │
     Phase 2          Phase 3   Phase 5         (Phase 6 starts
     TS SDK           Go SDK     CLI              after Phase 1
        │                │       │                 too — depends on
        │             Phase 4    │                  api.server.ts only)
        │              Proxy     │
        │                │       │
        └────────┬───────┴───────┘
                 │
              Phase 7
              Corpus + conformance
                 │
              Phase 6 (Dashboard) runs in parallel with phases 2–4
                 │
              Phase 8
              Playwright (after Dashboard exists)
                 │
              Phase 9
              Seed + integration (main thread)
```

#### Concrete parallel start lines

After Phase 1 lands:

- Worker A (TS SDK) starts Phase 2.
- Worker B (Go SDK + Proxy) starts Phase 3 → Phase 4.
- Worker C (CLI) starts Phase 5 — only writes `js/apps/cli/**` so does
  not conflict with Worker A.
- Worker D (Dashboard) starts Phase 6 — only writes `js/apps/dashboard/**`.
- Worker E (Corpus) starts Phase 7 fixtures (new files only; Go and
  TS test files land after Phases 2 + 3 merge).

The main thread integrates each worker's merge in turn. Generated
artifacts are reconciled on the main thread only — workers never
regenerate.

## Alternative Approaches Considered

### Single-PR mono-merge

Land the whole slice in one branch with one merge. Rejected because
slices 1–4 used phase-bundling commits on `main` for the same reason:
each phase is independently reviewable and the metaplan status notes
record each phase. Keeping the convention preserves the slice-1
`git log` shape that the slice-7 CI demo references.

### Real OpenFeature spec compliance

We could implement the full OpenFeature `client.getBooleanValue`
pipeline including hooks, finally-stage logic, and provider registry.
Rejected because the metaplan explicitly defers full OpenFeature
compliance ("Demo quality does not require: complete OpenFeature
compliance"). Slice 5 ships an OpenFeature-shaped interface that's
recognizable to the audience.

### Real authentication

We could wire OAuth or even a simple username/password flow. Rejected
because the metaplan defers hardened auth and the slice 3 `X-Actor`
header convention already gives us audit attribution. The CLI stub
preserves the demo's audit story (`falseflag-cli/$user`) without
adding a real auth surface.

### Monaco for the dashboard config editor

The config-as-code editor on `/projects/$slug/flags/$key/edit` could
use Monaco. Rejected: Monaco adds ~700KB to the SSR bundle and a
nontrivial client-only boundary. A plain `<textarea>` is good enough
for the demo and Playwright. Future slices can swap in Monaco.

### A separate "@falseflag/openfeature-provider" package

We could split the provider into its own pnpm package. Rejected:
extra package overhead is not justified at slice 5 demo-quality.
Provider lives inside `@falseflag/sdk` and is re-exported.

### Server-Sent Events for snapshot push

The slice 4 status notes called out real-time SSE as a slice 5+ gap.
We deliberately keep it deferred: polling at 10s is sufficient for
the demo, easier to test in Playwright, and avoids the snapshot-push
contract entering slice 5's design surface area.

## Risk Analysis & Mitigation

### Risk: Dashboard SSR build slows the inner loop

The Remix dashboard's Vite SSR build takes ~8s on a cold cache today
(with one route); adding 8 more routes and Radix components plausibly
pushes that past 25s. **Mitigation:** Phase 6 ends with `pnpm --filter @falseflag/dashboard build`
in the local validation ladder; if the cold build crosses 60s we
trim. The slice-7 Depot demo benefits from a slower-but-believable
build, but it must not be painful in development.

### Risk: Playwright Chromium download breaks CI

Playwright bundles a ~150MB Chromium download. **Mitigation:** Phase 8
documents the install step and gates `make dashboard-e2e` behind a
`PLAYWRIGHT_BROWSERS_PATH` check; slice 7's CI work will own the
Depot Cache wiring.

### Risk: Cross-runtime corpus diverges

The slice 2 cross-runtime corpus already shows that 15 fixtures pass
byte-identically. Adding 10 more risks introducing a fixture that
exposes a JS vs Go floating-point or string-coercion difference.
**Mitigation:** Phase 7 land fixtures one-at-a-time; if a fixture
shows divergence, fix it before adding the next. The 15 slice-2
fixtures are already trusted, so cross-runtime drift is unlikely.

### Risk: Proxy snapshot cache loses ETag semantics

The proxy honors API ETags only if `internal/sdkgo` does. If the SDK
sends `If-None-Match` correctly, the API returns 304 and the SDK
keeps cache. **Mitigation:** Phase 3 unit-tests this with a
`httptest.Server` that returns 304 once a previous ETag is seen.

### Risk: CLI credentials file location varies across OSes

`~/.config/` is Linux convention; macOS uses
`~/Library/Application Support/`; Windows uses `%APPDATA%`. For demo
purposes we hard-code `~/.config/falseflag/` and document it. The
demo is always run on macOS or Linux. **Mitigation:** README note;
defer cross-platform polish to slice 8.

### Risk: Dashboard pages couple to API field names

If the OpenAPI schema renames a field, the dashboard breaks at run
time. **Mitigation:** every loader uses Orval's Zod schemas to parse
responses (slice 3 pattern). Schema drift fails fast at the loader
boundary with a structured error.

### Risk: `make seed` race condition

If `make seed` runs before the API is healthy, it errors. **Mitigation:**
`cmd/falseflag-seed/main.go` retries the health endpoint up to 30s
with 1s backoff; compose `depends_on` with `condition: service_healthy`
where supported.

### Risk: Phase 9 grows into a rewrite

Slice 4's Phase 9 ("verification + status notes") stayed disciplined
at ~1 commit. Slice 5 has more cross-cutting surfaces (seed, compose
edits, Makefile targets, READMEs). **Mitigation:** the Phase 9 PR is
budgeted at 1–2 commits. If integration uncovers a worker-level bug,
fix in the owning worker's followup commit, not in Phase 9.

## Validation Ladder

Each phase ends with the phase-local subset. Phase 9 runs the entire
ladder.

### Per-phase quick check

| Phase | Quick check |
|---|---|
| 1 | `pnpm --dir js -r build && pnpm --dir js -r test` |
| 2 | `pnpm --filter @falseflag/sdk test && pnpm --filter @falseflag/sdk build` |
| 3 | `go test ./internal/sdkgo/... && go build ./cmd/falseflag-sdkgo-demo` |
| 4 | `go test ./internal/proxy/... && go build ./cmd/falseflag-proxy && hurl --test tests/hurl/11-proxy-evaluate.hurl` (with stack up) |
| 5 | `pnpm --filter @falseflag/cli test && pnpm --filter @falseflag/cli build` |
| 6 | `pnpm --filter @falseflag/dashboard test && pnpm --filter @falseflag/dashboard build` |
| 7 | `make conformance` |
| 8 | `make dashboard-e2e` (with compose stack up) |
| 9 | full ladder below |

### Full validation ladder (Phase 9)

```bash
# Go
go build ./cmd/...
go vet ./...
go test ./...
FALSEFLAG_TEST_DATABASE_URL=postgres://... go test ./internal/store/... ./internal/server/...
make generate-check
make contract-test
make conformance

# JS
pnpm --dir js install --frozen-lockfile
pnpm --dir js -r typecheck
pnpm --dir js -r test
pnpm --dir js -r build
pnpm --dir js lint

# End-to-end
docker compose -f infra/compose.yaml up -d
make seed
make smoke           # extended to 12-15 Hurl files (existing 10 + proxy + new dashboard probes if any)
make dashboard-e2e   # Playwright
docker compose -f infra/compose.yaml down

# Images
make bake-print
```

### Pass criteria

- All commands return exit 0.
- `git diff --exit-code` is clean after `make generate-check`.
- Playwright produces an artifact directory under
  `js/apps/dashboard/playwright-report/`.
- Conformance test prints "25/25 fixtures match" for each runtime.

## Success Metrics

- 9 dashboard routes shipped (the existing `_index` redirect plus 8
  new pages).
- 8 CLI commands shipped (3 existing + 5 new).
- 25 golden fixtures, byte-identical across Go and TS.
- 1 Playwright spec, ≥3 navigation steps.
- ≥4 new Hurl probes (proxy + any new dashboard smoke).
- All slice-1-through-4 validations still green.
- `docs/METAPLAN.md` checklist updated to mark slice 5 plan / implement / verify done.

## Dependencies & Prerequisites

- Slice 1 foundation (already shipped).
- Slice 2 configuration strategies + shared eval corpus + cel-lite + IR (already shipped).
- Slice 3 REST + Connect surfaces + Orval-generated client + Zod schemas (already shipped).
- Slice 4 operator (not strictly required for slice 5, but live on
  `main` so slice 5 must not regress slice 4 reconciler tests).
- New runtime dependencies:
  - JS: `@radix-ui/react-dialog`, `@radix-ui/react-tabs`,
    `@radix-ui/react-toast` (Phase 6); `yaml` (Phase 5);
    `@playwright/test` (Phase 8).
  - Go: none new — `internal/sdkgo` and `internal/proxy` use
    standard library + existing internal packages.

## Future Considerations

- **Slice 6 (MCP server)** consumes the same API surfaces. Slice 5's
  generated-client extensions and the SDK's evaluation contract are
  exactly what MCP tool implementations will reuse.
- **Slice 7 (slow CI + Depot)** picks up six new slow surfaces from
  slice 5: TS unit tests, dashboard SSR build, Playwright, Go SDK
  tests, Go conformance, and proxy image build. The slice-5 Makefile
  targets (`make conformance`, `make dashboard-e2e`, `make seed`) are
  pre-wired for CI consumption.
- **Slice 8 (polish + demo script)** can finally enable a one-command
  local smoke: `make seed && make smoke && make dashboard-e2e`.
- **Real auth, real OpenFeature, Monaco editor, SSE push** all remain
  deferred; documented above.

## Documentation Plan

- `docs/sdk-openfeature.md` — provider contract (Phase 1).
- `docs/snapshot-format.md` — canonical snapshot output format (Phase 1).
- `internal/sdkgo/README.md` — usage + example (Phase 3).
- `internal/proxy/README.md` — proxy operational notes (Phase 4).
- `js/packages/sdk-js/README.md` — extended with OpenFeature section (Phase 2).
- `js/apps/cli/README.md` — full command reference (Phase 5).
- `js/apps/dashboard/README.md` — route map + dev notes (Phase 6).
- `js/apps/dashboard/playwright/README.md` — how to run e2e locally (Phase 8).
- `db/seed/README.md` — what the demo dataset contains (Phase 9).
- `docs/METAPLAN.md` Status Notes — close-out entry for slice 5 (Phase 9).

## Sources & References

### Internal References

- METAPLAN entry 5 (slice 5 prompt) — `docs/METAPLAN.md:432-447`
- Slice 1 plan — `docs/plans/2026-05-20-001-feat-foundation-monorepo-scaffold-plan.md`
- Slice 2 plan (config strategies + cross-runtime corpus) — `docs/plans/2026-05-20-002-feat-configuration-strategies-plan.md`
- Slice 3 plan (API surfaces + generated client) — `docs/plans/2026-05-20-003-feat-api-grpc-openapi-plan.md`
- Slice 4 plan (operator, parallel-fan-out structure precedent) — `docs/plans/2026-05-20-004-feat-operator-crds-plan.md`
- Ideation — `docs/ideation/2026-05-20-synthetic-feature-flag-platform-depot-demo-ideation.md`
- Historical reference — `docs/ideation/2026-05-20-moonconfig-historical-reference.md`
- Existing TS SDK — `js/packages/sdk-js/src/index.ts`, `client.ts`, `evaluator.ts`
- Existing CLI — `js/apps/cli/src/index.ts`
- Existing dashboard skeleton — `js/apps/dashboard/app/routes/_index.tsx`
- Existing eval corpus — `tests/eval-corpus/`, `js/packages/shared-eval-corpus/`
- Generated client — `js/packages/generated-client-ts/` (npm name `@falseflag/generated-client`)
- IR evaluator — `internal/eval/`, `internal/config/`
- Inspiration repo — `/Users/wito/code/project-keat` (read-only)

### External References

- OpenFeature provider spec — https://openfeature.dev/specification/sections/providers
- Remix loaders — https://remix.run/docs/en/main/route/loader
- Radix UI primitives — https://www.radix-ui.com/primitives
- Playwright `webServer` config — https://playwright.dev/docs/test-webserver
- Commander.js — https://github.com/tj/commander.js

### Related Slices

- Slice 4 (operator) — same parallel-fan-out approach.
- Slice 6 (MCP) — consumes slice 5's SDK + generated-client surface.
- Slice 7 (CI + Depot) — depends on slice 5's slow surfaces existing.
