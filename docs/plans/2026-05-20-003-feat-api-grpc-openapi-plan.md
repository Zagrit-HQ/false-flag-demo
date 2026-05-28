---
title: "feat: API, gRPC, and OpenAPI Parity Surface for FalseFlag"
type: feat
status: completed
date: 2026-05-20
slice: 3
---

# API, gRPC, and OpenAPI Parity Surface

## Overview

Slice 3 turns FalseFlag's control plane into a credible two-surface API.
The existing oapi-codegen REST surface (10 operations across `projects`,
`flags`, `flag_versions`, and `evaluate`) gets **extended** with the
domain pieces the demo's product narrative implies but slice 2 left out
— **environments**, **segments**, **snapshots**, **audit search**, and
**evaluation-with-trace**. In parallel, every resource gets a
**ConnectRPC** sibling under `proto/falseflag/v1/`, served from a
second `http.Server` on its own port, so the proxy, the operator, and
the MCP server (slices 4 and 6) can call the control plane over gRPC
without forking business logic.

The two surfaces are not independent code paths. They share one Store
layer, one error-translation helper, one set of validation rules, and
one set of generated artifacts. The headline test artifact for this
slice is the **REST↔Connect contract test** at
`internal/server/contract_test.go`: every parity operation runs through
both an `httptest`-driven REST client and an in-process Connect client
against the same in-memory server, and the resulting domain values must
be equal.

This slice also closes two slice-2 carry-overs: **`AppendAudit` is
wired from the publish handler** (it was implemented in slice 2 but
never called), and the **dashboard and CLI consume the generated
client** (slice 2 generated it but neither app imported it). Orval
gains a second output block that emits **Zod schemas** alongside the
fetch client.

Demo-quality stays the rule. There is no auth, no per-environment flag
override (that's slice 4), no real-time subscription (that's slice 4),
no sandbox execution of TypeScript source (later slice), no operator
write-back (slice 4), and no segment versioning. Environments are
reference data, segments are project-scoped reusable predicates that
inline into the flag IR at publish time, and snapshots are immutable
project-wide compiled bundles produced synchronously on POST.

## Problem Statement / Motivation

Slice 2 shipped flag publication and evaluation, but only across a
narrow REST slice with no concept of environment, no reusable
targeting predicate, no compiled release bundle, and no audit history
readable from the API. For the conference demo to land, the control
plane has to look like a real platform from three vantage points:

1. **The CLI and dashboard** need to do more than print "not yet
   implemented." A user has to walk through *create project → create
   environment → define a segment → author a flag → compile snapshot
   → evaluate with trace → inspect audit log* in the demo, and every
   one of those steps must produce a believable response.
2. **The proxy and operator** (slices that follow) need a stable RPC
   surface to call into the control plane. The current
   HTTP-only-with-a-stranded-`HealthService`-proto situation makes
   slice 4 either invent its own contract or live with REST-over-gRPC,
   both of which look amateurish on stage.
3. **The MCP server** (slice 6) needs typed clients in both Go and
   TypeScript. Today the TypeScript fetch client exists but
   neither app uses it, and there are no Zod schemas the MCP server
   can validate against.

The slice 2 close-out called out three deferred items that land here:
"Real-time push to SDKs (slice 3)" — partially; we ship the **pull**
side (snapshots + Connect endpoints the proxy will call), not the push
side (SSE/WebSocket, deferred to slice 4 with the operator);
**dashboard UI for editing flags (slice 5)** stays deferred; **the
audit append plumbing is finished** here.

Until slice 3 lands, the dashboard and CLI are placeholder screens,
the proxy has nowhere to pull from, and the API has no idea who edited
what when. Slice 3 is the load-bearing slice for the operator
(slice 4), the dashboard UI (slice 5), the MCP server (slice 6), and
the demo script's "show the platform from three angles" beat.

## Proposed Solution

Build the slice in six phases, each committed directly to `main`
following the slice 1/2 cadence (~3–6 small conventional commits per
phase). The deliverables, in dependency order:

1. **Schema + store extensions.** Goose migration `0003_environments_segments_snapshots.sql`
   adds three new tables and an `actor` column on `audit_events`.
   New SQLC queries; new store methods. Owns: `db/migrations`,
   `db/queries`, `internal/db` (generated), `internal/store`.
2. **OpenAPI extension + REST handlers.** Bump `openapi.yaml` to
   `v0.3.0`. Add 14 new operations across environments, segments,
   snapshots, audit-events, and evaluate-trace. Wire `AppendAudit`
   from `PublishFlagVersion`. Owns: `api/openapi`,
   `internal/server/handlers`, `internal/server` plumbing.
3. **Proto + ConnectRPC handlers + second listener.** New `.proto`
   files alongside `health.proto`. ConnectRPC handlers that delegate
   to the same `Store`. Second `http.Server` on `FALSEFLAG_API_RPC_ADDR`
   (default `:8090`). Owns: `proto/falseflag/v1`,
   `internal/gen/proto` (generated), `internal/server/rpc`,
   `internal/server/server.go`, `internal/appconfig`.
4. **Orval Zod + dashboard/CLI consumption.** Extend
   `orval.config.ts` with a second output block emitting Zod schemas.
   Wire the generated client into `js/apps/dashboard` (read-only
   "Projects" route) and `js/apps/cli` (`project list`, `flag list`,
   `snapshot latest` commands). Owns: `js/packages/generated-client-ts`,
   `js/apps/dashboard`, `js/apps/cli`.
5. **Contract tests + Hurl expansion.** Go contract test exercising
   REST↔Connect parity. Six new Hurl files covering new endpoints and
   a Connect-over-JSON smoke. Owns: `internal/server/contract_test.go`,
   `tests/hurl/04-*.hurl` through `09-*.hurl`.
6. **Validation ladder + close-out.** Add `make generate-check` and
   `make contract-test` Makefile targets, exercise the full
   validation ladder, update METAPLAN status notes, flip plan status
   to `completed`. Owns: `Makefile`, `docs/METAPLAN.md`, this plan.

Total expected commit count: ~20, matching the slice 1 cadence
(slice 2 bundled to ~6; slice 3 has more surface area).

## Technical Approach

### Architecture

```
                  HTTP :8080  (REST, oapi-codegen)
                       │
                       ▼
              ┌────────────────────┐
              │  http.ServeMux #1  │
              │  openapi.Handler   │
              └─────────┬──────────┘
                        │
                        ▼
              ┌────────────────────────┐
              │ internal/server/       │
              │   handlers (REST)      │  ◄──┐
              └─────────┬──────────────┘     │
                        │                    │ shares
                        │                    │
              ┌─────────▼──────────────┐     │
              │ internal/server/rpc/   │  ◄──┤
              │   (ConnectRPC)         │     │
              └─────────┬──────────────┘     │
                        │                    │
                        ▼                    │
              ┌────────────────────────┐     │
              │ internal/store         │  ◄──┘
              │   (pgxpool + sqlc)     │
              └─────────┬──────────────┘
                        │
                        ▼
              ┌────────────────────────┐
              │   Postgres             │
              │   projects, flags,     │
              │   flag_versions,       │
              │   environments,        │
              │   segments,            │
              │   snapshots,           │
              │   audit_events         │
              └────────────────────────┘
                       ▲
                       │
                  HTTP/2 :8090  (Connect/gRPC, buf+connect-go)
                       │
              ┌────────────────────┐
              │  http.ServeMux #2  │
              │  connect.Handler   │
              └────────────────────┘
```

Both servers run from a single `cmd/falseflag-api` binary inside
`Server.Run(ctx)` using `errgroup`. Either listener failing tears down
the whole process. Migrations still run once on startup before either
listener binds.

#### Resource model (the load-bearing contract)

The new resources, with their natural keys and FK shape:

```
projects (existing)
  ├── environments (new)
  │     PK id, FK project_id, UNIQUE (project_id, slug)
  │     fields: id, project_id, slug, name, created_at
  │
  ├── segments (new)
  │     PK id, FK project_id, UNIQUE (project_id, key)
  │     fields: id, project_id, key, name, description,
  │             predicate jsonb (an IR Predicate, validated on
  │             write), created_at, updated_at
  │
  ├── snapshots (new)
  │     PK id, FK project_id, FK environment_id (nullable),
  │     UNIQUE (project_id, version)
  │     fields: id, project_id, environment_id, version,
  │             compiled jsonb (shape: {flags: {[key]: RulesTree}}),
  │             created_at
  │
  ├── flags (existing)
  └── flag_versions (existing)

audit_events (existing, extended)
  + actor text NULL  (populated from X-Actor request header; demo stub)
```

**`UNIQUE (project_id, slug)`** and **`UNIQUE (project_id, key)`** are
project-scoped uniqueness. Same slug across two projects is allowed and
the dashboard's URL space (`/projects/<slug>/environments/<envSlug>`)
relies on this. Both `slug` and `key` reuse the existing
`^[a-z0-9][a-z0-9-]{0,62}$` pattern via OpenAPI parameter constraints
and a Go-side `internal/idstr.IsSlug(s)` helper called from handlers.

**`segments.predicate`** is an IR `Predicate` JSON, validated at
write time using the existing `internal/config` predicate validator
(extracted from `validateTreeWith`). When a flag version references a
segment via `{kind: "segment", key: "beta-users"}`, the segment is
**resolved inline at compile time** (the publish handler looks up the
segment predicate and substitutes it into the rule tree). This means
deleting a segment later does **not** break already-published flag
versions — they hold the resolved predicate. The cost is that flag
versions don't reflect later edits to the segment until republished;
this is acceptable for demo and called out in the docs.

**`snapshots.compiled`** is `{"flags": {"<key>": <RulesTree>, ...}}`.
Empty projects produce `{"flags": {}}` (HTTP 201, never 5xx).
Snapshot `version` is a per-project monotonic counter assigned via the
same serializable-transaction pattern `PublishFlagVersion` uses.
`environment_id` is nullable in slice 3 — environments are reference
data, not yet a scoping dimension. Slice 4 will repurpose this column
when the operator reconciles per-environment overrides.

#### REST surface extension (api/openapi/openapi.yaml v0.3.0)

| Method | Path | Op ID | Notes |
|---|---|---|---|
| GET  | `/v1/projects/{slug}/environments` | `ListEnvironments` | |
| POST | `/v1/projects/{slug}/environments` | `CreateEnvironment` | 201 / 409 / 400 |
| GET  | `/v1/projects/{slug}/environments/{envSlug}` | `GetEnvironment` | 200 / 404 |
| GET  | `/v1/projects/{slug}/segments` | `ListSegments` | |
| POST | `/v1/projects/{slug}/segments` | `CreateSegment` | 201 / 409 / 400 |
| GET  | `/v1/projects/{slug}/segments/{segKey}` | `GetSegment` | 200 / 404 |
| PUT  | `/v1/projects/{slug}/segments/{segKey}` | `UpdateSegment` | 200 / 404 / 400 |
| POST | `/v1/projects/{slug}/snapshots` | `CompileSnapshot` | 201, body `{environment_slug?: string}` |
| GET  | `/v1/projects/{slug}/snapshots` | `ListSnapshots` | newest-first, default limit 50 |
| GET  | `/v1/projects/{slug}/snapshots/latest` | `GetLatestSnapshot` | 200 / 404 |
| GET  | `/v1/projects/{slug}/snapshots/{id}` | `GetSnapshot` | UUID param, 200 / 404 |
| GET  | `/v1/projects/{slug}/audit-events` | `ListAuditEvents` | filters: `action`, `actor`, `from`, `to`, `cursor`, `limit` (default 100, max 1000) |
| POST | `/v1/projects/{slug}/flags/{key}/evaluate-trace` | `EvaluateFlagWithTrace` | returns `{decision, trace}` |

Plus the **existing** `PublishFlagVersion` handler is modified to call
`Store.AppendAudit(ctx, AuditEvent{Action: "publish_version", Actor, ProjectID, FlagID, Payload})`
in the same serializable transaction as the version insert. `actor` is
extracted from the `X-Actor` request header (defaults to `""` for
unauthenticated demo callers). Audit events are also appended for
`CreateEnvironment`, `CreateSegment`, `UpdateSegment`, and
`CompileSnapshot` with appropriate action strings.

**Audit search response envelope:**

```json
{
  "items": [ { "id": "...", "action": "publish_version", ... }, ... ],
  "next_cursor": "<opaque string>" | null
}
```

Cursor is the base64-encoded `created_at|id` of the last returned row.
Cursors are stateless; the SQL filter is
`(created_at, id) < (cursor_ts, cursor_id) ORDER BY created_at DESC, id DESC LIMIT N`.
Demo-quality: no signed cursors, no expiration, no rate limit.

**Evaluate-trace response shape:**

```json
{
  "decision": { "value": ..., "reason": "rule_matched", "rule_id": "us-pro", "version": 4 },
  "trace": {
    "evaluated_rules": [
      {
        "rule_id": "us-pro",
        "matched": true,
        "predicate": {
          "kind": "all",
          "result": true,
          "children": [
            { "kind": "eq", "attr": "user.country", "attr_value": "US", "expected": "US", "result": true },
            { "kind": "in", "attr": "user.plan", "attr_value": "pro", "expected_values": ["pro","enterprise"], "result": true }
          ]
        }
      }
    ],
    "default_used": false
  }
}
```

`TraceNode` and `TraceRule` are defined as new Go types in
`internal/eval/trace.go`. The trace evaluator is a separate function
(`EvaluateWithTrace`) that mirrors `Evaluate` but records every
predicate node it touches; `Evaluate` remains hot-path and untouched.
Trace evaluation always evaluates **all** rules (even after a match)
so the demo can show why earlier rules didn't match — this is
acceptable because trace is opt-in via the dedicated endpoint.

#### Proto + ConnectRPC surface (proto/falseflag/v1/)

New proto files (each with a single service grouping related RPCs):

- `projects.proto` — `ProjectsService { ListProjects, GetProject, CreateProject }`; `EnvironmentsService { ListEnvironments, GetEnvironment, CreateEnvironment }`
- `flags.proto` — `FlagsService { ListFlags, GetFlag, CreateFlag, ListFlagVersions, GetFlagVersion, PublishFlagVersion }`
- `segments.proto` — `SegmentsService { ListSegments, GetSegment, CreateSegment, UpdateSegment }`
- `snapshots.proto` — `SnapshotsService { ListSnapshots, GetSnapshot, GetLatestSnapshot, CompileSnapshot }`
- `evaluation.proto` — `EvaluationService { Evaluate, EvaluateWithTrace }`
- `audit.proto` — `AuditService { ListAuditEvents }`
- `health.proto` (existing) — `HealthService { Check }` (now actually mounted)

Proto conventions:

- `package falseflag.v1`; `go_package = "github.com/depot/falseflag/internal/gen/proto/falseflag/v1;falseflagv1"`.
- Enums: `Strategy` (`STRATEGY_UNSPECIFIED`, `STRATEGY_JSON`, `STRATEGY_CEL`, `STRATEGY_TYPESCRIPT`), `ValueType`, `DecisionReason`. Zero value is always `_UNSPECIFIED`. JSON name annotations preserved so binary and JSON encodings match OpenAPI string enums.
- Optional fields: `rule_id` on `Decision`, `actor` on `AuditEvent`, `environment_id` on `Snapshot`, `description` on `Segment`, `next_cursor` on `ListAuditEventsResponse`, all use proto3 `optional`.
- `Segment.predicate` and `RulesTree`-shaped fields use `google.protobuf.Struct` from `google/protobuf/struct.proto`. Buf managed mode handles the import; we do not vendor the well-known types.
- `Decision.value` (any JSON value) uses `google.protobuf.Value`.
- Field numbers follow the OpenAPI field order; no reservations needed yet (greenfield slice).

Generated output lands at `internal/gen/proto/falseflag/v1/` with
`paths=source_relative` (matches existing convention; differs from
the registry reference, which uses `module=` — keep our existing
layout). `buf.gen.yaml` plugins stay pinned at
`protocolbuffers/go:v1.36.10` and `connectrpc/go:v1.19.0`.

**Connect handler layout:** new package `internal/server/rpc/` with
one file per service (`projects.go`, `environments.go`, `flags.go`,
`segments.go`, `snapshots.go`, `evaluation.go`, `audit.go`,
`health.go`). Each struct embeds the corresponding
`UnimplementedXxxServiceHandler` from the generated `falseflagv1connect`
package, and holds a `*store.Store` (plus a `*slog.Logger`). All handlers
delegate to existing/new Store methods — no business logic duplication.

**Error mapping** lives in `internal/server/rpc/errors.go` as
`func connectError(err error) *connect.Error`:

| Sentinel | Connect code | HTTP code |
|---|---|---|
| `store.ErrNotFound` | `CodeNotFound` | 404 |
| `store.ErrConflict` | `CodeAlreadyExists` | 409 |
| `validation error (sentinel `errBadRequest`)` | `CodeInvalidArgument` | 400 |
| anything else | `CodeInternal` | 500 |

The REST handlers already use `notFoundOrError(w, err)` for the
first two; this slice adds `badRequest(w, err)` for the third so REST
and Connect map identically. Sentinels live in `internal/store/errors.go`.

**Second listener wiring** in `internal/server/server.go`:

```go
// pseudo, target file: internal/server/server.go
func (s *Server) Run(ctx context.Context) error {
    if err := s.migrate(ctx); err != nil { return err }
    g, gctx := errgroup.WithContext(ctx)
    g.Go(func() error { return s.runHTTP(gctx) })   // existing :8080
    g.Go(func() error { return s.runRPC(gctx) })    // new      :8090
    return g.Wait()
}
```

`appconfig.APIConfig` gains an `RPCAddr string` field (env
`FALSEFLAG_API_RPC_ADDR`, default `:8090`). `cmd/falseflag-api/main.go`
needs no signature change.

**JSON casing divergence is acknowledged.** REST handlers use Go
`encoding/json` (snake_case via struct tags); Connect's `protojson`
codec emits camelCase by default. The Hurl Connect smoke tests use
JSON encoding explicitly via the `Content-Type: application/json`
header that Connect supports. The dashboard and CLI talk to **REST
only** in slice 3 — the Connect surface is for slice 4's proxy and
slice 6's MCP. This decision is documented in `Sources & References`.

#### TypeScript client + Zod (`js/packages/generated-client-ts/`)

`orval.config.ts` grows a second output block:

```ts
// js/packages/generated-client-ts/orval.config.ts (pseudo)
export default {
  client: {
    input: { target: "../../../api/openapi/openapi.yaml" },
    output: {
      mode: "single",
      target: "./src/generated/api.ts",
      client: "fetch",
      clean: true,
      override: { mutator: undefined },
    },
  },
  zod: {
    input: { target: "../../../api/openapi/openapi.yaml" },
    output: {
      mode: "single",
      target: "./src/generated/zod.ts",
      client: "zod",
      fileExtension: ".zod.ts",
      clean: true,
    },
  },
};
```

`src/index.ts` re-exports from both files (`export * from "./generated/api.js"; export * as zod from "./generated/zod.js";`).
The Zod file imports `zod` as a dependency the package now declares.
`mode: "single"` on both output blocks prevents Orval from
restructuring the existing fetch client into per-tag files.

**Dashboard consumption** — `js/apps/dashboard/app/routes/_index.tsx`
becomes a Remix loader that fetches `listProjects()` from the generated
client and renders the list. The loader's `Response` is validated with
the Zod schema, demonstrating both outputs in one place. `@falseflag/generated-client`
is added to the dashboard's `dependencies`. `FALSEFLAG_API_BASE_URL`
env var (defaults to `http://localhost:8080`) is read in the loader.

**CLI consumption** — `js/apps/cli/src/index.ts` gains three real
commands:

- `falseflag project list` — calls `listProjects()`, prints table.
- `falseflag flag list --project <slug>` — calls `listFlags({slug})`.
- `falseflag snapshot latest --project <slug>` — calls `getLatestSnapshot({slug})`, prints JSON.

The `project` placeholder command is replaced. `@falseflag/generated-client`
is added to the CLI's `dependencies`. The CLI reads
`FALSEFLAG_API_BASE_URL` from `process.env`.

#### Audit append from handlers

The store gains a transactional variant that audit-appends in the
same transaction as the mutation it describes:

```go
// internal/store/audit.go (pseudo)
func (s *Store) WithAudit(ctx context.Context,
    event AuditEvent,
    fn func(q *db.Queries) error,
) error { /* BEGIN; fn; AppendAudit; COMMIT */ }
```

Handlers wrap their mutation:

```go
// internal/server/handlers/flags.go (pseudo)
err := s.WithAudit(ctx, store.AuditEvent{
    Action: "publish_version",
    Actor:  actorFromRequest(r),
    ProjectID: uuid.NullUUID{UUID: project.ID, Valid: true},
    FlagID:    uuid.NullUUID{UUID: flag.ID,    Valid: true},
    Payload:   marshalJSON(map[string]any{"version": newVersion.Version}),
}, func(q *db.Queries) error {
    /* existing version-insert logic */
    return nil
})
```

`actorFromRequest(r)` reads `X-Actor` from request headers. The same
helper is used by the Connect handlers via `connect.Request.Header()`.

### Implementation Phases

#### Phase 0: Dependency adds + appconfig (1 commit)

Add `golang.org/x/sync/errgroup` and `google.golang.org/protobuf/types/known/structpb`
to `go.mod`. Add `RPCAddr` to `appconfig.APIConfig` with env
`FALSEFLAG_API_RPC_ADDR` (default `:8090`). Add `actorFromRequest(r *http.Request) string`
helper to `internal/server/handlers/handlers.go`. No behavior change yet.

Acceptance: `go build ./cmd/...` + `go vet ./...` pass.

#### Phase 1: Schema + store extensions (~5 commits)

Create `db/migrations/0003_environments_segments_snapshots.sql`:

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS environments (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    slug        text        NOT NULL,
    name        text        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, slug)
);

CREATE TABLE IF NOT EXISTS segments (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    key         text        NOT NULL,
    name        text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    predicate   jsonb       NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, key)
);

CREATE TABLE IF NOT EXISTS snapshots (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      uuid        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    environment_id  uuid        REFERENCES environments(id) ON DELETE SET NULL,
    version         integer     NOT NULL,
    compiled        jsonb       NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (project_id, version)
);

CREATE INDEX IF NOT EXISTS snapshots_project_id_created_at_idx
    ON snapshots (project_id, created_at DESC);

ALTER TABLE audit_events ADD COLUMN IF NOT EXISTS actor text;
CREATE INDEX IF NOT EXISTS audit_events_action_idx ON audit_events (action);
CREATE INDEX IF NOT EXISTS audit_events_actor_idx ON audit_events (actor);

-- +goose Down
DROP INDEX IF EXISTS audit_events_actor_idx;
DROP INDEX IF EXISTS audit_events_action_idx;
ALTER TABLE audit_events DROP COLUMN IF EXISTS actor;
DROP INDEX IF EXISTS snapshots_project_id_created_at_idx;
DROP TABLE IF EXISTS snapshots;
DROP TABLE IF EXISTS segments;
DROP TABLE IF EXISTS environments;
```

Add SQLC query files:

- `db/queries/environments.sql` — `CreateEnvironment :one`, `GetEnvironment :one` (by project_id+slug), `ListEnvironments :many` (by project_id, ordered by created_at).
- `db/queries/segments.sql` — `CreateSegment :one`, `GetSegment :one`, `UpdateSegment :one`, `ListSegments :many`.
- `db/queries/snapshots.sql` — `CreateSnapshot :one`, `GetSnapshot :one`, `GetLatestSnapshot :one`, `ListSnapshots :many`, `NextSnapshotVersion :one`.
- `db/queries/audit.sql` — extend the existing `AppendAuditEvent` to include `actor`; add `ListAuditEvents :many` with the cursor filter described above. Move queries from `flags.sql` into a dedicated `audit.sql` if it makes the diff cleaner; keep file count manageable.

Add store methods mirroring the queries; add `store.WithAudit` transactional helper; add `store.ErrNotFound`, `store.ErrConflict` sentinels (and update existing store methods to return them — replace ad-hoc `errors.Is(err, pgx.ErrNoRows)` checks). Add `internal/idstr.IsSlug(s)` reuse helper.

Acceptance: `make generate && git diff --exit-code` clean; `go test ./...` passes; new `internal/store/integration_test.go` cases for environments, segments, snapshots, audit search exercise live compose Postgres via `FALSEFLAG_TEST_DATABASE_URL` (mirrors slice 2 pattern).

#### Phase 2: OpenAPI + REST handlers + audit wiring (~5 commits)

Bump `api/openapi/openapi.yaml` to `v0.3.0`. Add 14 new operations
and corresponding schemas: `Environment`, `EnvironmentList`,
`CreateEnvironmentRequest`, `Segment`, `SegmentList`,
`CreateSegmentRequest`, `UpdateSegmentRequest`, `Snapshot`,
`SnapshotList`, `CompileSnapshotRequest`, `AuditEvent`,
`AuditEventList`, `EvaluateTraceResponse`, `TraceNode`, `TraceRule`.
Reuse `Predicate` shape (introduce as a recursive `oneOf` schema —
acceptable for demo, oapi-codegen and Orval both handle it).

Add `EvaluateWithTrace(compiled, ctx, version) (Decision, Trace)` in
`internal/eval/trace.go`. Existing `Evaluate` stays untouched.

Implement new `ServerInterface` methods in
`internal/server/handlers/`. Split into per-resource files
(`environments.go`, `segments.go`, `snapshots.go`, `audit.go`,
`evaluate_trace.go`) — keep `handlers.go` for the struct, lifecycle,
and shared helpers (under ~150 lines). Wire `Store.WithAudit` from
`PublishFlagVersion`, `CreateEnvironment`, `CreateSegment`,
`UpdateSegment`, `CompileSnapshot`.

Snapshot compile logic: in a transaction, list all flags for the
project + their latest versions; for each flag with at least one
published version, copy `version.Compiled` into the snapshot's
`{flags: {...}}` map; assign `version = NextSnapshotVersion(project_id)`;
insert; append audit. Empty project → `{flags: {}}` + 201.

Acceptance: `make generate && git diff --exit-code` clean; `go test ./...`
passes; manually invoking each new endpoint with `curl` returns
plausible JSON.

#### Phase 3: Proto + ConnectRPC handlers + second listener (~5 commits)

Author the 6 new `.proto` files. Run `make generate-go`; confirm
output lands under `internal/gen/proto/falseflag/v1/` and the connect
stubs land under `falseflagv1connect/`.

Create `internal/server/rpc/` package. One handler per service. Each
handler is a thin translation layer: convert `connect.Request[*pb.X]`
to the same args the REST handler uses, call `Store`, convert the
returned domain type into the proto message via a helper in
`internal/server/rpc/convert.go`.

`internal/server/server.go` grows `runHTTP` and `runRPC` methods
wrapped in `errgroup.WithContext`. `runRPC` builds a second
`*http.Server` with `Handler: rpcMux`, `Protocols: &http.Protocols{Http1: true, UnencryptedHTTP2: true}`,
mounts each `NewXxxServiceHandler(svc)` on `rpcMux`, listens on
`cfg.API.RPCAddr`. Graceful shutdown plumbed identically to the HTTP
listener (10s timeout from the existing pattern).

Acceptance: `make generate && git diff --exit-code` clean; new
`internal/server/rpc/health_test.go` constructs a Connect client
in-process and asserts `Check()` returns `SERVING`; `go build ./cmd/...`
links the new package.

#### Phase 4: Orval Zod + dashboard/CLI consumption (~3 commits)

Extend `orval.config.ts` with the Zod output block. Add `zod` to
`js/packages/generated-client-ts/package.json` dependencies. Run
`pnpm --dir js/packages/generated-client-ts generate` and commit the
regenerated `src/generated/api.ts` + `src/generated/zod.ts`.
`src/index.ts` re-exports both. The package's `tsconfig.build.json`
must include `src/generated/zod.ts`.

Add `@falseflag/generated-client` to `js/apps/dashboard/package.json`
and rewrite `app/routes/_index.tsx` as a Remix loader that fetches
`listProjects()` and renders the project list (server-side). Run
`pnpm --dir js install` to update the lockfile (this is a main-thread
file).

Add `@falseflag/generated-client` to `js/apps/cli/package.json` and
implement the three real Commander subcommands. Existing tests under
`js/apps/cli/src/index.test.ts` extended to cover the new commands
with a `nock`-style fetch mock.

Acceptance: `pnpm --dir js -r typecheck` clean; `pnpm --dir js -r test`
passes; `pnpm --dir js -r build` builds the dashboard SSR bundle
(Remix Vite); `pnpm --dir js lint` clean.

#### Phase 5: Contract tests + Hurl expansion (~3 commits)

`internal/server/contract_test.go` boots the full `Server` in-process
(REST + Connect), seeds a project, then for each parity operation
calls REST via `httptest.NewServer(srv.HTTPHandler())` and Connect via
an in-process `falseflagv1connect.NewXxxServiceClient(httpClient, baseURL)`
where `baseURL` points at the live Connect mux. Asserts the returned
domain values are equal. Parity surface:

- list+create project
- create environment, list environments
- create segment, get segment
- create flag, publish version, get latest version
- evaluate flag
- compile snapshot, get latest snapshot
- list audit events (assert publish_version row is present)

The test runs against compose Postgres (gated on
`FALSEFLAG_TEST_DATABASE_URL`, mirrors `internal/store/integration_test.go`).

Add Hurl files:

- `tests/hurl/04-environments.hurl` — create, list, get, duplicate-slug 409.
- `tests/hurl/05-segments.hurl` — create, list, get, update, invalid-predicate 400.
- `tests/hurl/06-snapshots.hurl` — empty-project compile (201, `{"flags":{}}`), publish a flag, recompile, get latest, get by id, list, version-monotonic assert.
- `tests/hurl/07-audit.hurl` — list after publish_version, filter by action, filter by actor (after a PUT with X-Actor header).
- `tests/hurl/08-evaluate-trace.hurl` — evaluate-trace for the same fixtures slice 2's `03-evaluate.hurl` uses, asserts `trace.evaluated_rules` array shape and `default_used` boolean.
- `tests/hurl/09-connect-smoke.hurl` — `POST /falseflag.v1.HealthService/Check` with Connect-over-JSON, assert `serving_status == "SERVING_STATUS_SERVING"`; `POST /falseflag.v1.ProjectsService/ListProjects` with empty body, assert HTTP 200 and JSON shape.

`scripts/smoke.sh` truncates the new tables (`environments`,
`segments`, `snapshots`) and `audit_events.actor` column updates need
no changes. Hurl glob `tests/hurl/*.hurl` already covers the new
files; verify lexical ordering keeps 03 → 04 → ... → 09 consistent.

Acceptance: `make smoke` passes locally with compose Postgres up;
contract test passes when `FALSEFLAG_TEST_DATABASE_URL` set.

#### Phase 6: Validation ladder + close-out (~2 commits)

Add Makefile targets:

```
generate-check: ## Regenerate everything and fail if the tree is dirty
    $(MAKE) generate
    git diff --exit-code

contract-test: ## REST↔Connect contract parity test
    FALSEFLAG_TEST_DATABASE_URL=$${FALSEFLAG_TEST_DATABASE_URL:?set this} \
        go tool gotestsum --format pkgname -- ./internal/server/...
```

Run the full ladder: `go build ./cmd/...`, `go vet ./...`,
`go test ./...`, `make generate-check`, `make contract-test`,
`pnpm --dir js -r typecheck`, `pnpm --dir js -r test`,
`pnpm --dir js -r build`, `pnpm --dir js lint`, `make smoke`,
`make bake-print`.

Update `docs/METAPLAN.md` Implementation Checklist and Status Notes
(quote ladder results verbatim, mirror slice 2's status-note style).
Flip this plan's frontmatter `status` from `active` → `completed`.

Acceptance: every command above exits 0; METAPLAN updated; this plan
marked completed; ready to start slice 4.

## Alternative Approaches Considered

| Alternative | Why rejected |
|---|---|
| Connect on the same `:8080` mux behind a path prefix | The registry reference uses two ports; the proxy/operator/MCP all want a dedicated RPC port; path-prefix mounting collides with REST routes if any future endpoint shares a path stem. |
| Buf-validate annotations for proto-level field validation | Adds a `buf.build/bufbuild/protovalidate` dep and a generated validator. Demo quality doesn't need it — handler-level `IsSlug` + sentinels are enough. Add in a later slice if useful. |
| Resolve segment references at evaluate time (dynamic lookup) | Coupling evaluation to segment table state means deleting a segment silently breaks every flag that referenced it. Inline-at-publish-time matches the slice-2 "compile to IR" mental model. |
| Per-environment flag overrides in slice 3 | The data model implies a `flag_environment_overrides` table that the operator (slice 4) is the natural author of. Adding it here forces decisions about precedence and reconciliation that should be slice 4's. |
| Snapshot diffing endpoint (`GET /snapshots/{id}/diff/{other_id}`) | Useful but not load-bearing for the demo. Defer to slice 8 polish. |
| Audit events written to a separate stream (Kafka/NATS) | Not justified at this scale. Postgres `audit_events` + index is enough for the demo. |
| Streaming RPC for snapshot subscription (`WatchSnapshots`) | This is slice 4's job (proxy real-time pull). Adding it here leaks operator concerns into the API slice. |
| One monolithic `ControlPlaneService` proto | Splitting per resource makes the proto files readable, scopes service auth (when added), and matches the REST tag grouping. Monolithic was rejected for cohesion. |
| Generate Zod from proto instead of OpenAPI | Tooling support is weaker (`protobuf-es-zod` is nascent). Orval Zod from OpenAPI is the AGENTS.md-blessed path. |
| Replace fetch client with TanStack Query Orval client | The dashboard is a Remix loader (server-side); fetch is fine. Adding TanStack Query is a slice-5 concern. |

## System-Wide Impact

### Interaction Graph

```
HTTP POST /v1/projects/foo/flags/x/evaluate-trace
  → handlers.EvaluateFlagWithTrace
      → store.GetProjectBySlug
      → store.GetFlagByKey
      → store.GetLatestFlagVersion
      → config.Compile(strategy, version.Compiled)
      → eval.EvaluateWithTrace(compiled, ctx, version.Version)
          → for each rule:
              walks predicate tree, records every kind/attr/result
          → returns (Decision, Trace)
      → JSON-encode {decision, trace}

gRPC POST /falseflag.v1.SnapshotsService/CompileSnapshot
  → rpc.SnapshotsService.CompileSnapshot
      → store.WithAudit(WithTransaction):
          → store.ListFlagsByProject
          → store.ListLatestFlagVersionsByProject  (new SQLC query)
          → assemble {flags: {key: RulesTree}}
          → store.NextSnapshotVersion
          → store.CreateSnapshot
          → store.AppendAudit (action="compile_snapshot")
      → convert.SnapshotToProto
      → return connect.Response[*pb.Snapshot]

REST or Connect mutation:
  → handler -> store.WithAudit -> BEGIN, mutation, AppendAudit, COMMIT
  → on error: rollback, return ErrConflict / ErrNotFound / errBadRequest
  → REST: notFoundOrError / badRequest -> HTTP code
  → Connect: connectError -> connect.Code
```

### Error & Failure Propagation

| Surface | Layer | Behavior |
|---|---|---|
| REST | handler → store → pgx | `pgx.ErrNoRows` → `store.ErrNotFound` → `notFoundOrError(w, err)` → 404 |
| REST | handler → store → pgx | unique violation → `store.ErrConflict` → 409 |
| REST | handler decode | JSON / validation failure → `errBadRequest` → `badRequest(w, err)` → 400 |
| REST | handler → eval | `eval.ErrTypeMismatch` → 200 + Decision with reason=`type_mismatch` (matches slice 2) |
| Connect | handler → store | same sentinels → `connectError(err)` → matching `connect.Code*` |
| Connect | handler decode | proto validation (nil deref check) → `CodeInvalidArgument` |
| Connect | server | listener bind failure → errgroup cancellation → process exit |
| Connect | request | `Content-Type: application/grpc-web` / `application/json` / `application/connect+json` all accepted (connect-go default) |
| Snapshot | compile | flag has no published version → flag is skipped (not error) |
| Snapshot | compile | request times out > 10s → `context.DeadlineExceeded` → 504 (REST) / `CodeDeadlineExceeded` (Connect) |
| Trace | evaluate | CEL error in predicate → trace node `{kind:"cel", result:false, error:"..."}`; eval falls through to next rule (does not 500) |

### State Lifecycle Risks

| Step | Risk | Mitigation |
|---|---|---|
| Goose migration 0003 | Statement order: snapshots references environments | Single file with environments → segments → snapshots ordering, verified by goose up-to + down test |
| `Store.WithAudit` | Partial failure between mutation and audit append | Both inside same `BEGIN/COMMIT`; rollback if either fails |
| Snapshot compile | Concurrent compiles producing same version | Serializable isolation level (matches `PublishFlagVersion` pattern); `UNIQUE (project_id, version)` is the backstop |
| Segment delete | Orphaned references in published flag versions | None — predicates are inlined at publish; deletion does not affect already-compiled versions. UI should warn but slice 5's problem. |
| Orval regen | Drift between dashboard imports and generated names | `generate-check` Makefile target enforces idempotency; `pnpm typecheck` catches name drift at build time. |
| Connect listener | Listener fails after migrations run | `errgroup.WithContext` cancels REST listener too; process exits non-zero; supervisor restart picks it up. |
| Audit cursor | Cursor lifetime | Stateless `created_at|id`; valid until the row is deleted. No expiration needed for demo. |
| Schema column add | `audit_events.actor` is NULL-able | Existing rows backfill to NULL; queries treat NULL as "no actor" — no migration needed for existing audit rows. |

### API Surface Parity

- **cmd/falseflag-api/main.go** — unchanged; the second listener is internal to `internal/server`.
- **cmd/falseflag-proxy/main.go** — unchanged; proxy starts consuming the new ConnectRPC snapshot endpoints in slice 4 (this slice ships the contract only).
- **operator/api/v1alpha1/** — unchanged; slice 4 will add CRDs that reconcile through the new ConnectRPC services.
- **internal/server/handlers/handlers.go** — extended (new methods on `API` struct, helpers shared with rpc package).
- **internal/server/rpc/** — new package.
- **internal/server/server.go** — extended (`runHTTP`, `runRPC`, second listener).
- **internal/store/** — extended (new methods, `WithAudit`, sentinels).
- **internal/eval/** — extended (`trace.go` new; `eval.go` untouched).
- **internal/config/** — extended (predicate validator extracted to shared helper; segment-reference resolver added).
- **api/openapi/openapi.yaml** — bumped to `v0.3.0`, new operations added, existing operations untouched.
- **proto/falseflag/v1/** — new files added alongside `health.proto`.
- **js/packages/generated-client-ts/** — regenerated with new operations + Zod output.
- **js/packages/sdk-js/** — **unchanged**; the SDK does local IR evaluation and does not consume the control-plane Connect surface.
- **js/apps/dashboard/** — consumes generated client for the read-only Projects route.
- **js/apps/cli/** — consumes generated client for `project list`, `flag list`, `snapshot latest`.

### Integration Test Scenarios

1. **REST↔Connect parity for the demo happy path.** Create project via REST; list projects via Connect; assert the new project appears. Create flag via Connect; publish version via REST; assert via REST that latest version matches what was published via Connect.
2. **Snapshot compile races publish.** Publish a flag version concurrently with snapshot compile; assert snapshot version is monotonically increasing across both successful operations; assert no row violates `UNIQUE (project_id, version)`.
3. **Segment inlining at publish.** Create segment `beta-users` with predicate `{kind: "eq", attr: "user.beta", value: true}`. Publish a flag whose source references `{kind: "segment", key: "beta-users"}`. Delete the segment. Re-fetch the flag version: the stored `compiled` IR should contain the **inlined** predicate, not the segment reference. Evaluate succeeds.
4. **Audit search filter combinations.** Publish 5 flag versions with `X-Actor: alice`, 3 with `X-Actor: bob`. Filter by `actor=alice` returns 5; filter by `action=publish_version` returns 8; filter by both returns 5; filter by time window in the past returns 0.
5. **Evaluate-trace consistency.** Run the same evaluation through `Evaluate` and `EvaluateWithTrace`. Decision values and reasons must match byte-for-byte.

## Acceptance Criteria

### Functional Requirements

- [ ] Migration `0003_environments_segments_snapshots.sql` applies cleanly on a fresh DB and rolls back without leaving stray objects.
- [ ] All 14 new REST operations return correct shapes on their happy-path Hurl tests.
- [ ] Every new REST operation has a Connect RPC sibling with parity semantics (contract test passes).
- [ ] `AppendAudit` is called from `PublishFlagVersion`, `CreateEnvironment`, `CreateSegment`, `UpdateSegment`, and `CompileSnapshot` — verified by `audit-events` listing after a flag publish in Hurl 07.
- [ ] `X-Actor` request header populates the `audit_events.actor` column (Hurl 07 asserts).
- [ ] Snapshot compile on an empty project returns `201` with `{"compiled": {"flags": {}}}` (Hurl 06 asserts).
- [ ] Snapshot version is monotonically increasing per project (Hurl 06 asserts).
- [ ] `GET /v1/projects/{slug}/audit-events` defaults to `limit=100`, caps at `1000`, returns `next_cursor` when more results exist.
- [ ] `evaluate-trace` returns a Decision identical to `evaluate` for the same input (contract test asserts).
- [ ] Segment references inline at publish; deleting the segment does not break already-compiled flag versions (integration test asserts).
- [ ] ConnectRPC HealthService is mounted and `Check()` returns `SERVING_STATUS_SERVING` via Hurl 09.
- [ ] Dashboard `/` route renders a project list fetched via the generated client (live API required, otherwise returns 503 from the loader).
- [ ] CLI `falseflag project list`, `flag list`, `snapshot latest` print product-shaped output against the live API.

### Non-Functional Requirements

- [ ] `make generate-check` exits 0 on a clean tree (idempotent across two consecutive runs).
- [ ] `make contract-test` exits 0 with `FALSEFLAG_TEST_DATABASE_URL` set; skipped otherwise.
- [ ] No `pkg/**` directory created.
- [ ] No `.ts` / `.tsx` files outside `js/**`.
- [ ] No new dependency that competes with the AGENTS.md stack (no zap, zerolog, gin, echo, etc.).
- [ ] `cmd/falseflag-api/main.go` stays under 50 lines.
- [ ] Snapshot compile completes inside `context.WithTimeout(r.Context(), 10*time.Second)`.
- [ ] Each proto file uses `STRATEGY_UNSPECIFIED`-style enum zero values; proto3 `optional` used for nullable scalar fields.
- [ ] `buf lint proto/falseflag/v1/` exits 0.

### Quality Gates

- [ ] `go build ./cmd/...`
- [ ] `go vet ./...`
- [ ] `go test ./...`
- [ ] `FALSEFLAG_TEST_DATABASE_URL=… go test ./internal/store/... ./internal/server/...`
- [ ] `make generate && git diff --exit-code` (i.e., `make generate-check`)
- [ ] `make contract-test`
- [ ] `pnpm --dir js -r typecheck`
- [ ] `pnpm --dir js -r test`
- [ ] `pnpm --dir js -r build`
- [ ] `pnpm --dir js lint`
- [ ] `make smoke` (now exercises 9 Hurl files)
- [ ] `make bake-print`

## Success Metrics

| Metric | Target |
|---|---|
| New REST operations | 14 |
| New ConnectRPC services | 6 (Projects+Environments+Flags+Segments+Snapshots+Evaluation+Audit, grouped into 6 service definitions) |
| Contract-test parity assertions | ≥ 7 happy-path scenarios |
| New Hurl files | 6 (`04-environments`, `05-segments`, `06-snapshots`, `07-audit`, `08-evaluate-trace`, `09-connect-smoke`) |
| New DB tables | 3 (`environments`, `segments`, `snapshots`) |
| New SQLC queries | ≥ 14 |
| Orval outputs | 2 (fetch client, Zod schemas) |
| Dashboard routes consuming generated client | 1 (read-only Projects index) |
| CLI commands consuming generated client | 3 (`project list`, `flag list`, `snapshot latest`) |
| Audit-append call sites wired | 5 (`PublishFlagVersion`, `CreateEnvironment`, `CreateSegment`, `UpdateSegment`, `CompileSnapshot`) |
| Cross-runtime corpus regression | 0 — slice 2's 15-fixture corpus still passes on both runtimes |

## Dependencies & Prerequisites

| Dependency | Source | Use |
|---|---|---|
| `golang.org/x/sync` | `go get` (already indirect) | `errgroup.WithContext` for the two-listener pattern in `Server.Run`. |
| `google.golang.org/protobuf/types/known/structpb` | `go get` | Marshalling `google.protobuf.Struct` / `Value` for predicate and decision-value fields. |
| `connectrpc.com/connect` | already vendored (slice 1) | Connect handler runtime. |
| Buf plugins | `buf.gen.yaml`, pinned | `protocolbuffers/go:v1.36.10`, `connectrpc/go:v1.19.0` — unchanged. |
| Orval | already vendored (slice 1) | Second output block emits Zod from OpenAPI. |
| `zod` | new dep in `js/packages/generated-client-ts/package.json` | Validation schemas. |
| Compose Postgres | `infra/docker-compose.yml` | Integration + contract tests + `make smoke`. |
| Slice 2 IR + evaluator | `internal/config`, `internal/eval` | Trace evaluator builds on top; segment validator extracts existing logic. |

## Risk Analysis & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Proto / OpenAPI shape drift between REST and Connect | Medium | High — silent divergence breaks demo trust | Contract test asserts domain equality; converter helpers in `internal/server/rpc/convert.go` are the single translation point. |
| Orval Zod regen restructures the existing fetch client | Medium | Medium — dashboard/CLI imports break | Force `mode: "single"` on both output blocks; separate `target` paths; pin Orval version in `package.json`. |
| `google.protobuf.Struct` usage adds opaque blobs to RPC payloads | Low | Medium — Connect clients can't introspect predicate shape | Document in the README; Zod schemas remain the typed surface for TS consumers; Go consumers use the IR types directly. |
| Snapshot compile latency on a project with many CEL flags | Medium | Low — synchronous compile blocks the request | `context.WithTimeout(r.Context(), 10*time.Second)` guard; CEL programs are compiled per-source-string cache hit in the Compile path. |
| `errgroup` cancellation racing graceful shutdown | Low | Medium — orphaned listener on SIGINT | Mirror the existing 10s shutdown pattern; both listeners read from the same `gctx`. |
| Audit cursor invalidates when rows are deleted | Low | Low — caller sees fewer results | Documented; demo doesn't delete audit rows. |
| `make generate-check` fails on a developer's first run because they haven't run codegen yet | Medium | Low — confusing first contact | Document in README; CI's first step is `make generate-check`, so contributors see it before pushing. |
| Connect serving JSON-codec only vs binary-protobuf | Medium | Low | Hurl 09 uses JSON explicitly; TS consumers (slice 6 MCP) will use `@connectrpc/connect-web` with the JSON transport. Binary is supported by default but not exercised in the slice. |
| Segment validator drift from `validateTreeWith` | Low | Medium — segment predicates that publish but fail at evaluate | Extract `validateTreeWith` into `internal/config.ValidatePredicate(p, allowCEL)` and reuse from both the segment write handler and the flag publish handler. |

## Resource Requirements

- One agent, working through the phases sequentially. Phase 1 (schema/store) is foundational; once it lands, phase 2 (REST) and phase 3 (Connect) could fan out in parallel — but the slice is small enough that sequential keeps the commit log clean and avoids contention on `internal/server/server.go`.
- Compose Postgres for integration and contract tests (already running locally per slice 2).
- No new tooling installs; all generators are `go tool` directives or `pnpm` workspace scripts.

## Future Considerations

- **Real-time push** (slice 4): the proxy subscribes to snapshot changes; this slice lands the snapshot pull surface but not SSE/WebSocket/gRPC streaming push.
- **Per-environment flag overrides** (slice 4): the `environments` table is populated by this slice; the operator owns the `flag_environment_overrides` join concept.
- **Segment versioning** (later slice): segments are currently mutable in place; an immutable-version model analogous to `flag_versions` is nice-to-have but not load-bearing.
- **Snapshot diffing** (slice 8 polish): `GET /v1/projects/{slug}/snapshots/{id}/diff/{other_id}` returns a per-flag added/removed/changed delta. Useful in the demo script, deferred until the polish slice.
- **Audit event ingestion from outside the API** (slice 4): the operator's reconciliation events would be a natural source. Out of scope here.
- **Auth on either surface** (later slice): even a stub bearer-token middleware is deferred. `X-Actor` is purely an actor-attribution header, not an authentication signal.
- **Buf-validate** annotations on proto fields: not wired this slice; handler-level validation suffices for demo quality.
- **`golangci-lint` end-to-end**: still open from slice 1; not in scope here.

## Documentation Plan

- Update `docs/METAPLAN.md`:
  - Tick the three slice-3 checklist items.
  - Append a status note in the slice-2 style: commands run, gaps deferred, next recommended step.
- Update `README.md` if it has a "what works today" section to mention the Connect port, the new resources, and the CLI commands.
- Add a one-paragraph `internal/server/rpc/README.md` explaining the package's role as a thin translation layer over `internal/store`.
- Add a one-paragraph `proto/falseflag/v1/README.md` documenting proto conventions (enum zero values, optional fields, `Struct`/`Value` use).
- Flip this plan's frontmatter `status: active` → `status: completed` in the close-out commit.

## Sources & References

### Internal References

- Slice 1 plan: `docs/plans/2026-05-20-001-feat-foundation-monorepo-scaffold-plan.md` — established `cmd/**`, `internal/**`, generated-artifact layout.
- Slice 2 plan: `docs/plans/2026-05-20-002-feat-configuration-strategies-plan.md` — `internal/config` IR, `internal/eval` evaluator, `flag_versions` and `audit_events` schema.
- Ideation: `docs/ideation/2026-05-20-synthetic-feature-flag-platform-depot-demo-ideation.md` — names environments, segments, snapshots, and the proxy/operator/MCP RPC contract use cases.
- Master plan: `docs/METAPLAN.md` §"Recommended ce-plan Sequence > 3. API, gRPC, and OpenAPI Plan", §"Quality Bar", §"Validation Ladder".
- Current OpenAPI: `api/openapi/openapi.yaml:0.2.0`, `api/openapi/cfg.yaml`.
- Current proto: `proto/falseflag/v1/health.proto` and generated `internal/gen/proto/falseflag/v1/falseflagv1connect/health.connect.go`.
- Current handlers: `internal/server/handlers/handlers.go`, `flags.go`, `projects.go` (split across files post-phase-2).
- Current store: `internal/store/store.go`, `flags.go`, `projects.go`, `audit.go`, `migrations.go`.
- Current Orval config: `js/packages/generated-client-ts/orval.config.ts`.
- Registry reference (read-only): `/Users/wito/code/project-depot/registry/internal/server/server.go` — two-listener `errgroup` pattern.

### External References

- ConnectRPC Go docs: [connectrpc.com/docs/go/getting-started](https://connectrpc.com/docs/go/getting-started)
- Buf managed mode: [buf.build/docs/configuration/v2/buf-gen-yaml#managed-mode](https://buf.build/docs/configuration/v2/buf-gen-yaml)
- Orval Zod client: [orval.dev/reference/configuration/output#client](https://orval.dev/reference/configuration/output#client)
- Proto3 `optional` fields: [protobuf.dev/programming-guides/proto3/#field-labels](https://protobuf.dev/programming-guides/proto3/#field-labels)
- `google.protobuf.Struct`: [protobuf.dev/reference/protobuf/google.protobuf/#struct](https://protobuf.dev/reference/protobuf/google.protobuf/)
- OpenFeature resolution reasons (we mirror the strings): [openfeature.dev/specification/sections/evaluation-api](https://openfeature.dev/specification/sections/evaluation-api)

### Related Work

- Slice 4 (Kubernetes operator and CRDs): will reconcile against the ConnectRPC surface this slice lands.
- Slice 5 (Dashboard, CLI, SDKs): will deepen the dashboard usage of the generated client and add Playwright coverage.
- Slice 6 (MCP server): will consume `falseflagv1connect` clients via `@connectrpc/connect-node` from the Go MCP binary.
- Slice 7 (CI / Depot): will adopt `make generate-check` and `make contract-test` as CI gates.
