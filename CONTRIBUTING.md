# Contributing to FalseFlag

FalseFlag is the synthetic feature flag platform used in the PlatformCon
demo about accelerating slow CI/CD with Depot. The codebase is
**demo-quality, not production-quality** — we prefer broad believable
surface area over deep hardening. This guide captures the layout rules
and conventions so parallel workers don't relitigate them.

## Repository Layout

```text
cmd/                 Go binaries. One directory per cmd/falseflag-*.
internal/            All Go implementation packages.
  appconfig/         Runtime env-driven config loaders.
  buildinfo/         Version, Commit, WithGracefulShutdown.
  logging/           log/slog factory tagged with service.name.
  server/            cmd/falseflag-api HTTP server.
  proxy/             cmd/falseflag-proxy HTTP server.
  db/                SQLC-generated Go query layer.
  gen/proto/         Buf-generated Go proto + ConnectRPC stubs.
  gen/openapi/       oapi-codegen output.
  (others)           Slice 2+: audit, config (strategies), eval, flags, …
operator/            Kubernetes operator code.
  api/v1alpha1/      CRD Go types + controller-gen DeepCopy output.
  controllers/       controller-runtime Reconcilers.
proto/               Buf source: proto/falseflag/v1/*.proto.
api/openapi/         OpenAPI 3 source + oapi-codegen cfg.
db/                  goose migrations and SQLC query files.
deploy/              Kubernetes manifests, Helm, Kustomize, CRDs.
infra/               Dockerfiles + docker-bake.hcl.
js/                  pnpm + Turborepo workspace for ALL TypeScript.
  apps/dashboard/    Remix v2 + Tailwind + Radix.
  apps/cli/          Commander CLI.
  packages/sdk-js/   TypeScript SDK.
  packages/config-ts/  Config-as-code DSL.
  packages/generated-client-ts/  Orval-generated REST client.
  packages/shared-eval-corpus/   Cross-runtime evaluation fixtures.
scripts/             Shell helpers (smoke.sh today).
tests/               Hurl + golden + e2e fixtures.
docs/                Plans, ideation, METAPLAN, architecture notes.
```

## Layout Rules

- **No Go `pkg/**` tree.** Implementation lives in `internal/**`. If
  something needs to be shared with an external module, lift it into a
  separate Go module later — don't pre-emptively create a public package.
- **Generated Go artifacts** live only in:
  - `internal/gen/proto/**` (Buf)
  - `internal/gen/openapi/**` (oapi-codegen)
  - `internal/db/**` (SQLC)
  - `operator/api/v1alpha1/zz_generated.deepcopy.go` (controller-gen)
  - `deploy/crds/**` (controller-gen CRD)
- **Generated TypeScript artifacts** live only in
  `js/packages/generated-client-ts/src/generated/**` (Orval).
- **All TypeScript** lives under `js/**`. No `.ts`/`.tsx` outside that
  tree.
- **Logging** in every Go binary goes through
  `internal/logging.New(suffix)`. `log/slog` only — no zap or zerolog.
- **Service entry points** in `cmd/*/main.go` wrap their run function in
  `buildinfo.WithGracefulShutdown(suffix, run)`. Keep `main.go` under
  50 lines.
- **Runtime env vars** are prefixed `FALSEFLAG_` and loaded in
  `internal/appconfig`. `internal/config` is reserved for slice-2
  config-strategy compilers, not runtime loading.

## Root-File Ownership

The following root files are touched by the main thread only. If your
parallel slice needs to change them, coordinate via the slice plan or
ask for an integration commit:

- `go.mod`, `go.sum`
- `js/package.json`, `js/pnpm-workspace.yaml`, `js/pnpm-lock.yaml`,
  `js/turbo.json`, `js/biome.json`, `js/tsconfig.base.json`,
  `js/tsconfig.json`
- `Makefile`
- `buf.yaml`, `buf.gen.yaml`
- `sqlc.yaml`
- `infra/docker-bake.hcl`, `infra/Dockerfile`, `infra/Dockerfile.dashboard`

See `docs/METAPLAN.md` for the full dependency/parallelisation map.

## Generators

Every generator is invoked via `go tool` so contributors don't need
Homebrew installs.

```bash
make generate          # everything
make generate-go       # buf + sqlc + controller-gen + oapi-codegen
make generate-js       # turbo orval (and friends)
```

`make generate` must be reproducible:

```bash
make generate && git diff --exit-code
```

If that diff is non-empty, someone hand-edited a generated file —
revert and re-generate.

## Validation Ladder

Slice 1 anchors the **Compile** and **HTTP Contract** rungs from
`docs/METAPLAN.md`. Before declaring a slice done:

```bash
go build ./cmd/...
go test ./...
make generate && git diff --exit-code
make lint                                # golangci-lint + biome
pnpm --dir js -r typecheck
pnpm --dir js -r test
pnpm --dir js -r build
make smoke                               # boots api + runs Hurl
cd infra && docker buildx bake --print   # parses bake-hcl
```

Update `docs/METAPLAN.md` Status Notes with what ran and any skipped
checks.

## Common Tasks

- `make api-dev` — run the API locally on :8080.
- `make dashboard-dev` — run the Remix dashboard on :3000.
- `make smoke` — boot api + run Hurl smoke.
- `pnpm --dir js --filter @falseflag/<package> <script>` — work on a
  single workspace package.

## Style

- Go: `go fmt`, `goimports`, golangci-lint defaults.
- TypeScript: Biome (`pnpm lint`). Double quotes, 2-space indent,
  organize-imports on save.
- Commit messages: conventional (`feat(scope): …`, `fix(scope): …`,
  `docs(plan): …`, `build(make): …`).
- One logical unit per commit. Phase 1 of slice 1 landed in seven
  commits — keep that cadence.
