# FalseFlag — PlatformCon Feature Flag Demo

FalseFlag is a synthetic feature flag platform used in the PlatformCon
conference demo about accelerating slow CI/CD pipelines with Depot.

The implementation target is **demo-quality**: broad, believable surfaces
that compile, run, and look real enough for a conference talk.
Production hardening is intentionally out of scope unless a slice
explicitly calls for it.

The repository directory keeps the conference codename
(`project-platformcon`); the product is FalseFlag.

## One-command demo

From a clean clone:

```bash
brew install go pnpm hurl docker      # or your platform equivalents
pnpm --dir js install

docker compose up -d --build          # boots db + api + proxy + operator + mcp + dashboard
make seed                             # 3 projects, 7 flags, 3 snapshots
make smoke                            # 14 hurl files, 89 requests against the live stack
open http://localhost:3030            # click any flag — real source, edit, publish snapshot
```

A guided walkthrough (`docs/demo/smoke-walkthrough.md`) and the
conference-talk script (`docs/demo/script.md`) build on this.

## Storage backends

FalseFlag ships with two interchangeable storage backends:

- **SQLite** — single binary, zero external dependency. Ideal for home
  labs, a single VPS, or a one-container deployment.
  `FALSEFLAG_DATABASE_URL=sqlite:///var/lib/falseflag/data.sqlite`
- **Postgres** — multi-process / multi-replica deployments, operational
  familiarity, mature backup tooling.
  `FALSEFLAG_DATABASE_URL=postgres://...`

The product surface is identical. Pick a backend by setting the DSN
scheme in `FALSEFLAG_DATABASE_URL`. Two example compose stacks are
included:

```bash
docker compose up -d --build                          # Postgres (compose.yaml)
docker compose -f compose.sqlite.yaml up -d --build   # SQLite (compose.sqlite.yaml)

make smoke         # Hurl suite against the Postgres stack
make smoke-sqlite  # same suite against the SQLite stack
```

Each backend has its own goose migration set —
`db/migrations/` for Postgres, `db/migrations/sqlite/` for SQLite. The
two evolve independently; cross-engine migration parity is
intentionally not enforced.

## Quickstart (dev loop)

```bash
# One-time setup
brew install go pnpm hurl docker
pnpm --dir js install

# Build and test
go build ./cmd/...
go test ./...
pnpm --dir js -r build
pnpm --dir js -r test

# Run the control-plane API and the dashboard directly (no docker)
make api-dev          # http://localhost:8080/healthz
make dashboard-dev    # http://localhost:3030

# Smoke check the API via Hurl (needs a running API)
make smoke
```

The dashboard dev server runs on **port 3030** to coexist with other
projects that default to 3000. Compose maps the dashboard container's
internal port 3000 to host 3030 as well.

## Layout

- `cmd/falseflag-*` — Go binary entry points (`api`, `proxy`,
  `operator`, `mcp`, `seed`).
- `internal/**` — Go implementation packages, including SQLC and Buf
  output under `internal/db/**` and `internal/gen/**`. No `pkg/**`.
- `operator/**` — Kubernetes operator (CRD types under
  `operator/api/v1alpha1`, reconcilers under `operator/controllers`).
- `proto/falseflag/v1` — protobuf contracts managed by Buf.
- `api/openapi` — OpenAPI 3 contract + oapi-codegen config.
- `db/migrations`, `db/queries` — goose migrations + SQLC sources.
- `js/**` — pnpm + Turborepo TypeScript workspace
  (`apps/{dashboard,cli}`, `packages/{sdk-js,config-ts,
  generated-client-ts,shared-eval-corpus}`).
- `tests/hurl/**` — HTTP API smoke/e2e tests.
- `tests/eval-corpus/**` — cross-runtime evaluation fixtures (Go + TS).
- `deploy/{helm,kustomize,manifests,crds}` — Kubernetes packaging.
- `infra/{Dockerfile,Dockerfile.dashboard,docker-bake.hcl}` — container
  builds. `compose.yaml` lives at the repo root.
- `docs/{plans,ideation,architecture,demo,METAPLAN.md}` — planning,
  architecture, and demo docs.

## Architecture

See [docs/architecture/diagram.mmd](./docs/architecture/diagram.mmd)
for the Mermaid source. Render with `mmdc -i diagram.mmd -o
diagram.svg` if you have the Mermaid CLI installed.

## FAQ

**Why are `.github/workflows/ci.yml` auto-triggers commented out?**
Slice 7b owns re-enabling them with Depot acceleration. Until then the
workflow file is `workflow_dispatch`-only and the local validation
ladder (`go test ./...`, `pnpm -r test`, `make smoke`,
`make conformance`, `make contract-test`) is the source of truth.

**Where do I read the demo script?**
[`docs/demo/script.md`](./docs/demo/script.md) for the 8–10 minute
conference pacing; [`docs/demo/smoke-walkthrough.md`](./docs/demo/smoke-walkthrough.md)
for the step-by-step interactive walkthrough.

## Conventions

See [CONTRIBUTING.md](./CONTRIBUTING.md) for layout rules,
root-file ownership, code-generation reproducibility, and the
validation ladder. See [AGENTS.md](./AGENTS.md) and
[docs/METAPLAN.md](./docs/METAPLAN.md) before adding new top-level
areas.
