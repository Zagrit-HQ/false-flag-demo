---
title: Slice 10 — SQLite storage backend alongside Postgres
type: feat
status: complete
date: 2026-05-27
---

# Slice 10 — SQLite storage backend alongside Postgres

## Overview

Slice 10 adds SQLite as a second first-class storage engine for
falseflag, sitting next to the existing Postgres backend. Every
`internal/store/*` call site picks its backend at runtime from the
DSN scheme (`postgres://` vs `sqlite://`); both backends pass the
same integration suite; CI fans every Go test job and every Hurl
smoke job out over `{postgres, sqlite}`.

This is the canonical self-hosted-tool pattern — Gitea, Vaultwarden,
Mealie, Forgejo, Paperless-ngx all carry both engines. The slice
deliberately roughly doubles CI work and codegen surface as a
realistic "slow CI to optimize" target (per METAPLAN slice 10
framing). It is **not** a production-hardening exercise; the
quality bar is demo-quality: SQLite must work end-to-end in compose
and pass the same suite as Postgres, but it does not need to be
performance-tuned, optimized for concurrent writers, or hardened
against pathological inputs.

Slice 11 (multi-arch container images) and slice 12 (deployment
artifact matrix) build on top of this slice. Slice 12 will own
backup/restore. Migration parity checking between the two schemas
is **intentionally out of scope** — both schemas drift independently
by design (METAPLAN line 570).

All work lands directly on `main`, one logical commit per phase,
matching the slice 8/9 pattern.

## Problem Statement / Motivation

Today `falseflag` only ships a Postgres path:

- `internal/store/store.go:23` holds a concrete `*pgxpool.Pool` and
  the constructor calls `pgxpool.New` unconditionally.
- `internal/store/store.go:58` exposes a `Pool()` getter; the
  integration test reaches through it to `TRUNCATE` between tests.
- `internal/db/**` is sqlc output emitting `pgtype.UUID`,
  `pgtype.Timestamptz`, `pgtype.Text` everywhere; the
  `internal/store/convert.go` layer converts them to domain types
  but only one direction is wired.
- `internal/store/audit.go:138`, `flags.go:109`, `snapshots.go:28`
  all start serializable transactions via
  `s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})`.
- `internal/store/errors.go:6,29` and `audit.go:160` check
  `pgconn.PgError` SQLSTATEs (23505, 23503, 40001) for conflict and
  retry classification.
- Every migration uses `uuid PRIMARY KEY DEFAULT gen_random_uuid()`,
  `timestamptz`, `jsonb`, `DEFAULT '{}'::jsonb`, and at least one
  `DISTINCT ON` query (`db/queries/snapshots.sql:31`).

A self-hoster running a small home-lab or a single VPS doesn't want
to babysit Postgres for a feature-flag service. The single-binary
+ SQLite shape is what they expect. From a slow-CI-baseline
standpoint, the dual-backend matrix is also the single highest-impact
way to broaden the real-world data layer that CI has to chew through:
sqlc codegen runs twice, goose migrations run twice, Go tests fan
out, and Hurl smokes run against two stacks.

The refactor cost is bounded: the pgx coupling is **entirely confined
to `internal/store/*`** (and one TRUNCATE in `integration_test.go:38`).
No handler, RPC, or downstream package touches pgx. The interface
boundary already exists in shape; the slice formalizes it.

## Proposed Solution

Eight phases, ordered so every step lands the tree in a buildable
state with both backends green for whichever ones exist at that
point.

**Phase 1 — Store refactor (Postgres-only, no behavior change).**
Drop the `Pool()` getter, introduce a `Backend` enum + DSN dispatch,
hide pgx behind unexported helpers. After this phase Postgres still
works end-to-end and the test suite is unchanged.

**Phase 2 — Move UUID generation to Go for both backends.**
Drop `gen_random_uuid()` defaults from Postgres migrations; insert
`google/uuid` IDs at the store layer. Postgres still green.

**Phase 3 — SQLite schema + queries + dual-engine sqlc config.**
Author SQLite migrations under `db/migrations/sqlite/`, queries
under `db/queries/sqlite/`, second sqlc block emitting to
`internal/db/sqlite/`. `make generate` produces both.

**Phase 4 — SQLite Store implementation.**
modernc.org/sqlite driver, goose multi-dialect, `BEGIN IMMEDIATE`
+ busy_timeout retry abstraction, backend-agnostic conflict/retry
error helpers. Wire through DSN dispatch from Phase 1.

**Phase 5 — Parametrize integration tests.**
`internal/store/integration_test.go` runs every existing test under
`t.Run("postgres"...)` and `t.Run("sqlite"...)`. Postgres uses the
existing env-gated path; SQLite uses `t.TempDir()`.

**Phase 6 — compose.sqlite.yaml + smoke-sqlite Make target.**
Compose overlay boots api with a named SQLite volume; `make smoke`
exercises Postgres, `make smoke-sqlite` exercises SQLite.

**Phase 7 — CI backend matrix.**
`backend: [postgres, sqlite]` matrix axis on `test-go`, `test-go-race`,
`contract-test`, `smoke`, `dashboard-e2e`, `kind-smoke`.

**Phase 8 — Documentation.**
README "single binary + SQLite" vs "compose + Postgres" section;
update demo walkthrough; tick METAPLAN.

Phases 1–7 are independently committable. The tree is buildable
and the test suite (for whatever backend(s) exist at that point) is
green after each commit.

## Phase 1 — Store refactor (Postgres-only)

**Goal:** the public store surface picks a backend from the DSN
scheme; no caller outside `internal/store/` ever sees `*pgxpool.Pool`,
`pgtype.*`, or `pgconn.PgError`. Postgres-only at this stage —
sqlite scheme returns "not yet supported" until Phase 4.

**Files:**

- `internal/store/store.go` — rename existing `Store` struct to
  unexported `pgStore`; introduce exported `Store` interface (with
  the union of currently exported methods); introduce `Open(ctx, dsn)`
  that parses scheme and dispatches.
- `internal/store/dsn.go` *(new)* — `parseBackend(dsn) (Backend, string, error)`
  matching `postgres://`, `postgresql://`, `sqlite://`, `file:` and
  returning `(BackendPostgres|BackendSQLite, driverDSN, err)`. The
  `sqlite://path` → `file:path` rewrite happens here per
  modernc.org/sqlite DSN conventions.
- `internal/store/postgres/` *(new directory)* — move
  `internal/store/{store.go internals, audit.go, flags.go, projects.go,
  environments.go, segments.go, snapshots.go, migrations.go,
  sqlstd.go, convert.go}` into `internal/store/postgres/` as the
  Postgres impl of the new interface. Keep test code at
  `internal/store/` for parametrization in Phase 5.
- `internal/store/errors.go` — relocate `ErrNotFound`, `ErrConflict`,
  `IsConflict` to remain in `internal/store/` (callers import these
  directly today). Make `IsConflict` driver-agnostic — see Phase 4
  for the SQLite half; for now Postgres-only.
- `internal/server/server.go`, `internal/server/handlers/handlers.go`,
  `internal/server/rpc/server.go` — change `*store.Store` to
  `store.Store` (interface). No other changes — the method set is
  unchanged.
- `internal/store/integration_test.go` — replace the
  `s.Pool().Exec("TRUNCATE …")` call (line 38) with a new
  `s.(interface{ TruncateForTest(ctx) error }).TruncateForTest(ctx)`
  type-assertion call. `TruncateForTest` lives on the Postgres impl
  only; Phase 5 will introduce the SQLite equivalent.

**Implementation notes:**

- The `Store` interface must include `WithAudit(ctx, fn func(Querier) error) error`.
  Today `WithAudit` exposes `*db.Queries` (pgx-specific). We replace
  the callback signature with a small `Querier` interface defined in
  `internal/store/` that includes only the methods `WithAudit` callers
  actually use. Grep for `WithAudit(` call sites and produce the
  minimal Querier interface (likely: `AppendAuditEvent`,
  `CreateFlagVersion`, `UpdateFlagVersionSourceText`, plus whatever
  snapshots/segments use). Both `db.Queries` (pg) and
  `dbsqlite.Queries` will satisfy this interface; sqlc emits methods
  with matching signatures when configured with `emit_interface: true`.
- The `PublishFlagVersionTx` method also exposes `*db.Queries` in its
  signature. Same treatment — narrow to `Querier`.
- `Backend` enum:
  ```go
  type Backend string
  const (
      BackendPostgres Backend = "postgres"
      BackendSQLite   Backend = "sqlite"
  )
  ```
- The Postgres impl moves wholesale; this phase contains no logic
  changes. The diff is largely import-path churn and file moves.

**Validation:**

```bash
# Everything builds
go build ./...

# Existing Postgres integration suite still passes
FALSEFLAG_TEST_DATABASE_URL="postgres://falseflag:falseflag@localhost:5432/falseflag?sslmode=disable" \
  go test ./internal/store/...

# Hurl smoke still green
make smoke

# No leaked pgx types outside the postgres impl package
! grep -r "pgxpool\.\|pgtype\.\|pgconn\." --include="*.go" \
    internal/server internal/rpc cmd
```

**Acceptance criteria:**

- [x] `Pool()` getter is gone. `TruncateForTest` replaces the only two
  consumers (`integration_test.go`, `contract_test.go`).
- [x] `Queries()` getter is gone (had zero callers outside its definition).
- [x] DSN dispatch infrastructure (`parseBackend`) in place; sqlite://
  scheme accepted by the parser and returned as `BackendSQLite`; the
  switch in `Open` rejects it with a clear "not yet supported (phase 4)"
  error until Phase 4 wires the SQLite impl.
- [x] Existing Postgres integration tests pass unchanged
  (`go test ./internal/store/...` and the contract test).
- [x] `make smoke` green against the running compose stack.

**Scope note (executed scope vs original plan):** Phase 1 deferred two
items to Phase 4: (a) the file moves into `internal/store/postgres/`
and (b) the `Store` interface + `Querier` narrowing of `WithAudit`'s
callback. Reason: building an interface for a single implementation is
premature — both backends need to exist before the interface shape can
be designed against real constraints (the pgtype-vs-database/sql
divergence in sqlc output means the obvious "wrap `*db.Queries`"
interface won't satisfy both backends). Phase 4 absorbs the file move
and introduces the interface as part of plugging in the SQLite impl.

## Phase 2 — UUID generation in Go

**Goal:** both backends accept Go-generated UUIDs at insert time.
Phase 1 leaves Postgres still using `gen_random_uuid()` defaults;
this phase makes that explicit and drops the defaults so SQLite can
adopt the same schema shape.

**Files:**

- `db/migrations/0001_init.sql`, `0002_flags.sql`, `0003_environments_segments_snapshots.sql` —
  drop `DEFAULT gen_random_uuid()` from every `uuid PRIMARY KEY`
  column. Note: these are existing migration files; editing them
  rewrites history for any deploy that's already run. **For demo
  scope, this is acceptable** — production data does not exist; the
  compose stack and CI rebuild from scratch. If keeping history is
  required, add a new migration `0005_remove_uuid_defaults.sql`
  that `ALTER TABLE … ALTER COLUMN … DROP DEFAULT`s instead.
  *(Default choice: edit in place. Document in the commit message.)*
- `db/queries/*.sql` — `CreateProject`, `CreateFlag`,
  `CreateFlagVersion`, `CreateEnvironment`, `CreateSegment`,
  `CreateSnapshot`, `AppendAuditEvent` — add explicit `id` column to
  `INSERT` (was previously omitted to rely on DEFAULT).
- `internal/store/postgres/*.go` — every `Create*` method generates
  `uuid.New()` and passes it through.
- `go.mod` — `github.com/google/uuid` is already present (used in
  `internal/store/types.go`). No new dep.
- Re-run `make generate` (sqlc) and commit the regenerated
  `internal/db/*.sql.go`.

**Implementation notes:**

- The store's `Create*` methods currently return the row produced
  by `RETURNING *`. After this phase, `RETURNING id` is still
  useful for consistency but the caller already knows the ID. Keep
  `RETURNING *` to preserve the existing `created_at`/`updated_at`
  default semantics.
- Audit events also rely on `gen_random_uuid()`. Same treatment.

**Validation:**

```bash
make generate     # sqlc regenerates
go build ./...
FALSEFLAG_TEST_DATABASE_URL=... go test ./internal/store/...
make smoke
```

**Acceptance criteria:**

- [x] No remaining `gen_random_uuid()` in `db/migrations/**`.
- [x] All `Create*` store methods generate UUIDs via `uuid.New()`
  (`projects`, `flags`, `flag_versions`, `audit_events`, `environments`,
  `segments`, `snapshots` — see `auditInsertParams` in `audit.go` for
  the shared audit helper).
- [x] sqlc regenerated; `internal/db/*.sql.go` now expects `ID pgtype.UUID`
  in every `Create*Params`.
- [x] Integration tests pass against fresh-migrated DB; `make smoke`
  14/14 green after rebuilding the api container image.

## Phase 3 — SQLite schema, queries, dual-engine sqlc

**Goal:** `make generate` produces both Postgres and SQLite Go bindings.
The SQLite migrations apply cleanly to a fresh database; the queries
compile (sqlc parses them); no Go runtime path uses them yet.

**Files:**

- `db/migrations/sqlite/0001_init.sql`, `0002_flags.sql`,
  `0003_environments_segments_snapshots.sql`,
  `0004_flag_versions_source_text.sql` *(new)* — one-to-one
  translations of the Postgres migrations:
  - `uuid` → `TEXT NOT NULL` (UUIDs stored as lowercase string)
  - `timestamptz` → `TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))`
  - `jsonb` → `TEXT NOT NULL` (JSON encoded as string; `DEFAULT '{}'`
    where the Postgres column had `DEFAULT '{}'::jsonb`)
  - `now()` → `(strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))`
  - `::cast` syntax → removed (operand types align natively)
  - `gen_random_uuid()` defaults — already removed in Phase 2; SQLite
    versions of these migrations match.
  - Partial index `WHERE actor IS NOT NULL` from `0003` —
    SQLite supports this; copy verbatim.
- `db/migrations/migrations.go` *(modify)* — extend the embed to
  cover both subdirs:
  ```go
  //go:embed *.sql sqlite/*.sql
  var FS embed.FS
  ```
  Existing Postgres files stay at the top level (no path churn);
  SQLite lives under `sqlite/`. The Postgres path uses
  `fs.Sub(FS, ".")` or equivalent (effectively the existing behavior);
  SQLite uses `fs.Sub(FS, "sqlite")`.
- `db/queries/sqlite/audit.sql`, `environments.sql`, `flags.sql`,
  `projects.sql`, `segments.sql`, `snapshots.sql` *(new)* — SQLite
  translations of the Postgres queries:
  - `DISTINCT ON (fv.flag_id)` in `ListLatestFlagVersions`
    (`snapshots.sql:31`) → rewrite as:
    ```sql
    SELECT … FROM (
      SELECT fv.*, ROW_NUMBER() OVER (
        PARTITION BY fv.flag_id ORDER BY fv.version DESC
      ) AS rn
      FROM flag_versions fv
      INNER JOIN flags f ON f.id = fv.flag_id
      WHERE f.project_id = ?
    ) WHERE rn = 1;
    ```
  - `(created_at, id) < ($7, $8)` in `ListAuditEventsByProject`
    (`audit.sql:15`) → expand to:
    `(created_at < ? OR (created_at = ? AND id < ?))`.
  - `sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action')` →
    drop the `::text` cast; SQLite infers.
  - `now()` in `UpdateSegment` (`segments.sql:22`) →
    `strftime('%Y-%m-%dT%H:%M:%fZ', 'now')` *or* push to Go
    (`SET updated_at = ?`). Push to Go is the cleaner unification
    — apply the same edit to the Postgres `UpdateSegment`.
  - `COALESCE(MAX(version), 0) + 1` — works in SQLite verbatim.
  - `RETURNING *` — works in SQLite ≥ 3.35.0 (modernc bundles 3.53.1).
  - All `$1`, `$2`, … placeholders → `?` (sqlc handles
    parameter mapping when `engine: sqlite`).
- `sqlc.yaml` — add second `sql:` block:
  ```yaml
  version: "2"
  sql:
    - engine: postgresql
      schema: db/migrations
      queries: db/queries
      gen:
        go:
          package: db
          out: internal/db
          sql_package: pgx/v5
          emit_interface: true
    - engine: sqlite
      schema: db/migrations/sqlite
      queries: db/queries/sqlite
      gen:
        go:
          package: dbsqlite
          out: internal/db/sqlite
          sql_package: database/sql
          emit_interface: true
          emit_pointers_for_null_types: true
  ```
  `emit_interface: true` on both is what makes the `Querier` interface
  from Phase 1 satisfiable by both generated `Queries` types.
- `internal/db/sqlite/*.sql.go`, `db.go`, `models.go` — committed
  generated output from `make generate`.

**Implementation notes:**

- The SQLite migrations are authored fresh, not auto-translated.
  Hand-rolling avoids the trap of "almost-correct" type mappings.
- `flag_versions.source_text TEXT NULL` (migration 0004) translates
  trivially — SQLite has the same `TEXT NULL` syntax.
- The query rewrites land in the SQLite-only directory; the Postgres
  queries stay unchanged (except `UpdateSegment` if we choose the
  Go-side `updated_at` unification — preferred).
- sqlc with `emit_pointers_for_null_types: true` makes the SQLite
  output ergonomic: `*string` and `*time.Time` instead of
  `sql.NullString` / `sql.NullTime`. The convert.go shape (Phase 4)
  can then normalize both backends to the same domain types.
- Verify modernc.org/sqlite's bundled SQLite version (`select sqlite_version();`)
  ≥ 3.35.0 at runtime; abort startup with a clear error if not.

**Validation:**

```bash
make generate                           # both backends regen
ls internal/db/sqlite/                  # db.go models.go *.sql.go present
go build ./...                          # whole tree builds
# Note: no SQLite runtime path wired yet — defer to Phase 4
```

**Acceptance criteria:**

- [ ] `sqlc.yaml` declares both engines.
- [ ] `make generate` produces `internal/db/**` and `internal/db/sqlite/**`.
- [ ] SQLite migrations exist at `db/migrations/sqlite/000{1,2,3,4}_*.sql`.
- [ ] SQLite queries exist at `db/queries/sqlite/*.sql`.
- [ ] Both `db.Queries` and `dbsqlite.Queries` satisfy the
  `internal/store.Querier` interface (compile check is sufficient).
- [ ] `go build ./...` clean.

## Phase 4 — SQLite Store implementation

**Goal:** `Open(ctx, "sqlite:///path/to/db.sqlite")` returns a working
`store.Store`; `s.Migrate(ctx, nil)` applies the SQLite migrations;
all `Store` methods are implemented; the same semantics as Postgres
(conflict → `ErrConflict`, not-found → `ErrNotFound`, retry-on-busy).

**Files:**

- `internal/store/sqlite/store.go` *(new)* — `sqliteStore` struct,
  `open(ctx, dsn)` calling `sql.Open("sqlite", ...)`, connection
  configuration:
  - `db.SetMaxOpenConns(1)` — single writer; eliminates `SQLITE_BUSY`
    in practice. Reads block on the same single connection, which
    is fine for demo workload (Hurl smoke is sequential).
  - DSN augmented with PRAGMAs:
    `_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)&_txlock=immediate`.
    The `parseBackend` helper from Phase 1 appends these to the
    operator-provided DSN.
- `internal/store/sqlite/migrations.go` *(new)* — `Migrate`
  implementation using `goose.NewProvider(goose.DialectSQLite3, db, fs.Sub(migrations.FS, "sqlite"))`,
  then `provider.Up(ctx)`.
- `internal/store/sqlite/{projects,flags,environments,segments,snapshots,audit}.go`
  *(new)* — one file per resource. Each method delegates to the
  generated `dbsqlite.Queries` method (one-to-one with the Postgres
  impl) and uses the `convert.go` helpers to map between
  `*string`/`*time.Time` and domain types.
- `internal/store/sqlite/convert.go` *(new)* — `*string ↔ string`
  ("" sentinel where caller expects non-nil), `*time.Time ↔ time.Time`,
  `string ↔ uuid.UUID` (parse/format), `[]byte ↔ json.RawMessage`
  (passthrough since SQLite JSON columns are TEXT and sqlc emits
  `string`; we re-cast).
- `internal/store/sqlite/transactions.go` *(new)* —
  `withImmediateTx(ctx, db, fn func(*sql.Tx) error) error` that does
  `BeginTx`, runs `fn`, commits, and retries on
  `modernc.org/sqlite/lib.SQLITE_BUSY` (5) or `SQLITE_BUSY_SNAPSHOT` (517).
  Mirrors the existing pgx serializable retry loop in
  `internal/store/postgres/audit.go:withAuditOnce`.
- `internal/store/errors.go` *(modify)* — `IsConflict(err)` becomes
  driver-agnostic: try `pgconn.PgError.SQLState == "23505"`; try
  `sqlite.Error.Code == SQLITE_CONSTRAINT_UNIQUE` (2067) or
  `SQLITE_CONSTRAINT_PRIMARYKEY` (1555). Returns `true` on either.
  Same treatment for `ErrNotFound` recognition (`sql.ErrNoRows` and
  `pgx.ErrNoRows` both map).
- `internal/store/store.go` *(modify)* — `Open` dispatch table:
  ```go
  func Open(ctx context.Context, dsn string) (Store, error) {
      backend, drvDSN, err := parseBackend(dsn)
      if err != nil { return nil, err }
      switch backend {
      case BackendPostgres:
          return postgres.Open(ctx, drvDSN)
      case BackendSQLite:
          return sqlite.Open(ctx, drvDSN)
      }
      return nil, fmt.Errorf("unsupported backend: %q", backend)
  }
  ```
- `go.mod` — `modernc.org/sqlite` added; `go mod tidy` updates `go.sum`.

**Implementation notes:**

- **Driver choice:** `modernc.org/sqlite` (pure-Go) is the right
  default. Pre-slice-11 the Dockerfile already builds with
  `CGO_ENABLED=0`; modernc satisfies that with no changes. CGO
  (`mattn/go-sqlite3`) would break the distroless image and the
  arm64 cross-compile target slice 11 lands. Register name is
  `"sqlite"` (not `"sqlite3"`).
- **JSON columns:** SQLite stores JSON as `TEXT`. sqlc with
  `database/sql` emits `string` for non-null `TEXT` columns. The
  convert layer hands a `json.RawMessage([]byte(stringValue))` back
  to callers — domain types are unchanged.
- **Timestamps:** sqlc with the override block from Phase 3 emits
  `time.Time`; modernc.org/sqlite round-trips via the `datetime`
  affinity. UTC is the contract — emit `Z`-suffixed RFC3339 strings
  from the schema defaults.
- **`gen_random_uuid()` removal:** already done in Phase 2.
- **Serializable replacement:** `_txlock=immediate` in the DSN means
  every `db.BeginTx(ctx, nil)` issues `BEGIN IMMEDIATE`, the
  Postgres-serializable analog for SQLite. Combined with single-conn
  writer pool, conflicts are exceptional. The retry loop still
  handles `SQLITE_BUSY_SNAPSHOT` (517) cleanly.
- **Conflict shape:** Postgres returns `23505` for unique violations
  on `(project_id, key)` etc. SQLite returns `SQLITE_CONSTRAINT_UNIQUE`
  (extended 2067, primary 19). Both must map to `store.ErrConflict`.
- **No `Pool()` on the SQLite impl** — `TruncateForTest` runs `DELETE
  FROM` per table (SQLite has no TRUNCATE). Same method signature
  as the Postgres test hook.

**Validation:**

```bash
go mod tidy
make generate
go build ./...

# Round-trip smoke: write to a temp SQLite DB and read back
FALSEFLAG_TEST_DATABASE_URL="sqlite:///tmp/falseflag-sqlite-smoke.db" \
  go test -run TestProjectLifecycle ./internal/store/...
```

**Acceptance criteria:**

- [ ] `store.Open(ctx, "sqlite:///...")` returns a working `Store`.
- [ ] `s.Migrate(ctx, nil)` applies SQLite migrations from
  `db/migrations/sqlite/`.
- [ ] All `Store` methods implemented for SQLite, semantics-equivalent
  to Postgres for ErrConflict / ErrNotFound / retry-on-busy.
- [ ] `internal/store/errors.go.IsConflict` recognizes both drivers.
- [ ] `go vet ./...` clean; no unused imports.

## Phase 5 — Parametrize integration tests

**Goal:** every existing test in `internal/store/integration_test.go`
runs against both backends via `t.Run`. The Postgres path stays
env-gated; the SQLite path runs unconditionally using `t.TempDir()`.

**Files:**

- `internal/store/integration_test.go` — replace
  `newTestStore(t) *Store` with `forEachBackend(t, func(t *testing.T, s Store))`:
  ```go
  func forEachBackend(t *testing.T, fn func(*testing.T, store.Store)) {
      t.Run("sqlite", func(t *testing.T) {
          dir := t.TempDir()
          s := mustOpen(t, "sqlite://"+filepath.Join(dir, "test.db"))
          fn(t, s)
      })
      t.Run("postgres", func(t *testing.T) {
          dsn := os.Getenv("FALSEFLAG_TEST_DATABASE_URL")
          if dsn == "" { t.Skip("FALSEFLAG_TEST_DATABASE_URL not set") }
          s := mustOpen(t, dsn)
          fn(t, s)
      })
  }
  ```
  Every test function (`TestProjectLifecycle`, `TestFlagLifecycle`,
  `TestEnvironmentLifecycle`, `TestSegmentLifecycle`,
  `TestSnapshotMonotonic`, `TestWithAuditRollback`,
  `TestWithAuditRollbackOnPanic`, etc.) becomes a
  `forEachBackend(t, ...)` shell whose body matches today's test.
- `internal/store/testhelpers_test.go` — `mustOpen`, `truncateAll`
  helpers; `truncateAll` does a type-assert to the (test-only)
  `TruncateForTest` hook.
- `Makefile` — add:
  ```
  test-store-sqlite:
  	go test ./internal/store/... -run '^Test' -count=1

  test-store-postgres:
  	FALSEFLAG_TEST_DATABASE_URL=$(FALSEFLAG_TEST_DATABASE_URL) \
  	  go test ./internal/store/... -run '^Test' -count=1
  ```
  These split the matrix for local debugging; CI uses subtest
  filters or runs both.

**Implementation notes:**

- The SQLite path runs **everywhere** — no env gate. It uses
  `t.TempDir()` so there's no shared state.
- The Postgres path stays env-gated so contributors without a local
  Postgres still get green tests (just the SQLite subtests run).
- This phase is the first time the SQLite runtime path is exercised.
  Bugs from Phase 4 surface here.

**Validation:**

```bash
# SQLite-only run (no env var) — must be green out of the box
go test ./internal/store/...

# Both backends
FALSEFLAG_TEST_DATABASE_URL="postgres://..." go test ./internal/store/... -v

# Race detector
go test -race ./internal/store/...
```

**Acceptance criteria:**

- [ ] `go test ./internal/store/...` runs SQLite subtests with no env
  config; all green.
- [ ] With `FALSEFLAG_TEST_DATABASE_URL` set, both Postgres and
  SQLite subtests run and pass.
- [ ] `go test -race ./internal/store/...` green for the SQLite path.

## Phase 6 — compose.sqlite.yaml + smoke-sqlite

**Goal:** `docker compose -f compose.sqlite.yaml up` boots an api
backed by SQLite in a named volume; `make smoke-sqlite` runs the
existing Hurl suite against it.

**Files:**

- `compose.sqlite.yaml` *(new)* — overlay or standalone compose file:
  ```yaml
  services:
    api:
      image: falseflag/api:dev
      environment:
        FALSEFLAG_DATABASE_URL: "sqlite:///data/falseflag.db"
      volumes:
        - falseflag-sqlite:/data
      ports:
        - "8080:8080"
      # no db dependency
  volumes:
    falseflag-sqlite:
  ```
  Standalone (not overlay) because the Postgres compose has a `db`
  service the SQLite stack must not start.
- `scripts/smoke.sh` — accept `FALSEFLAG_BACKEND` env (`postgres`
  default; `sqlite` opts in). The truncate step needs to differ:
  - Postgres: existing `docker compose exec db psql -c "TRUNCATE …"`
    path.
  - SQLite: `docker compose -f compose.sqlite.yaml down -v && up -d`
    (delete the volume, re-bootstrap). Cheap because there's no seed.
    Alternative: add a `DELETE FROM` script run via `docker compose
    exec api ... sqlite3 /data/falseflag.db < /tmp/wipe.sql`. The
    `down -v` form is simpler and what most self-hosted-tool smoke
    scripts use.
- `Makefile` — add:
  ```
  smoke-sqlite:
  	FALSEFLAG_BACKEND=sqlite ./scripts/smoke.sh
  ```
  Keep `smoke` as-is (Postgres).
- `cmd/falseflag-seed/main.go` — verify the seed binary is
  DSN-agnostic. It opens via `store.Open`, so SQLite should work
  unchanged. **Verify**, don't assume.

**Implementation notes:**

- The seed binary's 409-on-rerun idempotency must still hold for
  SQLite — verify against the SQLite-backed stack.
- All 14 hurl files (`tests/hurl/00-health.hurl` through `14-typescript-publish.hurl`)
  must pass against SQLite unmodified. Any hurl file that uses
  backend-specific behavior (e.g., a particular error message
  format) is a bug — fix in this phase.
- For a single-binary local dev experience, also document running
  the api binary directly: `FALSEFLAG_DATABASE_URL=sqlite:///tmp/ff.db ./bin/falseflag-api`.

**Validation:**

```bash
# Stand up the SQLite stack
docker compose -f compose.sqlite.yaml up -d --build
# Wait for readiness
until curl -sf http://localhost:8080/healthz; do sleep 1; done

# Seed + smoke
make seed
make smoke-sqlite

# Tear down
docker compose -f compose.sqlite.yaml down -v
```

**Acceptance criteria:**

- [ ] `compose.sqlite.yaml` boots an api service backed by a
  SQLite file in a named volume.
- [ ] `make smoke-sqlite` runs the full Hurl suite green against
  the SQLite stack.
- [ ] `make smoke` (Postgres) still green — unchanged.
- [ ] `make seed` idempotent against both stacks.

## Phase 7 — CI backend matrix

**Goal:** every CI job that exercises the storage layer fans out
over `backend: [postgres, sqlite]`. CI burn time roughly doubles
on those jobs — the intentional self-hosted-tool slow-CI surface.

**Files:**

- `.github/workflows/ci.yml`:
  - `test-go-race`: add `strategy.matrix.backend: [postgres, sqlite]`,
    `fail-fast: false`. Job runs `go test -race ./...` and conditionally
    exports `FALSEFLAG_TEST_DATABASE_URL` only when `matrix.backend
    == 'postgres'` (with a Postgres service container). For SQLite,
    no env var; SQLite subtests run unconditionally.
  - `contract-test`: same matrix. Postgres branch keeps the existing
    `services: postgres` block + `FALSEFLAG_TEST_DATABASE_URL`.
    SQLite branch skips the service and runs without the env var.
  - `smoke`: matrix on backend. Postgres branch runs `docker compose
    up` and `make smoke`. SQLite branch runs `docker compose -f
    compose.sqlite.yaml up` and `make smoke-sqlite`.
  - `dashboard-e2e`: matrix on backend. Same compose split as smoke.
  - `kind-smoke`: matrix on backend. Helm values need a SQLite
    variant — out of scope for this slice if k8s SQLite mount is
    nontrivial; **make it postgres-only and document the gap**
    (slice 12 owns the deployment artifact matrix and will pick this
    up). Comment in the workflow file.
  - Job names interpolate `${{ matrix.backend }}` so the GitHub UI
    shows `smoke (postgres)`, `smoke (sqlite)`, etc.
- `Makefile` — no changes needed (Phase 6 already added `smoke-sqlite`).

**Implementation notes:**

- GitHub Actions limitation: `services:` blocks are not
  matrix-conditional. The Postgres service container will start in
  both matrix entries. The fix is `if: matrix.backend == 'postgres'`
  on the wait-for-postgres step and the env var export; the SQLite
  entry ignores the running-but-unused container. This is the
  community-accepted idiom.
- `fail-fast: false` so a Postgres regression doesn't mask a SQLite
  one and vice versa.
- The `test-go` (no-race) job doesn't exercise storage today (it
  skips integration tests). Don't matrix it — keep CI lean.
- CI auto-triggers stay off (slice 7b owns re-enablement). All jobs
  remain `workflow_dispatch`-only for now.

**Validation:**

```bash
# Locally simulate via act or by triggering the workflow:
gh workflow run ci-baseline.yml
gh run watch
# Expect: smoke (postgres) ✅, smoke (sqlite) ✅, etc.
```

**Acceptance criteria:**

- [ ] `test-go-race`, `contract-test`, `smoke`, `dashboard-e2e`
  matrix over `[postgres, sqlite]`.
- [ ] `kind-smoke` runs postgres-only with a TODO comment pointing
  at slice 12.
- [ ] Workflow file is syntactically valid (`actionlint`).
- [ ] A manual `workflow_dispatch` run completes green across all
  matrix entries.

## Phase 8 — Documentation

**Goal:** a self-hoster lands in the repo and immediately understands
"single binary + SQLite" vs "compose + Postgres" without reading
the slice plans.

**Files:**

- `README.md` — new section, near the top:

  ```markdown
  ## Storage backends

  falseflag ships with two interchangeable storage backends:

  - **SQLite** — single-binary, zero-dependency, ideal for home labs,
    a single VPS, or a one-container deployment. `DATABASE_URL=sqlite:///var/lib/falseflag/data.sqlite`.
  - **Postgres** — multi-process / multi-replica deployments,
    operational familiarity, mature backup tooling. `DATABASE_URL=postgres://...`.

  The product surface is identical. Pick a backend with the DSN
  scheme in `FALSEFLAG_DATABASE_URL`. See `compose.yaml` (Postgres)
  and `compose.sqlite.yaml` (SQLite) for example stacks.

  Migration parity between backends is **not enforced**; each
  backend has its own goose migration set under `db/migrations/` and
  `db/migrations/sqlite/` that evolves independently.
  ```

- `docs/demo/walkthrough.md` (if it exists from slice 9) — add an
  "Alternate backend" sidebar pointing at `make smoke-sqlite`.
- `docs/METAPLAN.md` — tick slice 10 in the status notes; add a
  paragraph in the "Status Notes" section recording the SQLite
  backend as merged.
- `docs/plans/2026-05-27-001-feat-sqlite-backend-plan.md` — set
  `status: complete` in the front matter as the final commit step.

**Validation:**

- Read the README from the perspective of someone who hasn't seen
  the repo. Does "use SQLite or Postgres" come across in <60s?
- The two compose files are obviously discoverable from the README.

**Acceptance criteria:**

- [ ] README section explains the two backends in 100–200 words.
- [ ] METAPLAN status note ticks slice 10.
- [ ] Plan file flipped to `status: complete`.

## Out of Scope

Explicitly deferred:

- **SQLite-only optimizations** — no FTS5, no virtual tables, no
  partial index tuning beyond what Postgres already has.
- **Backup / restore tooling** — slice 12 owns this. SQLite backups
  are trivial (`.backup` command or file copy with checkpointing)
  but the slice 12 plan will treat backup uniformly for both
  backends and across the deployment matrix.
- **Migration parity checking** — by METAPLAN design, schemas drift.
  No CI job compares the two schemas.
- **Multi-arch container builds** — slice 11. The `modernc.org/sqlite`
  driver is pure Go, so slice 11's arm64 cross-compile will work
  for free once it lands, but the QEMU/buildx wiring is out of
  scope here.
- **Production hardening** — no connection-pool tuning beyond the
  single-writer SQLite default, no PRAGMA tuning for high-write
  workloads, no `mmap_size` tuning.
- **Concurrent-writer benchmark** — the slice does not include a
  performance comparison between backends.
- **Kubernetes SQLite path** — `kind-smoke` stays Postgres-only;
  slice 12 (deployment artifact matrix) will work through the
  StatefulSet vs PVC story for SQLite.
- **Dashboard or SDK changes** — none; storage is invisible above
  the API.

## System-Wide Impact

### Interaction graph

A write to any flag/project/segment now follows this chain regardless
of backend:

1. HTTP/Connect handler in `internal/server/{handlers,rpc}`
2. `store.Store` interface method (e.g. `CreateFlag`)
3. Backend dispatch: `postgres.Store.CreateFlag` or
   `sqlite.Store.CreateFlag`
4. Backend opens its native transaction (`pgx.Serializable` or
   `BEGIN IMMEDIATE`)
5. Generated query: `db.Queries.CreateFlag` or
   `dbsqlite.Queries.CreateFlag`
6. Driver round-trip
7. Convert layer normalizes return shape to domain types
8. Audit event (if `WithAudit`-wrapped) appends in same tx

The audit retry loop fires on `pgx`-flavored SQLSTATE 40001
(serialization_failure) **or** `sqlite.Error` codes 5/517
(BUSY/BUSY_SNAPSHOT). The retry logic itself is identical; only the
"should I retry?" predicate is backend-aware via
`internal/store/errors.go`.

### Error propagation

`store.ErrNotFound` and `store.ErrConflict` are the only sentinels
the API layer cares about. Both must be raised from the SQLite
impl with semantics identical to Postgres:

- `ErrNotFound`: returned when `sql.ErrNoRows` or `pgx.ErrNoRows`
  fires.
- `ErrConflict`: returned on SQLSTATE 23505 (pg) or
  `SQLITE_CONSTRAINT_UNIQUE`/`SQLITE_CONSTRAINT_PRIMARYKEY` (sqlite).

Any error not classified as the above bubbles up as a generic
`error`; handler layer maps to HTTP 500.

### State lifecycle risks

- **SQLite db file ownership** — the compose volume is owned by the
  container's runtime UID. If the api container is rebuilt with a
  different UID (e.g. distroless static → distroless cc), the file
  becomes unreadable. Mitigation: pin the UID in the Dockerfile and
  document.
- **WAL files** — `falseflag.db-wal` and `falseflag.db-shm` are
  created next to the db file. They must be in the same volume.
  The compose volume mount is at the directory level, which
  satisfies this naturally.
- **Half-applied migrations on SQLite** — goose runs each migration
  in its own tx. SQLite supports transactional DDL (unlike MySQL),
  so a failed migration rolls back cleanly. No half-applied state.

### API surface parity

- The public REST + Connect surface is **unchanged**. Slice 10 is
  a storage-only refactor; no protobuf or OpenAPI churn.
- The `internal/store.Store` interface is the new explicit boundary.
  Slice 12 will lean on this for backup orchestration.

### Integration test scenarios

1. `TestFlagLifecycle` against both backends — creates, gets,
   lists, publishes versions, asserts monotonic version numbers.
2. `TestSnapshotLatestFlagVersions` against both — exercises the
   `DISTINCT ON` / `ROW_NUMBER` divergence in the snapshot query.
3. `TestAuditRetryOnSerializationFailure` — manually contend two
   transactions; assert the retry loop fires on pgx 40001 and
   sqlite BUSY_SNAPSHOT (517) with the same observable behavior.
4. `make smoke-sqlite && make smoke` back-to-back — end-to-end
   parity at the Hurl level. Same 14 files green against both.
5. `make seed` twice against a fresh SQLite db — idempotency holds
   (409 swallowed) the same way Postgres does it.

## Sources & References

### Origin

- **METAPLAN slice 10:** `docs/METAPLAN.md` lines 531–571 — the
  prompt that this plan implements verbatim.
- **METAPLAN slow-CI framing:** `docs/METAPLAN.md` lines 533–535
  — why slices 10–12 stack multiplicatively.

### Internal references

- **Current Postgres store:** `internal/store/store.go:23` (struct),
  `:34` (`pgxpool.New`), `:58` (`Pool()` getter — to be removed).
- **Pg-coupled call sites:** `internal/store/audit.go:138`,
  `flags.go:109`, `snapshots.go:28` (serializable BeginTx);
  `errors.go:29` (SQLSTATE check).
- **Generated sqlc output (Postgres):** `internal/db/models.go`,
  `internal/db/*.sql.go` — pgtype types throughout.
- **Current sqlc config:** `sqlc.yaml` — single-engine.
- **Current migrations:** `db/migrations/000{1,2,3,4}_*.sql` —
  use `uuid`, `jsonb`, `timestamptz`, `gen_random_uuid()`.
- **DISTINCT ON usage:** `db/queries/snapshots.sql:31`,
  `internal/db/snapshots.sql.go:94`.
- **Row-value cursor:** `db/queries/audit.sql:15–16`.
- **Hurl smoke entry:** `scripts/smoke.sh`.
- **Compose:** `compose.yaml` (Postgres path; no SQLite yet).
- **CI workflow:** `.github/workflows/ci.yml` — single-engine today.
- **Plan style reference:** `docs/plans/2026-05-26-002-feat-slice-9-polish-demo-script-plan.md`.

### External references

- **modernc.org/sqlite** ([pkg.go.dev](https://pkg.go.dev/modernc.org/sqlite)) —
  pure-Go SQLite driver. Bundled SQLite 3.53.1 (May 2026); registers
  as `"sqlite"` with `database/sql`; DSN supports `_pragma=…` and
  `_txlock=immediate` URI params.
- **sqlc multi-engine config**
  ([docs.sqlc.dev configuration reference](https://docs.sqlc.dev/en/stable/reference/config.html))
  — second `sql:` block with `engine: sqlite`, `sql_package: database/sql`,
  `emit_pointers_for_null_types: true`. `emit_interface: true` makes
  both generated `Queries` types satisfy a shared interface.
- **goose v3 multi-dialect**
  ([pressly/goose pkg.go.dev](https://pkg.go.dev/github.com/pressly/goose/v3))
  — `goose.NewProvider(dialect, db, fs.Sub(embeddedFS, "sqlite"))` is
  the recommended stateful API; dialect string `"sqlite3"` (or
  constant `goose.DialectSQLite3`).
- **SQLite RETURNING** — supported from 3.35.0 (March 2021);
  modernc bundles 3.53.1.
- **SQLite DISTINCT ON workaround** — `ROW_NUMBER() OVER (PARTITION BY …)`
  in a subquery; supported since SQLite 3.25 (window functions).
- **SQLite BEGIN IMMEDIATE + busy_timeout** — the closest analog to
  `pgx.Serializable` for a single-writer workload. Combined with
  `MaxOpenConns(1)` the writer is effectively serialized.
- **SQLITE_BUSY_SNAPSHOT (extended code 517)** — WAL-mode-specific
  "your read snapshot is stale; retry the whole tx" error. Must be
  recognized by the retry loop.
- **GitHub Actions matrix + services** — `services:` blocks are not
  matrix-conditional; community idiom is to start the service
  unconditionally and gate connection steps with
  `if: matrix.backend == 'postgres'`.

### Related slices

- **Slice 7a (slow-CI baseline):** the matrix fan-out here doubles
  the surface that slice 7a established.
- **Slice 7b (Depot acceleration):** owns CI auto-trigger re-enable;
  this slice keeps triggers off.
- **Slice 11 (multi-arch images):** modernc.org/sqlite is pure Go,
  so the arm64 cross-compile pre-work in this slice is zero.
- **Slice 12 (deployment artifact matrix):** owns backup/restore
  for both backends and the StatefulSet/PVC story for the
  kind-smoke SQLite path.

## Commit Plan

Eight logical commits to `main`, in this order:

1. `refactor(store): hide pgx behind Store interface and DSN dispatch`
   *(Phase 1)*
2. `refactor(store): generate UUIDs in Go; drop gen_random_uuid() defaults`
   *(Phase 2)*
3. `feat(db): sqlc dual-engine config and SQLite schema/queries`
   *(Phase 3)*
4. `feat(store): SQLite backend via modernc.org/sqlite`
   *(Phase 4)*
5. `test(store): parametrize integration suite over postgres and sqlite`
   *(Phase 5)*
6. `feat(compose): compose.sqlite.yaml + make smoke-sqlite`
   *(Phase 6)*
7. `ci: backend matrix fan-out over {postgres, sqlite}`
   *(Phase 7)*
8. `docs: SQLite backend section + slice 10 tick`
   *(Phase 8)*

No PRs; CI auto-triggers remain off (slice 7b territory).
