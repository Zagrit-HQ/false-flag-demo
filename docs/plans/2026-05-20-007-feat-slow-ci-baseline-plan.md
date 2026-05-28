---
title: Slow CI Baseline (Slice 7a)
type: feat
status: active
date: 2026-05-20
---

# Slow CI Baseline (Slice 7a)

## Overview

Codify the FalseFlag monorepo's "before Depot" CI story as a single GitHub Actions workflow that runs slowly because it does real work across many surfaces — not because anything has been deliberately pessimized. Slice 7a delivers the baseline workflow, supporting cache and tool wiring, a machine-readable timing artifact, and a short explainer doc. Slice 7b (deferred to the maintainer) layers Depot runners, Depot Cache, and Depot container builds onto a sibling workflow that reads the same timing shape.

This is the slice that finally wires up two items deferred since slice 1: `golangci-lint` end-to-end and the Playwright dashboard E2E (`docs/METAPLAN.md:599`, `docs/METAPLAN.md:703`).

## Problem Statement

The repo has six slices of real product surface (Go services, TS apps, generated code, K8s operator, MCP server, Hurl smoke, Playwright E2E) but **no CI**. There's no `.github/workflows/` directory and no automated verification of any kind beyond what a developer runs locally with `make`. For the conference demo to land, we need a credible "before" picture: a workflow that a competent team would actually ship, configured the way they'd actually configure it, running on the runners they'd actually use, slow for the reasons it would actually be slow.

Two failure modes to avoid:

1. **Pessimization theatre.** Stripping `actions/cache`, forcing cold builds, inserting `sleep` calls, running everything in a single sequential job. The audience must not be able to point at the workflow and say "that's not real, no one writes CI like that."
2. **Premature optimization.** Skipping multi-arch builds on PR, gating expensive jobs behind path filters, sharing pre-built images between jobs via registry pushes. These are reasonable optimizations a team *might* apply — but they're the exact wins slice 7b will demonstrate with Depot. Doing them here erases the contrast.

The target sits between: stock GitHub-hosted runners, stock `actions/cache`, stock `docker/buildx` with QEMU for arm64, stock `services:` containers, and one workflow file that runs everything on every PR.

## Proposed Solution

One workflow file (`.github/workflows/ci.yml`) with 14 independent jobs plus a timing aggregator. Each job emits a per-job timing JSON artifact; the aggregator merges them into `timing-summary.json` and writes a Markdown table to `$GITHUB_STEP_SUMMARY`. Slice 7b emits the same shape so the two runs can be diffed at the conference.

The workflow reuses existing `make` targets (`make generate-check`, `make lint-go`, `make test-go`, `make conformance`, `make smoke`, `make mcp-smoke`, `make dashboard-e2e`, `make kind-smoke`, `make bake`) rather than reimplementing the underlying invocations. This keeps the workflow file readable, gives developers a one-to-one local reproduction story, and means slice 7b only has to swap runners/cache backends — not rewrite the work.

## Technical Approach

### Architecture

**One workflow, fourteen jobs, one aggregator.** All jobs run on `ubuntu-latest` (currently `ubuntu-24.04`). Jobs are fully independent — no `needs:` chains except for the aggregator. GitHub Actions' default concurrency limit caps wall-time naturally; we add `concurrency: { group: ..., cancel-in-progress: true }` so a re-push doesn't double-queue.

**Tool installation pattern.** Go via `actions/setup-go@v5` (built-in module cache). Node + pnpm via `pnpm/action-setup@v4` → `actions/setup-node@v4` with `cache: pnpm`. Buildx via `docker/setup-buildx-action@v3` + `docker/setup-qemu-action@v3`. Kind via `helm/kind-action@v1`. Hurl via pinned `.deb` download from the upstream GitHub release.

**Generator tools (buf, sqlc, controller-gen, oapi-codegen, golangci-lint, gotestsum) require no install steps** — they're pinned in the Go `tool` block of `go.mod` (lines 419–427) and invoked via `go tool <name>` exactly as the Makefile does. `setup-go` caches the module download so first-run install is the only slow path.

**Generated code first.** Three JS jobs (`typecheck-js`, `test-js`, `build-js`) need `pnpm -r generate` (orval against `api/openapi/openapi.yaml`) before they can run, because `@falseflag/dashboard` imports from `@falseflag/generated-client`. Each job runs its own `pnpm install && pnpm -r generate` — no artifact sharing between jobs.

### Workflow Shape

```
.github/workflows/ci.yml
├── triggers: [pull_request → main, push → main, workflow_dispatch]
├── concurrency: group per ref, cancel-in-progress
├── permissions: { contents: read, actions: read }  # actions:read for timing aggregator
│
├── job generate-check       # buf+sqlc+controller-gen+oapi+orval, then git diff --exit-code
├── job lint-go              # `make lint-go`
├── job lint-js              # `make lint-js`
├── job typecheck-js         # pnpm install → pnpm -r generate → pnpm -r typecheck
├── job test-go              # `make test-go`  (gotestsum, no DB)
├── job test-go-race         # go test -race ./internal/evaluation/... ./internal/store/...
├── job test-js              # pnpm install → pnpm -r generate → pnpm -r test
├── job build-js             # pnpm install → pnpm -r generate → pnpm -r build
├── job contract-test        # services: postgres:16-alpine → `make contract-test`
├── job conformance          # `make conformance`
├── job build-images         # docker/setup-qemu+buildx → `make bake` (linux/amd64,linux/arm64)
├── job smoke                # docker compose up --build → make seed → make smoke → make mcp-smoke
├── job dashboard-e2e        # docker compose up --build → make seed → cached chromium → make dashboard-e2e
├── job kind-smoke           # `make kind-smoke`  (helm/kind-action provides the cluster)
│
└── job timing-summary       # needs: [all], if: always() → download all timing-* artifacts → merge → $GITHUB_STEP_SUMMARY
```

### Decisions and Trade-offs

SpecFlow surfaced eight ambiguities. Resolved as follows; each decision favors the "realistic but unaccelerated" middle path.

| # | Decision | Choice | Rationale |
|---|----------|--------|-----------|
| 1 | `kind-smoke` on PR or main-only? | **PR + main** | The operator IS one of the products. Real teams that ship operators gate kind on PR. Also: slice 7b can't accelerate what isn't in the baseline. |
| 2 | Multi-arch buildx on PR? | **Always multi-arch** | The repo ships multi-arch images per `infra/docker-bake.hcl:24`. PR-amd64-only is a real optimization; skip it so slice 7b can show the QEMU savings. |
| 3 | `contract-test` Postgres source | **`services:` block** | Native GHA, simpler YAML, no extra Go dependency. `pg_isready` poll before `make contract-test`. |
| 4 | Push images to GHCR? | **Build, don't push** | Slice 7a doesn't produce releases. Bake with `--set *.output=type=docker` (or `type=image,push=false`). Avoids `packages: write` permission. |
| 5 | `dashboard-e2e` target | **Compose container on :3000** | Closer to production, matches the slice 5 deferral note. Set `FALSEFLAG_DASHBOARD_URL=http://127.0.0.1:3000` so Playwright's `webServer` block (`js/apps/dashboard/playwright.config.ts:16-27`) is bypassed. |
| 6 | Timing artifact format | **JSON per job + merged summary** | `timing-<job>.json` with `{job, start_ts, end_ts, duration_s, outcome, runner_os}`. Aggregator emits `timing-summary.json` (machine) and renders a table to `$GITHUB_STEP_SUMMARY` (human). Slice 7b mirrors the schema. |
| 7 | Run `smoke` from runner host | **Yes** | `docker compose exec` on ubuntu-latest is standard. Matches `scripts/smoke.sh:24` as-is. |
| 8 | Triggers | **`pull_request` + `push:main` + `workflow_dispatch`** | Standard. `workflow_dispatch` lets the maintainer fire baseline runs on demand for demo prep. |

Additional decisions:

- **One workflow file, not split.** Easier to compare slice 7a's `ci.yml` against slice 7b's `ci-depot.yml` side-by-side at the conference.
- **No path filters.** Every PR runs everything. A real team often adds path filters; skipping that is a deliberate "we haven't optimized yet" call that fits the demo narrative without being unrealistic.
- **No matrices.** Job names stay stable (matters if branch protection is later added). Race tests target specific packages directly (`./internal/evaluation/... ./internal/store/...` per the ideation doc) instead of a matrix over packages.
- **`fail-fast: true` is default**; no matrices to worry about.
- **`actions/cache` is used everywhere a real team would use it**: Go module cache (via `setup-go`), pnpm store (via `setup-node`), buildx local cache (`/tmp/.buildx-cache`), Playwright browsers (`~/.cache/ms-playwright`). Buildx cache is keyed per branch with a fallback to `main`.
- **Composite actions for timing.** `.github/actions/timing-start/action.yml` and `.github/actions/timing-end/action.yml` keep each job's YAML to two extra steps. The end action runs with `if: always()` so failed jobs still emit timing.

### Implementation Phases

Phase-per-commit, matching the workflow that worked for slices 1–6.

#### Phase 1: Bake/Dockerfile readiness

**Goal:** verify `infra/docker-bake.hcl` builds headless under CI conditions and that the buildx local-cache mounts in the Dockerfiles cooperate with `actions/cache`.

- Inspect `infra/Dockerfile` `--mount=type=cache` directives (lines 16–17, 22–23) and confirm they are addressed by cache IDs.
- Add `cache-to`/`cache-from` plumbing in the `bake` invocation when called from CI. Either:
  - Extend `docker-bake.hcl` with optional `cache-to`/`cache-from` blocks gated on a `CI=true` env var.
  - Or pass `--set *.cache-to=...` / `--set *.cache-from=...` from the workflow only.
- Add a `make ci-bake` target if needed (thin wrapper that injects `--load=false` and the cache flags).
- Verify locally: `make bake-print` shows expected groups.

**Files touched:** `infra/docker-bake.hcl` (optional tweak), maybe `Makefile`.

**Done when:** `docker buildx bake --set *.platform=linux/amd64,linux/arm64 default` runs end-to-end on a clean checkout with cache-to/cache-from set to a local dir, and prints all five image tags.

#### Phase 2: Workflow skeleton + static jobs

**Goal:** the simplest possible `ci.yml` that runs on PR and lights up the first three checkmarks.

- Create `.github/workflows/ci.yml` with triggers, concurrency, permissions, env block.
- Create composite actions `.github/actions/timing-start/action.yml` and `.github/actions/timing-end/action.yml`.
- Add jobs: `generate-check`, `lint-go`, `lint-js`. All three need Go and/or Node setup.
- Wire timing emission on each.

**Files touched:** `.github/workflows/ci.yml`, `.github/actions/timing-{start,end}/action.yml`.

**Done when:** a PR-target branch pushed to GHA shows three green checks and three `timing-*` artifacts.

#### Phase 3: Go + JS test, typecheck, build, conformance

**Goal:** all of `test-go`, `test-go-race`, `typecheck-js`, `test-js`, `build-js`, `conformance` jobs added.

- Each JS job: `pnpm/action-setup@v4` → `actions/setup-node@v4` with `cache: pnpm` → `pnpm install --frozen-lockfile` → `pnpm -r generate` (for typecheck/test/build) → target command.
- `test-go-race` scopes to packages from `docs/ideation/2026-05-20-synthetic-feature-flag-platform-depot-demo-ideation.md` ("shared evaluation and repository packages") — likely `./internal/evaluation/... ./internal/store/...`. Validate the path list against the actual layout when implementing.
- `conformance` needs both Go and Node; install both.

**Done when:** all six jobs pass.

#### Phase 4: Postgres-backed contract test

**Goal:** `contract-test` job runs against a Postgres 16 service container.

- `services: postgres: { image: postgres:16-alpine, env: { POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB }, ports: ['5432:5432'], options: '--health-cmd pg_isready ...' }`.
- Workflow waits via `until pg_isready -h localhost; do sleep 2; done` (legitimate retry poll, not artificial sleep).
- Apply migrations via the existing `goose` tool (pinned in `go.mod:423`) before running `make contract-test`.
- Set `FALSEFLAG_TEST_DATABASE_URL` to point at the service.

**Done when:** job exits 0, all contract tests pass.

#### Phase 5: Multi-arch image builds

**Goal:** `build-images` job runs `make bake` for the `default` group across `linux/amd64,linux/arm64`.

- `docker/setup-qemu-action@v3` (registers binfmt handlers for arm64).
- `docker/setup-buildx-action@v3`.
- `actions/cache@v4` for `/tmp/.buildx-cache`, keyed on `runner.os + hashFiles('infra/Dockerfile*', 'go.sum', 'js/pnpm-lock.yaml')` with a restore-key on `runner.os`.
- Invoke bake with `--cache-to=type=local,dest=/tmp/.buildx-cache-new,mode=max` and `--cache-from=type=local,src=/tmp/.buildx-cache`. After build, move new cache over old to control growth.
- `make bake-print` first to validate the bake file parses.
- Do **not** push.

**Done when:** all five image targets build for both arches. First run may take 30+ minutes; subsequent runs with warm cache should drop to ~10.

#### Phase 6: Compose-up jobs (smoke + dashboard-e2e)

**Goal:** `smoke` and `dashboard-e2e` jobs bring up the compose stack, seed it, and run their respective E2E suites.

- Pre-step in each: install Hurl from upstream `.deb` release, pinned version (e.g. `hurl_<version>_amd64.deb` from `https://github.com/Orange-OpenSource/hurl/releases`). Choose the latest stable at implementation time.
- `smoke` job: `docker compose -f infra/compose.yaml up -d --build` → wait for `:8080/healthz` and `:8092/healthz` → `make seed` → `make smoke` (12 Hurl files, 73 requests) → `make mcp-smoke` redundantly skipped if covered. Decide whether to keep them as separate `Run` steps or merge.
- `dashboard-e2e` job: same compose-up + seed pattern → cache `~/.cache/ms-playwright` keyed on Playwright version (extract from `js/apps/dashboard/package.json:26`) → `pnpm --filter @falseflag/dashboard exec playwright install --with-deps chromium` (only downloads if cache miss) → `FALSEFLAG_DASHBOARD_URL=http://127.0.0.1:3000 make dashboard-e2e`.
- After both jobs: `docker compose -f infra/compose.yaml logs > compose-logs.txt` and upload as artifact on failure for debuggability.

**Done when:** both jobs exit 0 against a seeded stack. Hurl shows 73/73, Playwright shows 1/1.

#### Phase 7: Kind cluster job

**Goal:** `kind-smoke` runs `make kind-smoke` inside a kind cluster.

- `helm/kind-action@v1` with a pinned `node_image: kindest/node:v1.31.x` (latest 1.31 minor at implementation time).
- Build operator image locally (single-arch, `linux/amd64`) and `kind load docker-image` it.
- `azure/setup-kubectl@v4` and `azure/setup-helm@v4` with pinned versions.
- Run `make kind-smoke`. Job timeout: 20 minutes (kind boot + operator rollout + flag-binding wait).

**Done when:** operator reconciles all CRDs and `kind-smoke.sh` exits 0.

#### Phase 8: Timing aggregator + docs

**Goal:** `timing-summary` job consolidates all `timing-*.json` into one summary, and `docs/ci-baseline.md` explains the workflow.

- `timing-summary` job: `needs: [generate-check, lint-go, lint-js, typecheck-js, test-go, test-go-race, test-js, build-js, contract-test, conformance, build-images, smoke, dashboard-e2e, kind-smoke]`, `if: always()`.
- Steps: `actions/download-artifact@v4` with `pattern: timing-*` and `merge-multiple: true` → `scripts/ci/timing-summary.sh` (or `.js`) reads all JSONs, computes workflow wall time, emits `timing-summary.json` + `timing-summary.md`. Pipe Markdown into `$GITHUB_STEP_SUMMARY`. Upload `timing-summary.json` as artifact.
- Schema (must be stable for slice 7b):
  ```json
  {
    "workflow": "ci-baseline",
    "run_id": "<github.run_id>",
    "commit": "<sha>",
    "started_at": "<ISO8601>",
    "ended_at": "<ISO8601>",
    "wall_clock_s": 1234,
    "jobs": [
      { "job": "lint-go", "start_ts": 1234567890, "end_ts": 1234567990, "duration_s": 100, "outcome": "success", "runner_os": "Linux" },
      ...
    ]
  }
  ```
- `docs/ci-baseline.md` covers:
  - What the workflow does, job by job (table).
  - Why it's slow (multi-arch QEMU, cold buildx cache per branch, sequential compose-up per E2E job, no Depot Cache).
  - How to read the timing artifact (`gh run download <id> -n timing-summary`).
  - What slice 7b will replace (Depot runners, Depot Cache, Depot container builds) — without giving the demo away.

**Done when:** a workflow run shows a complete Markdown timing table in the run's Summary tab, and `timing-summary.json` is downloadable.

#### Phase 9: Verify + METAPLAN status note

**Goal:** prove the baseline runs end-to-end on the real Zagrit-HQ/false-flag repo and record the baseline numbers.

- Push the slice 7a branch to `Zagrit-HQ/false-flag`.
- Open a PR; wait for full CI run.
- Record:
  - Per-job wall times (first run, cold cache).
  - A second run on the same PR (warm cache; documents the natural caching delta — separate from Depot's contribution).
  - Workflow wall-clock time (cold and warm).
- Write the slice 7a status note into `docs/METAPLAN.md` per the convention of slices 1–6, including:
  - Verification command list with green checkmarks.
  - Known gaps deferred (image push, branch protection, path filters — all explicitly out of scope).
  - Baseline timing numbers (the headline figures slice 7b will improve).
- Tick the `[ ] Plan slice 7a` and `[ ] Implement slice 7a` and `[ ] Verify slice 7a locally` boxes.

**Done when:** the slice 7a PR has a fully green check column with the timing summary visible in the Actions UI, and METAPLAN reflects completion.

## System-Wide Impact

**Interaction graph.** Slice 7a doesn't change any product code paths, only CI. But it triggers every existing make-target chain: the generate path (proto → Go, sql → Go, CRD types → CRDs, OpenAPI → Go + JS), the test path (gotestsum, vitest, playwright), the smoke path (compose up → seed → hurl), and the operator path (kind + apply). Every chain runs on every PR.

**Error propagation.** Each job is independent and surfaces failure as its own status check. Failures do not cascade. The timing aggregator runs `if: always()` so partial failures still produce a timing artifact (with `outcome: failure` on the failed jobs).

**State lifecycle.** All state is per-job: kind clusters torn down by `helm/kind-action`, compose stacks left to be GCed with the runner VM, Docker layer cache pruned by the cache-rotation step in Phase 5. No persistent state on the runner.

**API surface parity.** The workflow exercises every public surface this repo has — REST (`:8080`), ConnectRPC (`:8090`), proxy (`:8081`), MCP (`:8091`), dashboard (`:3000`), and operator (kind). Slice 7b will mirror this exactly via Depot.

**Demo narrative.** Slice 7a's per-job and total wall-clock numbers become the "before" picture. Conference attendees will see two GitHub Actions runs at the talk; slice 7b's run-summary table is meant to differ visibly from this one. The timing JSON schema is the contract between the two slices.

## Acceptance Criteria

### Functional

- [ ] `.github/workflows/ci.yml` exists and validates with `actionlint` (or equivalent) without errors.
- [ ] Workflow triggers on `pull_request` to `main`, `push` to `main`, and `workflow_dispatch`.
- [ ] `concurrency` is set with `cancel-in-progress: true`.
- [ ] All 14 leaf jobs run on `ubuntu-latest` and reach status `success` on a clean PR against `Zagrit-HQ/false-flag`'s `main`.
- [ ] `make lint-go` (golangci-lint) runs and exits 0 in the `lint-go` job (first time end-to-end in CI — closes the deferral noted six times in METAPLAN status).
- [ ] `make dashboard-e2e` runs and exits 0 in the `dashboard-e2e` job (first time E2E executed in CI — closes the slice 5 deferral at `docs/METAPLAN.md:703`).
- [ ] `make kind-smoke` runs and exits 0 in the `kind-smoke` job.
- [ ] `make contract-test` runs against a real Postgres 16 service container and exits 0.
- [ ] `docker buildx bake` builds all five `default`-group targets for both `linux/amd64` and `linux/arm64`.
- [ ] Per-job `timing-<job>.json` artifacts upload successfully for all jobs, including failed ones.
- [ ] `timing-summary` aggregator job produces `timing-summary.json` and writes a Markdown table to `$GITHUB_STEP_SUMMARY`.

### Non-Functional

- [ ] **No artificial sleeps.** `grep -nE '^\s*(- run:.*\bsleep\b|sleep [0-9])' .github/workflows/ci.yml` returns no hits other than legitimate readiness/retry loops (`pg_isready`, kind health, healthz probes). Note the exceptions in code comments.
- [ ] **No deliberate pessimization.** No stripped caches, no forced `--no-cache` on docker, no `--cache-from=type=registry,ref=nowhere`, no per-step `actions/setup-go` re-installs within a single job.
- [ ] **All third-party actions pinned** to a major version tag (preferable: SHA, but tags acceptable for slice 7a). No `@main` or `@latest`.
- [ ] **All tool versions pinned**: Go 1.26.0, Node 22, pnpm 10.0.0, Hurl (pinned to specific release), kind `node_image` pinned to a specific patch version, kubectl + helm pinned.
- [ ] Workflow file is under 600 lines (a guideline; if longer, justify in PR description).

### Quality Gates

- [ ] Job names are stable and self-documenting (no auto-generated matrix expansions); future branch protection rules can name them directly.
- [ ] `docs/ci-baseline.md` exists and explains: what the workflow does, why it's slow, how to read the timing artifact, what slice 7b will replace.
- [ ] METAPLAN slice 7a status note added with verification list and baseline timing numbers.
- [ ] METAPLAN checklist boxes ticked for slice 7a (plan, implement, verify).

## Out of Scope (Slice 7b owns these)

- Depot runners, Depot Cache, Depot container builds.
- An optimized workflow file (`ci-depot.yml` or equivalent).
- Before/after comparison docs at the README level.
- Pushing images to `ghcr.io` (no `packages: write` permission requested).
- Branch protection rules and required status checks (the maintainer can configure these in the repo settings independently).
- Path filters (`on: pull_request: paths: ...`).
- Renovate/Dependabot configuration.
- CI for slice 8 polish work (loadgen integration, README screenshots, etc.).

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| **QEMU OOMs on arm64 dashboard build.** `Dockerfile.dashboard` is memory-heavy (pnpm install + Vite + Remix bundle) and ubuntu-latest has 16 GB. | Set `NODE_OPTIONS=--max-old-space-size=4096` in the dashboard build stage if needed. If still OOMs, fall back to `linux/amd64`-only for the dashboard target while keeping multi-arch for the four Go services. Document the carve-out. |
| **Playwright browser download rate-limited.** First run downloads ~130 MB. | `actions/cache` keyed on Playwright version. Cache hit rate should be 100% after first run per branch. |
| **kind cluster boot is flaky.** Known intermittent failure mode on shared runners. | 20-minute job timeout (hard ceiling), no retry. Slice 7a's job goes red on flake; that's accurate baseline information. Slice 7b is not responsible for fixing kind flakiness. |
| **`smoke.sh` truncate races with compose-up.** Script has no retry loop on the healthz probe (line 18). | Add a workflow-level `until curl -fsS http://localhost:8080/healthz; do sleep 2; done` retry loop *before* invoking `make smoke`. This is legitimate readiness polling, not artificial slowness. |
| **`generate-check` false positive on dep-only PRs.** A `go.sum` bump could trigger buf-generated file drift. | Accepted. Real teams hit this and either re-run generate or update the lockfile. We don't add path filters; we document the gotcha in `docs/ci-baseline.md`. |
| **Hurl release URL drifts.** Pinned `.deb` URL may 404 if Orange-OpenSource reorganizes releases. | Pin to a known-good Hurl version at implementation time. If it 404s in the future, slice 7b's maintainer-owned work likely replaces the install anyway. |
| **First run is much slower than steady state.** Buildx cache cold, no module cache, no pnpm store. | This *is* part of the baseline story. Record both cold and warm timings in the verification step. |
| **Workflow YAML grows unwieldy.** 14 jobs + 2 composite actions can balloon. | Extract repeated setup blocks into composite actions only where the repetition exceeds ~3 jobs. Composite actions kept under `.github/actions/`. |

## Sources & References

### Internal

- **Origin slice (METAPLAN section):** `docs/METAPLAN.md:469-498` (slice 7a definition).
- **Ideation doc:** `docs/ideation/2026-05-20-synthetic-feature-flag-platform-depot-demo-ideation.md` — "Suggested CI Shape" + rejection of artificial sleeps.
- **Prior slice status notes** in `docs/METAPLAN.md`:
  - `:599`, `:711`, etc.: "golangci-lint still not run end-to-end" (recurring deferral closed by Phase 2).
  - `:703`: "Slice 7 owns the CI wiring" for Playwright (closed by Phase 6).
- **Existing make targets:** `Makefile` (full inventory in research notes — every target used by CI is preexisting).
- **Bake config:** `infra/docker-bake.hcl` — `default` group, multi-arch platforms, REGISTRY var.
- **Dockerfiles:** `infra/Dockerfile` (Go services, distroless), `infra/Dockerfile.dashboard` (Remix SSR, Node 22).
- **Compose stack:** `infra/compose.yaml` — 6 services, db healthcheck.
- **Hurl corpus:** `tests/hurl/*.hurl` — 12 files, 73 requests; runner is `scripts/smoke.sh`.
- **Playwright config:** `js/apps/dashboard/playwright.config.ts` — Chromium-only, single worker, headless, `webServer` block to bypass.
- **Conformance corpus:** `js/packages/shared-eval-corpus` — 25 fixtures, Go + JS consumers via `make conformance`.

### External (commodity GitHub Actions ecosystem)

- `actions/checkout@v4`, `actions/setup-go@v5`, `actions/setup-node@v4`, `actions/cache@v4`, `actions/upload-artifact@v4`, `actions/download-artifact@v4`.
- `pnpm/action-setup@v4`.
- `docker/setup-qemu-action@v3`, `docker/setup-buildx-action@v3`.
- `helm/kind-action@v1`, `azure/setup-helm@v4`, `azure/setup-kubectl@v4`.
- Hurl release archive: `https://github.com/Orange-OpenSource/hurl/releases` (pin specific tag at implementation time).

### Related Plans

- Slice 1: `docs/plans/2026-05-20-001-feat-foundation-monorepo-scaffold-plan.md` (Make target layer this slice builds on).
- Slice 5: `docs/plans/2026-05-20-005-feat-dashboard-cli-sdks-plan.md` (Playwright config this slice finally exercises in CI).
- Slice 6: `docs/plans/2026-05-20-006-feat-mcp-server-plan.md` (MCP smoke this slice runs).
