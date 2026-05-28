---
title: Slice 10 handoff — SQLite backend, phases 3–8
type: handoff
status: complete
date: 2026-05-27
companion: 2026-05-27-001-feat-sqlite-backend-plan.md
---

# Slice 10 handoff (SQLite backend)

You are picking up slice 10 mid-flight. Two of eight phases are
committed to `main`; the remaining six are fully specified in the
companion plan but unexecuted. Read this handoff first, then the
plan, then start Phase 3.

## Where things stand

`git log --oneline` on `main`:

```
3992ac0 refactor(store): generate UUIDs in Go; drop gen_random_uuid() defaults
8f053d9 refactor(store): DSN dispatch + drop Pool()/Queries() getters
2d8c731 docs: slice 10 plan — SQLite storage backend alongside Postgres
```

Working tree should be clean apart from a pre-existing
`M docs/METAPLAN.md` (not slice 10's problem). Branch: `main`.
Per METAPLAN/slice-9 convention, commit directly to `main`, one
phase per commit, no PRs, CI auto-triggers stay off.

### Verified green at HEAD

- `go build ./...`
- `go test ./...` (every package, no env var)
- `FALSEFLAG_TEST_DATABASE_URL="postgres://falseflag:falseflag@localhost:5432/falseflag?sslmode=disable" go test ./internal/store/... ./internal/server/...`
- `make smoke` (14/14 hurl files) against the compose stack with a
  freshly-rebuilt api image.

### What changed under the hood

- `internal/store/dsn.go` (new) parses `FALSEFLAG_DATABASE_URL` and
  returns `(Backend, driverDSN, error)`. SQLite scheme is **accepted
  by the parser** but the dispatch in `store.Open` returns
  `"sqlite backend not yet supported (slice 10 phase 4)"` — flip that
  arm of the switch when you wire `internal/store/sqlite/` in Phase 4.
- `Pool()` and `Queries()` are gone from `*store.Store`. The two test
  TRUNCATE callers (`internal/store/integration_test.go`,
  `internal/server/contract_test.go`) use a new method
  `(s *Store) TruncateForTest(ctx) error`.
- Every `Create*` store method now calls `fromUUID(uuid.New())` at
  insert time. The audit insert goes through `auditInsertParams` so
  both `AppendAudit` and the in-tx audit append in `WithAudit` share
  the helper.
- All `db/migrations/*.sql` have `PRIMARY KEY` (no default). All
  `db/queries/*.sql` `INSERT`s take `id` as the first column. sqlc
  output regenerated.

### Scope deferred from Phase 1 → Phase 4

The original Phase 1 in the plan included three things; I executed
two and deferred one. **Phase 4 must absorb the deferred work:**

- **File moves** — `internal/store/{store,audit,flags,projects,environments,segments,snapshots,migrations,sqlstd,convert,errors}.go`
  should move into `internal/store/postgres/` as part of Phase 4.
- **`store.Store` interface** — introduce when the SQLite impl
  lands. The pg impl becomes `postgres.Store` (unexported field via
  the package boundary); the public `store.Store` becomes the
  interface.
- **`WithAudit` callback narrowing** — currently
  `func(q *db.Queries) error`. Narrow to a `store.Tx` (recommended)
  or `store.Querier` interface that both backends can satisfy.

See "Open Phase 4 decision" below for the recommended `Tx` shape.

## Gotchas I hit so you don't repeat them

1. **The api container must be rebuilt for Go changes to take
   effect.** `compose.yaml`'s `api` service has `build:`, not a bind
   mount of source. After any change to migrations or Go store code,
   run `docker compose up -d --build api` before `make smoke`. The
   stale-image symptom is HTTP 500 with `null value in column "id"`
   — old code (no ID passed) talking to new schema (no default).
   Cost me one wasted smoke run.

2. **Goose remembers.** The DB's `goose_db_version` table records
   applied migrations. If you edit a migration in place (which Phase
   2 does) on a DB that already has it applied, goose won't re-run
   it — `current version: 4` and a no-op. To pick up edited
   migrations in dev, drop the schema first:
   ```bash
   docker compose exec -T db psql -U falseflag -d falseflag \
     -c "DROP TABLE IF EXISTS audit_events, snapshots, segments, environments, flag_versions, flags, projects, goose_db_version CASCADE;"
   ```
   Then `docker compose restart api` (or `up -d --build api` if Go
   changed too) re-runs migrations against the empty DB.

3. **LSP stale-cache false positives.** After regenerating sqlc, the
   in-editor diagnostics will show `unknown field ID in struct
   literal of type db.CreateXxxParams` even though `go build ./...`
   is clean. Trust the build. The diagnostics catch up after a few
   seconds.

4. **No `docs/solutions/` directory exists.** The learnings
   researcher confirmed there's no institutional knowledge base
   here. Anything worth remembering goes in the plan or in CLAUDE.md
   (which itself doesn't exist at the repo root — only personal
   `~/.claude/CLAUDE.md`).

5. **`pgtestcontainer` is a phantom.** The slice prompt mentions it
   but the actual test pattern is env-gated:
   `if FALSEFLAG_TEST_DATABASE_URL == "" { t.Skip(...) }`. No
   testcontainers dependency exists in the repo. Don't add one — for
   the SQLite half of Phase 5, just use `t.TempDir()`.

## Open Phase 4 architectural decision

The `WithAudit(ctx, ev, fn func(q *db.Queries) error)` callback is
the trickiest piece of the interface refactor. The pgx `db.Queries`
methods take `pgtype.UUID`, `pgtype.Text`, `pgtype.Timestamptz`. The
SQLite `dbsqlite.Queries` will take `uuid.UUID`/`string`/`time.Time`
(or pointer variants with `emit_pointers_for_null_types: true`).
**Their method signatures don't line up**, so the obvious "shared
interface satisfied by both `Queries` types" is impossible.

**Recommended shape (the `Tx` approach):** define an interface in
`internal/store/` whose methods are *high-level store operations*,
not low-level sqlc methods:

```go
package store

type Tx interface {
    PublishFlagVersion(ctx context.Context, p PublishFlagVersionParams) (FlagVersion, error)
    AppendAudit(ctx context.Context, p AppendAuditParams) (AuditEvent, error)
    // add more as call sites demand — keep it minimal
}

type Store interface {
    // ... all existing exported methods ...
    WithAudit(ctx context.Context, ev AppendAuditParams, fn func(Tx) error) error
}
```

Each backend impl supplies its own `txImpl` that wraps the
txn-scoped `*db.Queries` (pg) or `*dbsqlite.Queries` (sqlite) and
does the type conversion inside. Callers in `internal/server/{rpc,handlers}`
change from:

```go
h.store.WithAudit(ctx, ev, func(q *db.Queries) error {
    v, err := h.store.PublishFlagVersionTx(ctx, q, params)
    return err
})
```

to:

```go
h.store.WithAudit(ctx, ev, func(tx store.Tx) error {
    v, err := tx.PublishFlagVersion(ctx, params)
    return err
})
```

That eliminates `PublishFlagVersionTx` as a top-level Store method —
it's now `Tx.PublishFlagVersion`. The non-tx `PublishFlagVersionStandalone`
collapses into a single `Store.PublishFlagVersion` that opens its own
tx internally. This is a real ergonomic improvement, not just a
backend-routing concession.

Grep `WithAudit` callers (~14 call sites under `internal/server/`)
when implementing; most callbacks ignore `q` entirely (they call
`h.store.CreateFlag` etc., not via `q`), so they trivially become
`func(_ store.Tx) error`. Two callers actually use `q` (the
publish-version paths in `flags.go` and `snapshots.go`); those need
the new `Tx.PublishFlagVersion` / `Tx.CompileSnapshotInTx`
methods.

## Runtime / dev cheat sheet

```bash
# Compose stack control
docker compose up -d --build api          # rebuild Go binary
docker compose restart api                # if only env/config changed
docker compose logs api --tail 30         # check what just happened
docker compose exec -T db psql -U falseflag -d falseflag -c "\d projects"

# Local Postgres for tests (already exposed by compose on :5432)
export FALSEFLAG_TEST_DATABASE_URL='postgres://falseflag:falseflag@localhost:5432/falseflag?sslmode=disable'

# Sqlc regen + builds + smoke
make generate-go
go build ./...
go test ./...
make smoke                                # against compose stack
```

For Phase 6 you'll want a sibling `make smoke-sqlite` and a
`compose.sqlite.yaml` standalone (not an overlay — Postgres `db`
service must not start in the SQLite stack).

## Suggested Phase 3 starting moves

1. `go get modernc.org/sqlite@latest` — pin the version that ships
   SQLite ≥ 3.35 (for RETURNING). As of May 2026 the latest is
   ~`v1.50.1` bundling SQLite 3.53.1.
2. Author `db/migrations/sqlite/0001_init.sql` through
   `0004_flag_versions_source_text.sql` by hand. Don't auto-translate
   — the SQL is small enough that hand-rolling avoids "almost-right"
   type-mapping bugs. Reference table for translations is in the
   plan's Phase 3 section.
3. Author `db/queries/sqlite/*.sql`. Watch for:
   - `DISTINCT ON` in `snapshots.sql` → `ROW_NUMBER()` window in a
     subquery (rewrite shown in plan).
   - Row-value cursor `(created_at, id) < (...)` in `audit.sql` →
     expand to `created_at < ? OR (created_at = ? AND id < ?)`.
   - Drop every `::cast` — SQLite doesn't have that syntax. The
     `sqlc.narg` macro itself works in the SQLite engine.
   - `now()` in `UpdateSegment` — push to Go (`updated_at = ?`) for
     both backends, since SQLite uses `CURRENT_TIMESTAMP` and unifying
     in Go is cleaner than diverging.
   - All `$1`, `$2` → `?` placeholders (sqlc handles the param
     names; the SQL syntax differs).
4. Extend `db/migrations/migrations.go`'s `//go:embed *.sql` directive
   to cover the new subdir: `//go:embed *.sql sqlite/*.sql`. Then
   serve postgres via `fs.Sub(FS, ".")` (or equivalent) and sqlite
   via `fs.Sub(FS, "sqlite")`.
5. Extend `sqlc.yaml` with a second `sql:` block (full snippet in
   plan). Set `emit_interface: true` on **both** blocks — useful
   when designing the Phase 4 `Tx` shape.
6. `make generate-go` should now emit `internal/db/sqlite/{db,models}.go`
   plus six `*.sql.go` files. Commit the generated output.
7. No runtime is wired yet — Phase 3 ends with `go build ./...`
   clean and the new files committed. Phase 4 brings it to life.

## What "done" looks like for slice 10

The slice is complete when:

- `FALSEFLAG_DATABASE_URL=sqlite:///tmp/ff.db ./bin/falseflag-api`
  boots and serves a working API.
- `go test ./internal/store/...` runs both `t.Run("postgres",...)`
  (skipped if env unset) and `t.Run("sqlite",...)` (always runs via
  `t.TempDir()`), both green.
- `make smoke` (Postgres compose) and `make smoke-sqlite` (SQLite
  compose) both 14/14.
- `.github/workflows/ci.yml` shows `smoke (postgres)` and
  `smoke (sqlite)` as separate green jobs.
- README has the "Storage backends" section the plan specifies.
- `docs/METAPLAN.md` status note ticks slice 10.

## Pointers

- Plan: `docs/plans/2026-05-27-001-feat-sqlite-backend-plan.md`
- Earlier slice patterns: `docs/plans/2026-05-26-002-feat-slice-9-polish-demo-script-plan.md`
- METAPLAN slice 10 prompt: `docs/METAPLAN.md` lines 531–571
- Driver docs: `modernc.org/sqlite` on pkg.go.dev — bundled SQLite
  version verifiable at runtime via `SELECT sqlite_version()`
- sqlc multi-engine: `docs.sqlc.dev` configuration reference
- goose multi-dialect: `pressly/goose/v3` — use
  `goose.NewProvider(dialect, db, fs.Sub(FS, "sqlite"))`, not the
  global `goose.SetDialect`/`SetBaseFS` pair (cleaner; no shared
  global state when running both backends in the same test binary).
