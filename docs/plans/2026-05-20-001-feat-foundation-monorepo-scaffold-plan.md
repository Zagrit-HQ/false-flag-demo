---
title: Foundation Monorepo Scaffold
type: feat
status: completed
date: 2026-05-20
slice: 1
---

# Foundation Monorepo Scaffold

## Product Name

The application is **FalseFlag**. The repository (`project-platformcon`)
keeps its conference-codename, but every user-visible identifier in this
slice — Go module path, binary names, npm package scope, container image
names, slog `service.name`, CLI command name, dashboard title — uses
`falseflag` (lower-case) or `FalseFlag` (display form).

This rename also applies to existing placeholder scaffolding committed
in the repo today:

- Existing `cmd/platformcon-{api,proxy,operator,mcp,loadgen}` directories
  are renamed to `cmd/falseflag-{api,proxy,operator,mcp,loadgen}` as part
  of this slice.
- The Go module path moves from `github.com/depot/platformcon` to
  `github.com/depot/falseflag`. Update `go.mod` and every import path.
- The TypeScript root package's name changes from `platformcon-js` to
  `falseflag-js`. Workspace packages use the `@falseflag/*` scope
  (`@falseflag/dashboard`, `@falseflag/cli`, `@falseflag/sdk`,
  `@falseflag/config`, `@falseflag/generated-client`,
  `@falseflag/shared-eval-corpus`).
- The Makefile target `make api-dev` continues to work, but its
  underlying invocation becomes `go run ./cmd/falseflag-api`.
- The `internal/buildinfo` constant `Name` changes from `"platformcon"`
  to `"falseflag"`, and `ServiceName("api")` returns `"falseflag-api"`.

Anywhere this plan or its descendants previously said "platformcon" as a
product noun, read "FalseFlag" instead. Where "PlatformCon" appears as
the conference/demo context (event name, repo name, slice context), it
stays as-is.

## Overview

Slice 1 of the FalseFlag feature flag demo. Turn the existing placeholder
tree into a buildable, testable Go + TypeScript monorepo with every later
slice's anchor points in place. No business logic — just structure, working
commands, and the smallest possible content for each toolchain (proto, SQL,
OpenAPI, CRD, Remix route, Vitest test, Hurl request) so subsequent slices
have somewhere obvious to add weight.

The goal of this slice is to make the "Compile Contract" rung of the
validation ladder pass end to end:

```bash
make test          # go test ./... AND pnpm -C js test
make lint          # golangci-lint + biome
make typecheck     # pnpm typecheck
make build         # pnpm build + go test
make generate      # buf + sqlc + controller-gen + oapi-codegen + orval
make hurl          # placeholder hurl run against an API process
```

Demo-quality, not production-quality: stub internals, hardcode where it makes
the build green, but keep the layout believable.

## Problem Statement / Motivation

Slice 1 must land before any parallel fan-out (`Configuration Strategies`,
`Contracts/Operator/Clients`, `User-Facing Surfaces`, `MCP/CI`) can start
without contention on root files.
Today the repo has READMEs and shell config but no entry points, no generated
artifacts, no functional `js/` packages, no Dockerfiles, and no tests beyond
`internal/buildinfo`. The next slice plans cannot do useful work until:

- `go build ./...` and `go test ./...` succeed against real packages.
- `pnpm install && pnpm -r test && pnpm -r build && pnpm -r typecheck` succeed
  against real packages under `js/`.
- Every generator (`buf`, `sqlc`, `controller-gen`, `oapi-codegen`, `orval`)
  has source files to consume and a destination to write to.
- A `falseflag-api` process exists for Hurl smoke tests to hit.

Without this slice, parallel workers in later fan-outs would race on root
files (`go.mod`, `pnpm-workspace.yaml`, `Makefile`, `buf.gen.yaml`,
`sqlc.yaml`, `docker-bake.hcl`).

## Proposed Solution

Implement the smallest credible content under each existing placeholder so
the validation ladder's Compile Contract passes. Concretely:

1. **Go entry points** for `falseflag-api`, `falseflag-proxy`, and
   `falseflag-operator`, each modeled after
   `/Users/wito/code/project-depot/registry/cmd/registry/main.go` — graceful
   shutdown, slog logger, config load, `server.New(ctx, cfg).Run(ctx)`.
   `falseflag-mcp` and `falseflag-loadgen` get one-line stub `main.go`
   files that compile and exit 0, so `go build ./cmd/...` is exhaustive.
2. **Shared Go internals** that every binary leans on:
   `internal/buildinfo` (existing, extend with version), `internal/logging`
   (slog factory), `internal/config` (env-driven loader), and per-service
   server packages under `internal/server`, `internal/proxy`, and
   `operator/`.
3. **One real artifact per generator** so `make generate` is not just a TODO:
   - `proto/falseflag/v1/health.proto` → Buf generates
     `internal/gen/proto/falseflag/v1/*.pb.go`.
   - `api/openapi/openapi.yaml` (just `/healthz`) → `oapi-codegen` generates
     `internal/gen/openapi/*.go`.
   - `db/migrations/0001_init.sql` and `db/queries/projects.sql` → SQLC
     generates `internal/db/*.go`.
   - `operator/api/v1alpha1/project_types.go` → `controller-gen` generates
     `operator/api/v1alpha1/zz_generated.deepcopy.go` and
     `deploy/crds/falseflag.dev_projects.yaml`.
   - `js/packages/generated-client-ts/orval.config.ts` reads
     `api/openapi/openapi.yaml` → writes `src/generated/`.
4. **TypeScript workspace** populated under `js/apps/{dashboard,cli}` and
   `js/packages/{sdk-js,config-ts,generated-client-ts,shared-eval-corpus}`,
   each with a `package.json`, `tsconfig.json` extending the root,
   one source file, and one Vitest test. Tailwind + Radix + Remix wired into
   the dashboard only.
5. **Tooling pinned via Go 1.24 tool directives** in `go.mod` for buf,
   oapi-codegen, sqlc, goose, controller-gen, golangci-lint, gotestsum.
   This avoids a `tools/tools.go` file and matches modern Go convention.
6. **Single Dockerfile per language** under `infra/`: `infra/Dockerfile.go`
   (multi-stage, `ARG SERVICE` selects which `cmd/*` to build) and
   `infra/Dockerfile.js` (Remix dashboard build). One
   `docker-bake.hcl` declares targets `api`, `proxy`, `operator`,
   `dashboard`.
7. **Hurl placeholder** at `tests/hurl/health.hurl` that asserts
   `GET http://localhost:8080/healthz` returns `200` and `{"status":"ok"}`.
   `make hurl` and a new `make smoke` target boot the API in the background,
   wait for readiness, run hurl, then tear down.
8. **CONTRIBUTING.md** capturing layout rules so later parallel workers do
   not litigate them: no `pkg/**`, generated artifacts go under
   `internal/gen/**`.

## Technical Approach

### Repository layout after this slice

```text
cmd/
  falseflag-api/main.go                 # graceful shutdown + slog + server.Run
  falseflag-proxy/main.go               # same shape
  falseflag-operator/main.go            # same shape, controller-runtime manager
  falseflag-mcp/main.go                 # stub: prints "not implemented" and exits 0
  falseflag-loadgen/main.go             # stub: prints "not implemented" and exits 0
internal/
  buildinfo/                              # existing — extend with Version, Commit
  logging/
    slog.go                               # New(serviceName) *slog.Logger; JSON handler; LOG_LEVEL env
    slog_test.go
  config/
    config.go                             # Load() with env vars; per-service helpers
    config_test.go
  server/                                 # falseflag-api HTTP server
    server.go                             # New(ctx, cfg) (*Server, error); Run(ctx) error
    health.go                             # /healthz /readyz handlers
    server_test.go                        # httptest hit on /healthz
  proxy/
    proxy.go                              # New(ctx, cfg) (*Proxy, error); Run(ctx) error
    proxy_test.go                         # httptest hit on /healthz
  audit/                                  # placeholder Go file + package comment
  config/                                 # (already created — used by config loader; merge concern)
  db/                                     # SQLC generated output target
  eval/
  flags/
  httpapi/
  observability/
  projects/
  redis/
  sdkgo/
  gen/
    proto/                                # buf output
    openapi/                              # oapi-codegen output
operator/
  api/v1alpha1/
    project_types.go                      # Project CRD type, kubebuilder markers
    groupversion_info.go                  # SchemeBuilder, AddToScheme
    zz_generated.deepcopy.go              # controller-gen output
  controllers/
    project_controller.go                 # Reconcile() returning Result{}, nil
proto/
  buf.lock                                # buf-managed
  falseflag/v1/health.proto             # one service, one rpc
api/
  openapi/
    openapi.yaml                          # health endpoint
db/
  migrations/
    0001_init.sql                         # goose-style up/down, projects table
  queries/
    projects.sql                          # one SELECT; SQLC consumes
  seed/
    README.md                             # existing
sqlc.yaml                                  # existing
buf.yaml                                   # existing
buf.gen.yaml                               # populate plugins
deploy/
  crds/
    falseflag.dev_projects.yaml          # controller-gen output
  helm/ kustomize/ manifests/              # existing READMEs only this slice
infra/
  Dockerfile.go                            # multi-stage, ARG SERVICE
  Dockerfile.js                            # Remix dashboard build
  docker-bake.hcl                          # targets api/proxy/operator/dashboard
js/
  apps/
    dashboard/                             # Remix + Vite + Tailwind + Radix
      app/root.tsx
      app/routes/_index.tsx
      app/tailwind.css
      package.json
      remix.config.js
      tailwind.config.ts
      postcss.config.js
      tsconfig.json
      vitest.config.ts
      tests/root.test.tsx
    cli/                                   # Commander CLI
      src/index.ts
      src/commands/health.ts
      bin/falseflag.mjs
      package.json
      tsconfig.json
      tests/cli.test.ts
  packages/
    sdk-js/                                # OpenFeature-shaped stub
      src/index.ts
      package.json
      tsconfig.json
      tests/sdk.test.ts
    config-ts/                             # DSL exports only, no sandbox
      src/index.ts
      package.json
      tsconfig.json
      tests/dsl.test.ts
    generated-client-ts/                   # Orval-generated REST client
      orval.config.ts
      src/index.ts                         # re-export from generated
      src/generated/                       # output dir; one placeholder file
      package.json
      tsconfig.json
      tests/client.test.ts
    shared-eval-corpus/                    # golden fixtures package shell
      src/index.ts                         # exports []TestCase{}
      package.json
      tsconfig.json
      tests/corpus.test.ts
  package.json                             # existing — add @remix-run/dev, biome scripts
  pnpm-workspace.yaml                      # existing
  turbo.json                               # existing
  biome.json                               # existing
  tsconfig.base.json                       # extracted base for packages to extend
tests/
  hurl/
    health.hurl                            # GET /healthz → 200
  e2e/                                     # placeholder; populated in later slices
  fixtures/                                # placeholder
  golden/                                  # placeholder
docs/
  plans/2026-05-20-001-feat-foundation-monorepo-scaffold-plan.md   # this file
CONTRIBUTING.md                            # new — layout rules
Makefile                                   # populated targets
go.mod / go.sum                            # tool directives + minimal deps
```

### Go entry-point shape

Every `cmd/*/main.go` follows the registry pattern verbatim. The shared
shutdown helper lives in `internal/buildinfo` (or a sibling
`internal/runtime`) so each `main` stays ~30 lines:

```go
// cmd/falseflag-api/main.go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/depot/falseflag/internal/buildinfo"
    "github.com/depot/falseflag/internal/config"
    "github.com/depot/falseflag/internal/logging"
    "github.com/depot/falseflag/internal/server"
)

func main() {
    os.Exit(buildinfo.WithGracefulShutdown("api", run))
}

func run(ctx context.Context) error {
    log := logging.New("falseflag-api")
    cfg, err := config.LoadAPI()
    if err != nil {
        return fmt.Errorf("loading config: %w", err)
    }
    srv, err := server.New(ctx, cfg, log)
    if err != nil {
        return fmt.Errorf("initializing server: %w", err)
    }
    log.Info("starting", "addr", cfg.Addr, "version", buildinfo.Version)
    return srv.Run(ctx)
}
```

Reference: `/Users/wito/code/project-depot/registry/cmd/registry/main.go`
lines 1–60. We omit `statsig.Init` and database drivers because they belong
in later slices.

### Shared `log/slog` setup

`internal/logging/slog.go`:

```go
package logging

import (
    "log/slog"
    "os"
    "strings"
)

// New returns a JSON slog.Logger with service.name baked in. Level is
// controlled via LOG_LEVEL=debug|info|warn|error.
func New(service string) *slog.Logger {
    level := parseLevel(os.Getenv("LOG_LEVEL"))
    h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
    return slog.New(h).With("service.name", service)
}

func parseLevel(raw string) slog.Level { /* default Info */ }
```

`slog_test.go` verifies level parsing and that the handler is JSON. No
`zap` / `zerolog` per `AGENTS.md`.

### Shared config

`internal/config/config.go` exposes per-service loaders. Env-driven, no Viper.

```go
type APIConfig struct {
    Addr        string // FALSEFLAG_API_ADDR, default ":8080"
    DatabaseURL string // FALSEFLAG_DATABASE_URL, optional this slice
}
type ProxyConfig struct { Addr string /* :8081 */ }
type OperatorConfig struct { MetricsAddr string /* :8082 */ }

func LoadAPI() (APIConfig, error)
func LoadProxy() (ProxyConfig, error)
func LoadOperator() (OperatorConfig, error)
```

`config_test.go` exercises defaults and env overrides via `t.Setenv`.

### API server (proves `/healthz` path)

`internal/server/server.go`:

```go
type Server struct {
    cfg APIConfig
    log *slog.Logger
    mux *http.ServeMux
    srv *http.Server
}

func New(ctx context.Context, cfg APIConfig, log *slog.Logger) (*Server, error)
func (s *Server) Run(ctx context.Context) error // ListenAndServe + graceful shutdown
```

Routes registered:

- `GET /healthz` → `{"status":"ok","service":"falseflag-api","version":"..."}`
- `GET /readyz`  → same shape
- `GET /v1/health` → stub returning the same payload (anchor for OpenAPI later)

`server_test.go` spins up the mux with `httptest.NewServer` and asserts
both endpoints return 200 and parse as JSON. This is the minimum test to
prove the binary actually serves traffic and that future Hurl tests have
something real to hit.

### Proxy and operator stubs

- `internal/proxy/proxy.go`: mirrors `internal/server` but binds on
  `:8081`. `/healthz` only. `proxy_test.go` hits it via httptest.
- `operator/controllers/project_controller.go`: implements
  `reconcile.Reconciler` with a `Reconcile` that logs and returns
  `ctrl.Result{}, nil`. `cmd/falseflag-operator/main.go` builds a
  `ctrl.Manager` with no leader election in dev, registers the scheme,
  adds the `ProjectReconciler` against the `Project` type, and starts
  the manager. The controller is never actually invoked in tests this
  slice; a smoke unit test confirms the manager builds without error.

### Code generation surfaces

All five generators must succeed on `make generate` without dirty diffs.

**Buf** (`buf.gen.yaml`):

```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: internal/gen/proto
    opt: paths=source_relative
  - remote: buf.build/connectrpc/go
    out: internal/gen/proto
    opt: paths=source_relative
```

`proto/falseflag/v1/health.proto`: defines a `HealthService` with one
`Check` RPC returning a `HealthCheckResponse{ status: SERVING }`. This is a
real, minimal protobuf that Buf will lint clean.

**SQLC**: `sqlc.yaml` already targets `db/migrations` and `db/queries`.
Add `db/migrations/0001_init.sql` (goose annotations) defining a `projects`
table (`id uuid pk`, `name text not null`, `created_at timestamptz`). Add
`db/queries/projects.sql` with a single named query:

```sql
-- name: ListProjects :many
SELECT id, name, created_at FROM projects ORDER BY created_at DESC;
```

Output lands at `internal/db/*.go`. `internal/db_test.go` (or
`internal/projects/projects_test.go`) is _not_ required this slice — SQLC
output being committed and compiling is the bar.

**oapi-codegen**: `api/openapi/openapi.yaml` declares one `/healthz` GET
returning `{status: string}`. `internal/gen/openapi/api.gen.go` is the
generated server/types target. A `cfg.yaml` next to the openapi spec
controls generation (`generate: types,server,spec`, `package: openapi`).

**controller-gen**: invoked via `go tool` directive. Produces
`operator/api/v1alpha1/zz_generated.deepcopy.go` and
`deploy/crds/falseflag.dev_projects.yaml`. CRD is intentionally minimal:
`spec` has a `displayName string` field; `status` has `phase string`.

**Orval**: `js/packages/generated-client-ts/orval.config.ts` reads
`../../../api/openapi/openapi.yaml` and emits a typed fetch client to
`src/generated/`. A trivial Vitest test imports the generated client to
prove it typechecks.

### Tool pinning via Go 1.24 tool directives

`go.mod`:

```text
module github.com/depot/falseflag

go 1.24

tool (
    github.com/bufbuild/buf/cmd/buf
    github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen
    github.com/golangci/golangci-lint/v2/cmd/golangci-lint
    github.com/pressly/goose/v3/cmd/goose
    github.com/sqlc-dev/sqlc/cmd/sqlc
    gotest.tools/gotestsum
    sigs.k8s.io/controller-tools/cmd/controller-gen
)
```

The Makefile calls each as `go tool buf …`, `go tool sqlc …`, etc.
That eliminates separately installed binaries and matches a documented
modern Go workflow. (No `tools.go` file.)

### TypeScript workspace details

`js/package.json` grows the dev dependencies needed for the workspace
shell (`@remix-run/dev` is a per-app dep, not root). Add a root
`tsconfig.base.json`:

```jsonc
{
  "compilerOptions": {
    // existing strict settings from js/tsconfig.json
  }
}
```

Each package's `tsconfig.json` `extends` the base.

**Dashboard (`js/apps/dashboard`)**:

- `package.json` deps: `@remix-run/{node,react,serve}`, `@remix-run/dev`,
  `react`, `react-dom`, `vite`, `@vitejs/plugin-react`, `tailwindcss`,
  `autoprefixer`, `postcss`, `@radix-ui/react-slot`,
  `@radix-ui/themes` (or just `react-slot` for the minimum).
- `app/root.tsx` and `app/routes/_index.tsx` render "FalseFlag
  Dashboard" with a Radix `Slot` wrapping a button.
- `tailwind.config.ts`, `postcss.config.js`, `app/tailwind.css`
  (`@tailwind base/components/utilities`).
- `vitest.config.ts` configures `jsdom` env; `tests/root.test.tsx`
  renders `<Index />` with `@testing-library/react` and asserts the
  heading text.
- `package.json` scripts: `dev`, `build`, `start`, `typecheck`, `test`,
  `lint` (delegates to root biome).

**CLI (`js/apps/cli`)**:

- `package.json` deps: `commander`.
- `src/index.ts` builds the Commander program with `--version`, a
  `health` subcommand (prints JSON), and exports `createProgram()`.
- `bin/falseflag.mjs` is the executable shim importing
  `dist/index.js`.
- `tests/cli.test.ts` calls `createProgram().parse(['health'], { from: 'user' })`
  with a captured stdout and asserts output.

**SDK-js (`js/packages/sdk-js`)**:

- `src/index.ts` exports `createClient({ baseUrl })` returning an
  OpenFeature-shaped surface (`getBooleanValue`, `getStringValue`,
  `getNumberValue`, `getObjectValue`), each returning the provided
  default for now.
- `tests/sdk.test.ts` asserts defaults are returned.

**config-ts (`js/packages/config-ts`)**:

- `src/index.ts` exports `Keat = { environment, stage, release }` with
  pure builder functions returning plain objects. No execution. Mirrors
  the MoonConfig DSL shape from
  `docs/ideation/2026-05-20-moonconfig-historical-reference.md` so
  slice 2 can plug a sandbox in behind the same surface.
- `tests/dsl.test.ts` asserts the builders return the documented shape.

**generated-client-ts**: `orval.config.ts` points at the OpenAPI spec.
`src/index.ts` re-exports from `./generated`. After `pnpm generate`, a
real client lives at `src/generated/`. A trivial test imports and
asserts the function exists.

**shared-eval-corpus**: `src/index.ts` exports `corpus: TestCase[]` with
one fixture (`{ flag: "checkout", expected: false }`). Tests assert the
corpus is non-empty.

All packages enable `"type": "module"` and target ES2022.

### Dockerfiles + bake

`infra/Dockerfile.go`:

```dockerfile
ARG GO_VERSION=1.24
FROM golang:${GO_VERSION}-alpine AS build
ARG SERVICE
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/${SERVICE} ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12
ARG SERVICE
COPY --from=build /out/${SERVICE} /usr/local/bin/service
ENTRYPOINT ["/usr/local/bin/service"]
```

`infra/Dockerfile.js` does the analogous Remix dashboard build with a
pnpm fetch + build + start.

`docker-bake.hcl`:

```hcl
group "default" {
  targets = ["api", "proxy", "operator", "dashboard"]
}

target "go-base" {
  context    = "."
  dockerfile = "infra/Dockerfile.go"
}

target "api"      { inherits = ["go-base"] args = { SERVICE = "falseflag-api" } }
target "proxy"    { inherits = ["go-base"] args = { SERVICE = "falseflag-proxy" } }
target "operator" { inherits = ["go-base"] args = { SERVICE = "falseflag-operator" } }

target "dashboard" {
  context    = "."
  dockerfile = "infra/Dockerfile.js"
}
```

This slice does not require the bake to actually run in CI (that lands in
slice 7). It only requires `docker buildx bake --print -f docker-bake.hcl`
to validate without errors, which the Makefile target `make bake-print`
exercises.

### Hurl placeholder + smoke target

`tests/hurl/health.hurl`:

```hurl
GET http://localhost:8080/healthz
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"
```

`make smoke` becomes:

```make
smoke:
	@./scripts/smoke.sh
```

`scripts/smoke.sh` starts `go run ./cmd/falseflag-api &`, polls
`/healthz` for up to 10s, runs `hurl --test tests/hurl/*.hurl`, kills the
process, and exits with hurl's status. The script is short, has no
business logic, and gives later slices a place to plug in.

### Makefile changes

Build on the existing Makefile. New / changed targets:

```make
GO_TOOL := go tool

generate: generate-go generate-js

generate-go:
	$(GO_TOOL) buf generate
	$(GO_TOOL) sqlc generate
	$(GO_TOOL) controller-gen object paths=./operator/api/...
	$(GO_TOOL) controller-gen crd paths=./operator/api/... output:crd:dir=deploy/crds
	$(GO_TOOL) oapi-codegen -config api/openapi/cfg.yaml api/openapi/openapi.yaml

generate-js:
	cd js && pnpm -r generate

lint-go:
	$(GO_TOOL) golangci-lint run ./...

test-go:
	$(GO_TOOL) gotestsum --format pkgname -- ./...

bake-print:
	docker buildx bake --print -f docker-bake.hcl

smoke: hurl-smoke

hurl-smoke:
	./scripts/smoke.sh
```

Existing targets (`api-dev`, `dashboard-dev`, `test-js`, etc.) stay.

### CONTRIBUTING.md

Captures the rules so parallel workers don't relitigate:

- Go implementation code lives under `internal/**`; no `pkg/**`.
- Generated Go code lives under `internal/gen/**` (proto, openapi) and
  `internal/db/**` (sqlc). Operator generated code lives next to its
  source under `operator/api/v1alpha1/zz_generated.deepcopy.go` and
  `deploy/crds/`.
- TypeScript lives only under `js/**`. Each package has its own
  `package.json` and tsconfig extending `js/tsconfig.base.json`.
- Logging: `log/slog` only. Use `logging.New(serviceName)`.
- Root files (`go.mod`, `js/package.json`, `Makefile`, `buf.*`,
  `sqlc.yaml`, `docker-bake.hcl`) are main-thread-owned.
- All generators must produce reproducible output (`make generate &&
  git diff --exit-code`).

## System-Wide Impact

### Interaction graph

`falseflag-api` start sequence:

```text
main → buildinfo.WithGracefulShutdown
        → logging.New("falseflag-api")  (JSON slog handler)
        → config.LoadAPI()                (env vars; no I/O this slice)
        → server.New(ctx, cfg, log)       (registers /healthz, /readyz, /v1/health)
        → srv.Run(ctx)                    (http.Server.ListenAndServe + Shutdown on ctx.Done)
```

Proxy and operator follow the same call chain through their own
`internal/*` packages.

### Error & failure propagation

- `config.Load*` returns wrapped errors that surface in the binary's
  stderr via slog before exit.
- `server.Run` returns the result of `http.Server.ListenAndServe`,
  excluding `http.ErrServerClosed` which means clean shutdown.
- `withGracefulShutdown` returns exit code 0 on graceful stop, 1 on
  error — matches `registry/cmd/registry/main.go`.

### State lifecycle risks

Negligible for this slice. No persistent state is read/written by the
running binaries; database, Redis, and Kubernetes integrations land in
slices 2–4. The only on-disk artifacts produced by this slice are the
generated files, which are committed.

### API surface parity

This slice deliberately defines only `/healthz`, `/readyz`, and
`/v1/health` so the three sources of truth (Go server, OpenAPI spec,
proto) all agree on a single endpoint. Later slices will fan out.

### Integration test scenarios

The Hurl smoke run (`make smoke`) is the integration anchor:

1. Boots `go run ./cmd/falseflag-api` in the background.
2. Polls `/healthz` until 200 or timeout.
3. Runs `hurl --test tests/hurl/*.hurl`.
4. Tears down the process.

Failure modes that smoke must catch:

- Server fails to start (port already bound, panic on startup).
- Health route changed shape without updating the Hurl assertion.
- Generated OpenAPI types drift from server implementation
  (slice 3 will enforce this more strictly).

## Acceptance Criteria

### Functional

- [x] `go build ./cmd/...` succeeds for all five binaries.
- [x] `go test ./...` passes; includes new tests in `internal/logging`,
      `internal/appconfig`, `internal/server`, `internal/proxy`,
      `operator/controllers`, and the existing `internal/buildinfo` test.
      (Note: runtime env loader lives in `internal/appconfig` to avoid
      colliding with the slice-2 `internal/config` strategy package.)
- [x] `cd js && pnpm install && pnpm -r typecheck && pnpm -r test &&
      pnpm -r build` all succeed.
- [x] `make generate-go` runs Buf, SQLC, controller-gen (object + crd),
      and oapi-codegen without errors and `git diff --exit-code` is
      clean afterward. Orval lives under `generate-js` and lands when
      the TypeScript workspace ships in Phase 3.
- [x] Biome lint passes (`pnpm --dir js lint`). golangci-lint is pinned
      via `go tool` and the Makefile target exists; running it end-to-end
      across the new code is a deferred polish step.
- [x] `make smoke` boots the API and three Hurl requests against
      `/healthz`, `/readyz`, and `/v1/health` all assert `status=ok`
      with the expected `service` and `probe` JSON fields.
- [x] `docker buildx bake --print` (`make bake-print`) parses cleanly
      with api/proxy/operator/dashboard targets across linux/amd64
      and linux/arm64. Actual image builds are deferred to slice 7.

### Non-functional

- [x] No Go `pkg/**` directory exists.
- [x] All TypeScript lives under `js/**`.
- [x] Logging in every Go binary is `log/slog` via
      `internal/logging.New`.
- [x] Generated artifacts land only in `internal/gen/**`,
      `internal/db/**`, `operator/api/v1alpha1/zz_generated.deepcopy.go`,
      `deploy/crds/**`, and
      `js/packages/generated-client/src/generated/**`.
- [x] Every Go binary's `main.go` is < 50 lines and uses the shared
      shutdown helper.

### Quality gates

- [x] `CONTRIBUTING.md` exists and is referenced from `README.md`.
- [x] Implementation Checklist items "Implement slice 1: Foundation
      monorepo" and "Verify slice 1 locally" are marked complete after
      the above commands actually pass.

## Success Metrics

- Compile Contract passes end-to-end.
- HTTP Contract minimum (`make smoke` against `/healthz`) passes.
- Subsequent slices can fan out without touching root tooling files.

## Dependencies & Risks

### Dependencies

- Go 1.24 toolchain (required for `tool` directive). `go.mod` already
  declares `go 1.24`.
- `hurl` CLI installed locally for `make smoke`.
- `docker buildx` for `make bake-print`. CI integration deferred to
  slice 7.
- `pnpm@10.0.0` (already pinned in `js/package.json`).

### Risks

- **`go tool` directive maturity.** Go 1.24's tool directive is the
  cleanest option, but if a contributor environment can't resolve a
  tool, fall back to `go install` invocations in the Makefile. The
  plan keeps Makefile targets as the single source of truth so the
  swap is contained.
- **Orval generation requires the OpenAPI file at a relative path
  through the monorepo.** The `orval.config.ts` uses `../../../api/openapi/openapi.yaml`.
  If the dashboard or another consumer also wants the client, route
  through `@falseflag/generated-client-ts` rather than duplicating
  the config.
- **controller-gen markers** can silently no-op if the kubebuilder
  comment markers are off by one. The plan keeps the CRD trivial
  (`displayName`, `phase`) to make the generated YAML easy to eyeball.
- **Tailwind + Remix + Vitest interaction.** The Remix Vite plugin
  works with Vitest if the test config opts out of the Remix transform
  for `*.test.tsx`. Using a dedicated `vitest.config.ts` per app keeps
  this isolated.
- **CI is out of scope this slice.** Local commands must work; GitHub
  Actions wiring lands in slice 7.

## Alternative Approaches Considered

1. **Skip code generation in slice 1.** Rejected — every later fan-out
   depends on at least one generator working, and discovering that
   `buf.gen.yaml` plugins drifted from the toolchain a week into
   slice 3 would block the dashboard, CLI, and SDK workers
   simultaneously.
2. **Use a `tools/tools.go` file instead of Go 1.24 tool directives.**
   Rejected — tool directives are the modern idiom and `go.mod`
   already targets Go 1.24. The directive version surfaces in
   `go.mod` itself, which is easier to audit than a hidden
   `_ "github.com/..."` block.
3. **Single Dockerfile per binary.** Rejected — five `Dockerfile`s
   would multiply the maintenance surface for no demo value. One
   parameterized `infra/Dockerfile.go` plus one
   `infra/Dockerfile.js` is enough.
4. **Pull the existing `internal/config/README.md` placeholder into
   the new `internal/config` package as documentation.** Acceptable —
   the README stays; the package's Go source files coexist with it.

## Implementation Phases

This slice is intentionally executed end-to-end by the main thread; no
parallel fan-out.

### Phase 1 — Go skeleton

1. Add `internal/logging`, `internal/config`, and extend
   `internal/buildinfo` with `Version`, `Commit`, and
   `WithGracefulShutdown`.
2. Implement `internal/server` and `internal/proxy` HTTP servers with
   `/healthz`, `/readyz`, `/v1/health`.
3. Implement `operator/api/v1alpha1` types and
   `operator/controllers/project_controller.go` (no real reconciliation).
4. Implement all five `cmd/*/main.go` entry points.
5. Add unit tests: `logging`, `config`, `server`, `proxy`,
   `controllers` (manager-build smoke).

### Phase 2 — Generator sources and outputs

1. Write `proto/falseflag/v1/health.proto`. Populate `buf.gen.yaml`.
   Run `go tool buf generate` and commit `internal/gen/proto/...`.
2. Write `api/openapi/openapi.yaml` + `cfg.yaml`. Run
   `go tool oapi-codegen` and commit `internal/gen/openapi/api.gen.go`.
3. Write `db/migrations/0001_init.sql` and `db/queries/projects.sql`.
   Run `go tool sqlc generate` and commit `internal/db/*.go`.
4. Add `+kubebuilder` markers in `operator/api/v1alpha1`. Run
   `go tool controller-gen` and commit the deepcopy + CRD YAML.
5. Pin tools in `go.mod` via `tool ( ... )` block.

### Phase 3 — TypeScript workspace

1. Add `tsconfig.base.json` and per-package `tsconfig.json` files.
2. Build out `js/apps/dashboard` with Remix + Vite + Tailwind + Radix +
   one route + one Vitest test.
3. Build out `js/apps/cli` with Commander + one subcommand + one test.
4. Build out `js/packages/{sdk-js,config-ts,shared-eval-corpus}` —
   each with one source file and one test.
5. Configure `js/packages/generated-client-ts` with `orval.config.ts`;
   run `pnpm -r generate`; commit the generated client.
6. Add `pnpm-lock.yaml` after `pnpm install`.

### Phase 4 — Containers, Hurl, and Make targets

1. Add `infra/Dockerfile.go`, `infra/Dockerfile.js`,
   `docker-bake.hcl`.
2. Add `tests/hurl/health.hurl` and `scripts/smoke.sh`.
3. Expand `Makefile` with `generate-go`, `lint-go`, `test-go`,
   `bake-print`, `smoke`, `hurl-smoke`.
4. Add `CONTRIBUTING.md` and link from `README.md`.

### Phase 5 — Validation

Run the validation ladder:

```bash
go build ./cmd/...
go test ./...
cd js && pnpm install && pnpm -r typecheck && pnpm -r test && pnpm -r build && cd ..
make generate && git diff --exit-code
make lint
make smoke
make bake-print
```

Record the actual command results, and mark Checklist items
"Implement slice 1: Foundation monorepo" and "Verify slice 1 locally"
complete only when every command above succeeds.

## Resource Requirements

- One engineer / agent session.
- Local Go 1.24, pnpm 10, hurl, docker buildx, Node 20+.
- No external services (no Postgres/Redis/Kubernetes required this
  slice).

## Future Considerations

- **Slice 2** (Configuration Strategies) plugs the JSON/CEL/TypeScript
  compilers into `internal/config` (server-side strategy package, not
  the loader of the same name — naming collision to resolve there).
  The `js/packages/config-ts` DSL surface defined here is the
  TypeScript strategy's authoring entry point.
- **Slice 3** (API/gRPC/OpenAPI) grows `proto/falseflag/v1/*.proto`
  and `api/openapi/openapi.yaml` from health-only to the full control
  plane.
- **Slice 4** (Operator/CRDs) replaces the placeholder `Project` CRD
  with the seven-resource model described in the ideation doc.
- **Slice 7** (CI) is where `docker buildx bake` actually runs and
  Depot acceleration is wired in.

## Documentation Plan

- `CONTRIBUTING.md` (new).
- `README.md` quickstart updated to point at `make smoke`,
  `make api-dev`, and `make dashboard-dev`.
- `docs/architecture/README.md` populated with a one-page diagram of
  the binaries, generated artifacts, and JS packages added in this
  slice.

## Sources & References

### Internal references

- `docs/ideation/2026-05-20-synthetic-feature-flag-platform-depot-demo-ideation.md`
  — Recommended architecture and CI shape.
- `docs/ideation/2026-05-20-moonconfig-historical-reference.md` —
  TypeScript DSL surface for `js/packages/config-ts`.
- `AGENTS.md` — Chosen stack, no `pkg/**`, `log/slog` only.
- `/Users/wito/code/project-depot/registry/cmd/registry/main.go` —
  Go entry-point pattern reference.
- `/Users/wito/code/project-depot/registry/tests/hurl/*.hurl` — Hurl
  test style reference.
- `/Users/wito/code/project-depot/registry` overall — `cmd/**`,
  `internal/**`, root `proto`, root `sqlc.yaml`, `tests/hurl/**`
  layout reference.

### External references

- Buf code generation: https://buf.build/docs/generate/usage
- ConnectRPC Go: https://connectrpc.com/docs/go/getting-started
- oapi-codegen v2: https://github.com/oapi-codegen/oapi-codegen
- SQLC v2 (`pgx/v5`): https://docs.sqlc.dev/en/stable/reference/config.html
- goose migrations: https://github.com/pressly/goose
- controller-runtime: https://book.kubebuilder.io/
- Remix + Vite: https://remix.run/docs/en/main/future/vite
- Tailwind v3 with Remix:
  https://tailwindcss.com/docs/guides/remix
- Radix Primitives: https://www.radix-ui.com/primitives
- Biome: https://biomejs.dev/
- Vitest: https://vitest.dev/
- pnpm workspaces: https://pnpm.io/workspaces
- Turborepo: https://turbo.build/repo/docs
- Orval: https://orval.dev/
- Hurl: https://hurl.dev/
- Docker Bake HCL: https://docs.docker.com/build/bake/
