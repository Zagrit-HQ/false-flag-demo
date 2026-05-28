# CI baseline (slice 7a)

This is the "before Depot" picture of FalseFlag's CI. One workflow
file — `.github/workflows/ci.yml` — runs everything every PR touches
on stock `ubuntu-latest` runners, with stock `actions/cache`, stock
`docker/buildx` + QEMU for arm64, and stock `services:` containers.
It is slow because it does real work across the whole monorepo,
not because it has been deliberately pessimized.

Slice 7b will ship a sibling workflow that swaps in Depot runners,
Depot Cache, and Depot container builds. The contrast at the
conference comes from reading both runs' per-job times out of the
GitHub Actions API — no custom timing instrumentation in either
workflow.

## What runs

| job | what it does | makefile target |
|---|---|---|
| `generate-check` | regenerates buf, sqlc, controller-gen, OpenAPI, orval and fails if the tree is dirty | `make generate-check` |
| `lint-go` | runs `golangci-lint` against `./...` | `make lint-go` |
| `lint-openapi` | runs Spectral against `api/openapi/openapi.yaml` with `spectral:oas` recommended ruleset | — |
| `lint-js` | runs `biome check` over the JS workspace | `make lint-js` |
| `typecheck-js` | `pnpm typecheck` (turbo orchestrates `^build` of dependencies first) | — |
| `test-go` | `gotestsum` over the full Go module, no DB | `make test-go` |
| `test-go-race` | `go test -race` over `eval`, `sdkgo`, `store` — packages with shared mutable state | — |
| `test-js` | `pnpm test` after `pnpm generate` | `make test-js` |
| `build-js` | `pnpm build` after `pnpm generate` | `make build-js` |
| `conformance` | shared 25-fixture corpus run against Go SDK and TS SDK | `make conformance` |
| `contract-test` | REST↔Connect parity test against a Postgres 16 service container | `make contract-test` |
| `build-images` | `docker buildx bake default` for both `linux/amd64` and `linux/arm64` (cache-only output) | `make bake` (with overrides) |
| `image-scan` | amd64 bake into the docker daemon, then Trivy HIGH/CRITICAL scan over every service image | — |
| `smoke` | full compose stack up, seed, Hurl corpus (12 files, 73 requests), MCP smoke | `make smoke` + `make mcp-smoke` |
| `dashboard-e2e` | full compose stack up, seed, Playwright happy-path against the running dashboard on `:3000` | `make dashboard-e2e` |
| `kind-smoke` | kind cluster, kind-loaded operator image, kustomize apply, sample manifest reconciliation | `make kind-smoke` |

## Why it's slow

1. **Multi-arch buildx via QEMU.** Every PR builds five images for both `linux/amd64` and `linux/arm64`. arm64 runs under emulation on the amd64 runner; this dominates the wall time of `build-images` cold.
2. **Cold buildx cache per branch.** `actions/cache` is keyed on the branch name with `main` as the universal restore-key. The first push on a new branch pays a cold cache. The buildx local cache also can't share work between jobs running in parallel — `smoke` and `dashboard-e2e` each rebuild their own compose stack.
3. **Two compose-up jobs.** `smoke` and `dashboard-e2e` each do a `docker compose up -d --build` from cold. Sharing the build by pushing pre-built images between jobs is exactly the kind of thing slice 7b will demonstrate.
4. **No Depot Cache.** No remote, deduplicated, cross-job build cache. `setup-go`, `pnpm`, `actions/cache` all serve their narrow caches; nothing ties them together.
5. **No matrices.** Every job is named so future branch protection can pin them; no auto-generated expansions.
6. **No path filters.** Every PR runs every job. A real team would add path filters eventually — that's a manual optimization slice 7b is not responsible for.

## Reading per-job timings

GitHub Actions records `started_at` / `completed_at` / `conclusion`
on every job; no in-workflow instrumentation is needed.

```sh
# Get the full job list with timestamps for a given run.
gh run view <run-id> --json jobs --jq '.jobs[] | {name, status, conclusion, started_at, completed_at}'

# Or for a finer-grained payload (per-step durations included):
gh api repos/Zagrit-HQ/false-flag/actions/runs/<run-id>/jobs --jq '.jobs[] | {name, started_at, completed_at}'
```

Slice 7b reads from the same API, so a side-by-side comparison is
just two `gh run view` calls.

## What slice 7b will replace

- `runs-on: ubuntu-latest` → Depot runners.
- `actions/cache` for buildx layers → Depot Cache.
- `docker buildx bake` over QEMU → Depot container builds on native arm64.
- Two compose-up jobs that re-build everything → shared image hand-off via Depot's ephemeral registry.

## Known gotchas

- A `go.sum` bump can trigger `generate-check` failure if any buf or
  sqlc plugin output drifts. Re-run generate locally and commit.
- The Hurl `.deb` URL is pinned to a specific Orange-OpenSource
  release tag. If it 404s in the future, bump `HURL_VERSION` in
  `.github/workflows/ci.yml`.
- `kind-smoke` rewrites `deploy/kustomize/overlays/dev/patch.yaml`
  in-place during CI to use the docker0 bridge gateway IP instead of
  `host.docker.internal`. The edit is not committed — the dev
  overlay stays as-is for local development.
