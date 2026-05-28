# Metaplan: Autonomous Build Workflow

## Goal

Build a believable, intentionally large feature flag platform for a conference demo about accelerating slow CI/CD with Depot.

The software should look and feel like it works. It does not need production-grade correctness, security, or operational hardening. Prefer broad credible surface area, passing local commands, visible workflows, and real build/test complexity over deep product polish.

## Operating Mode

Use this file as the coordination checklist. When the user asks "what's next?", inspect this checklist and the dependency/fan-out map below. If the next incomplete work is a sequential gate, continue that gate. If multiple unblocked items can safely run in parallel, propose or start the parallel set with explicit write ownership.

Default behavior:

- Work autonomously within the current slice.
- Use the ideation docs as source material.
- Prefer demo-quality implementation over production-quality hardening.
- Stub internals when needed, but keep public commands, builds, tests, and UI paths believable.
- Use parallel subagents only when write scopes are clearly separate.
- Avoid asking the user for decisions unless the choice affects the demo direction or blocks progress.

Reference docs:

- `docs/ideation/2026-05-20-synthetic-feature-flag-platform-depot-demo-ideation.md`
- `docs/ideation/2026-05-20-moonconfig-historical-reference.md`
- `/Users/wito/code/project-keat` for read-only inspiration and Git history

## Skill Strategy

Use `compound-engineering:ce-plan` for each major slice before implementation. Plans should be concrete enough for autonomous work, but scoped enough that implementation can finish without turning into an open-ended rewrite.

Use normal implementation work or `compound-engineering:ce-work` after a plan exists.

Use `compound-engineering:lfg` only after the repo has a working skeleton and the task is bounded enough for the full autonomous pipeline: plan, work, review, test, commit, PR, and CI watch. Early greenfield scaffolding is better handled with `ce-plan` plus direct implementation, because `lfg` is optimized for PR-ready tasks, not shaping the first version of a synthetic product.

## Parallel Subagent Guidance

Parallel subagents are useful once the slice can be split by ownership.

Good parallel splits:

- Go API/proxy/config model
- Kubernetes CRDs/operator
- TypeScript dashboard/CLI/SDK under `js/**`
- CI, Docker, and build tooling
- Tests, fixtures, and golden evaluation corpus

Avoid parallel work when multiple agents would edit the same root package structure, shared types, generated artifacts, or build configuration at the same time. Establish the skeleton first, then fan out.

## Dependency And Parallelization Map

Treat this as the execution graph for "what's next?" decisions.

### Sequential Gates

These should happen mostly in sequence because later work depends on shared structure or contracts:

1. **Foundation plan** before any implementation.
2. **Foundation implementation** before broad parallelization. This establishes root tooling, package layout, Go module shape, pnpm workspace, Turborepo, shared commands, and codegen placeholders.
3. **Foundation verification** before starting multiple feature workers.
4. **Shared config/release model** at the start of the configuration slice before splitting JSON/CEL/TypeScript strategy workers.
5. **API contract skeleton** before dashboard, CLI, SDKs, MCP, and operator deeply integrate with generated clients.
6. **Final integration verification** after each parallel fan-out group.

### Safe Parallel Fan-Out Groups

Once the listed prerequisites are complete, these can run in parallel.

#### Fan-Out A: Configuration Strategies

Prerequisites:

- Foundation implemented and verified.
- Configuration strategy plan exists.
- Shared normalized config/release model is created.

Parallel workers:

- **JSON config worker:** owns JSON config compiler, fixtures, and tests.
- **CEL config worker:** owns CEL compiler/evaluator boundary, fixtures, and tests.
- **TypeScript config worker:** owns `js/packages/config-ts/**` and TypeScript config-as-code validation/sandbox boundary.
- **Persistence worker:** owns SQLC migrations/queries for config source, compiled snapshots, and audit records.

Integration owner:

- Main thread integrates all strategies behind the project-scoped config interface and runs the full config test suite.

#### Fan-Out B: Contracts, Operator, And Clients

Prerequisites:

- Foundation implemented and verified.
- API/gRPC/OpenAPI plan exists.
- Initial proto/OpenAPI/resource model skeleton exists.

Parallel workers:

- **API contracts worker:** owns proto/OpenAPI definitions, generated Go/TypeScript clients, and contract tests.
- **Operator worker:** owns `operator/**`, CRDs, controller-runtime skeleton, Helm, and Kustomize.
- **Go runtime worker:** owns API/proxy service handlers that consume the generated contracts.
- **Client preparation worker:** owns generated TypeScript client wiring and SDK/client package placeholders.

Integration owner:

- Main thread owns generated artifact reconciliation, root commands, and cross-surface compile/test verification.

#### Fan-Out C: User-Facing Surfaces

Prerequisites:

- Foundation implemented and verified.
- Enough API/generated-client shape exists for mocked or demo-backed clients.
- Client surfaces plan exists.

Parallel workers:

- **Dashboard worker:** owns `js/apps/dashboard/**`, Remix routes, Tailwind/Radix components, Vitest tests, and Playwright coverage.
- **CLI worker:** owns `js/apps/cli/**`, Commander commands, CLI fixtures, and tests.
- **TypeScript SDK worker:** owns `js/packages/sdk-js/**` and TypeScript SDK tests.
- **Go SDK/proxy worker:** owns `internal/sdkgo/**`, `internal/proxy/**`, local snapshot evaluation, and Go SDK tests.
- **Golden corpus worker:** owns shared evaluation fixtures and conformance tests.

Integration owner:

- Main thread runs workspace-wide TypeScript checks, Go tests, browser smoke tests, and SDK conformance checks.

#### Fan-Out D: MCP And CI

Prerequisites:

- API contract skeleton exists.
- At least one generated or hand-written API client path exists.
- Enough build/test surfaces exist to make CI meaningful.

Parallel workers:

- **MCP worker:** owns `cmd/platformcon-mcp/**` or equivalent MCP service files, tools, tests, and container target.
- **CI/build worker:** owns `.github/**`, `docker-bake.hcl`, Dockerfiles, cache wiring, Depot workflow, and timing artifacts.
- **Docs/demo worker:** owns README/demo script updates that describe MCP and CI usage.

Integration owner:

- Main thread verifies local commands, generated-code checks, Docker/build commands where available, and updates the metaplan checklist.

### Parallel Work Rules

- Every parallel worker must have an explicit owned path list.
- Avoid parallel edits to root files: `go.mod`, `go.sum`, `js/package.json`, `js/pnpm-lock.yaml`, `js/pnpm-workspace.yaml`, `js/turbo.json`, `Makefile`, `buf.yaml`, `sqlc.yaml`, and generated artifacts. If root files must change, the main thread owns them or assigns them to exactly one worker.
- Shared contracts come first. Do not split workers across dashboard/CLI/SDK/operator if the API model they depend on is still changing.
- Generated artifacts are integration points. Prefer one owner for generation commands and generated output.
- After every fan-out, the main thread reconciles changes, runs verification, and updates this checklist.

### "What's Next?" Response Rule

When asked "what's next?":

1. Read the checklist and status notes.
2. Identify unblocked sequential gates and fan-out groups.
3. If only one gate is unblocked, start or propose that gate.
4. If a fan-out group is unblocked, propose the parallel worker set with owned paths and a short integration plan.
5. If the user asked for maximum autonomy, start the unblocked fan-out directly with subagents when available.
6. If no subagents are available, execute the same work sequentially in the main thread, preserving the same ownership order.

## Quality Bar

Demo quality means:

- The repo layout is coherent.
- Main services have clear entrypoints.
- CLI commands and dashboard flows exist, even if backed by simple mocked or seeded data.
- The dashboard is a Remix app using Tailwind for styling and Radix UI for custom components.
- TypeScript linting/formatting runs through Biome, and TypeScript tests run through Vitest.
- Backend relational database access uses SQLC-generated Go query packages.
- Go backend logging uses the standard `log/slog` package.
- The chosen stack is ConnectRPC or gRPC with Buf, `oapi-codegen`, `pgxpool` + SQLC + goose, `go-redis`, OpenTelemetry + Prometheus, `golangci-lint` + `gotestsum`, pnpm workspaces + Turborepo, Zod + Orval, Commander, Playwright, and Kubebuilder/controller-runtime + Helm + Kustomize.
- Go code follows the `/Users/wito/code/project-depot/registry` shape: `cmd/**` entrypoints, `internal/**` implementation packages, `internal/db` SQLC output, root `proto`, and `tests/hurl`. Do not create `pkg/**`.
- TypeScript is isolated under `js/**`, including the pnpm workspace, Turborepo config, Remix dashboard, CLI, config DSL, generated clients, and TypeScript SDK.
- APIs and generated specs are plausible.
- Tests exercise real code paths.
- CI has meaningful slow surfaces: Go tests, TypeScript builds, generated code checks, Docker builds, browser tests, Kubernetes/operator tests, and backend matrices.
- Depot-optimized workflows show credible acceleration paths.

Demo quality does not require:

- hardened auth
- full multi-tenant security
- production-grade sandboxing
- perfect CRD versioning
- complete OpenFeature compliance
- resilient distributed systems behavior
- complete UI polish

## Validation Ladder

Every implementation slice must end with validation. Do not mark a slice verified unless at least one compile/build command and one smoke/demo check for that slice have been run successfully, or the unmet check is explicitly recorded as a known gap in status notes.

### 1. Compile Contract

Purpose: prove the repo is not just a pile of files.

Run the relevant subset for the slice:

```bash
go test ./...
pnpm install
pnpm lint
pnpm typecheck
pnpm test
pnpm build
```

As generation surfaces appear, include:

```bash
buf generate
sqlc generate
controller-gen ...
go tool oapi-codegen ...
pnpm orval
```

Criteria:

- Go packages compile.
- TypeScript packages typecheck.
- Generated code is reproducible.
- Unit tests pass for touched packages.
- Documented commands are not obviously broken.

### 2. HTTP/API Contract

Purpose: prove the API starts and behaves plausibly.

Use Hurl for HTTP e2e and smoke tests against the API server.

Expected shape:

```bash
make api-dev
hurl --test test/hurl/*.hurl
```

Minimum Hurl coverage over time:

- health endpoint returns OK
- project list/create happy path
- flag list/create happy path
- config validate endpoint for JSON, CEL, and TypeScript modes
- snapshot compile/export endpoint
- evaluation trace or flag evaluation endpoint
- audit event/search endpoint

Criteria:

- API server starts locally.
- Hurl tests exercise real HTTP requests and assert status codes plus key JSON fields.
- Failures are fixed before verification, unless the endpoint is intentionally deferred and recorded.

### 3. Runtime Smoke Contract

Purpose: prove compiled pieces can run together enough to look real.

Expected smoke paths as slices mature:

- API starts and `/healthz` returns OK.
- Proxy starts and can return a static or compiled flag evaluation.
- Dashboard starts and renders meaningful seeded data.
- CLI lists projects/flags from seed data or the API.
- Go SDK and TypeScript SDK evaluate a seeded flag from a local snapshot.
- Operator binary starts or CRD manifests validate.
- At least one Go service image and the dashboard image build.

Criteria:

- A user-visible path works without inspecting internals.
- CLI output looks product-like.
- Dashboard screens show seeded product data, not only placeholders.
- SDK examples print believable evaluation results.

### 4. Demo Contract

Purpose: prove the project can be presented.

By the later slices, maintain:

- README quickstart that works.
- Seed data.
- One happy path script:
  - create/list project
  - validate config
  - compile snapshot
  - evaluate flag
  - show audit/evaluation trace
- Playwright coverage for the dashboard happy path.
- Hurl coverage for the API happy path.
- Baseline CI workflow and Depot-optimized workflow.
- Documented before/after CI story.

Criteria:

- A conference audience can understand the product shape from the dashboard, CLI, API, and CI output.
- The demo does not depend on production hardening.

### 5. Verification Record

After each implementation or fan-out integration, update status notes with:

- commands run
- checks passed
- checks skipped and why
- known gaps deferred to later slices

If a command is expected to pass and fails, fix it before marking the slice verified.

## Recommended `ce-plan` Sequence

Run these in order. Each plan should be saved under `docs/plans/`.

### 1. Foundation Monorepo Plan

Prompt:

```text
Use ce-plan to plan the first implementation slice for the PlatformCon feature flag demo.

Source docs:
- docs/ideation/2026-05-20-synthetic-feature-flag-platform-depot-demo-ideation.md
- docs/ideation/2026-05-20-moonconfig-historical-reference.md

Goal: scaffold a believable Go/TypeScript monorepo for the demo.

Quality bar: demo-quality, not production-quality. Prefer breadth and visible working paths over hardening.

Include:
- Go module and service entrypoints for API, proxy, and operator
- Go repository structure based on `/Users/wito/code/project-depot/registry`: `cmd/**`, `internal/**`, root `proto`, root `sqlc.yaml`, and `tests/hurl`
- no Go `pkg/**` tree
- shared `log/slog` setup for Go binaries
- TypeScript workspace under `js/**` for Remix dashboard, CLI, SDK, and config DSL
- Tailwind, Radix UI, Biome, and Vitest wired into the TypeScript workspace
- pnpm workspaces and Turborepo as the TypeScript monorepo layer
- baseline tooling stubs for Buf, `oapi-codegen`, SQLC, goose, controller-gen, `golangci-lint`, and `gotestsum`
- Hurl API smoke/e2e test directory at `tests/hurl/**` and placeholder test command
- shared repo conventions and Makefile/task commands
- Dockerfiles or a docker-bake shape
- minimal tests that prove the skeleton builds
- baseline places for generated OpenAPI/gRPC/CRD/SQLC artifacts

Do not implement deep business logic in this slice. Establish the structure that later slices can fill in.
```

### 2. Configuration Strategies Plan

Prompt:

```text
Use ce-plan to plan the configuration strategy slice.

Source docs:
- docs/ideation/2026-05-20-synthetic-feature-flag-platform-depot-demo-ideation.md
- docs/ideation/2026-05-20-moonconfig-historical-reference.md

Goal: implement project-scoped configuration strategies where each project chooses one mode: JSON, CEL, or TypeScript config-as-code.

Quality bar: demo-quality. The strategies should compile into the same normalized release snapshot. It is acceptable for some internals to be simplified, but each strategy needs visible validation/compile paths and tests.

Include:
- shared normalized config/release model
- SQLC-backed persistence shape for config source, compiled snapshots, and audit records
- JSON config compiler
- CEL config compiler using targeting expressions
- TypeScript config-as-code DSL package and validation flow
- sandbox/WASM concept stub or prototype boundary
- CLI/API validation commands
- tests and fixtures for each strategy
```

### 3. API, gRPC, and OpenAPI Plan

Prompt:

```text
Use ce-plan to plan the API contract slice.

Goal: add both gRPC and OpenAPI surfaces to the PlatformCon feature flag demo.

Quality bar: demo-quality. The API should look coherent and generate clients/specs, but does not need complete production behavior.

Include:
- control-plane API resources for projects, environments, flags, segments, config strategies, snapshots, and audit events
- ConnectRPC or gRPC with Buf-managed protobuf definitions
- `oapi-codegen` for OpenAPI REST generation
- SQLC-generated query layer for relational persistence behind the control-plane API
- `pgxpool`, SQLC, and goose as the relational persistence stack
- gRPC/Connect service definitions for internal API/proxy/operator communication
- OpenAPI REST surface for dashboard, CLI, and public admin APIs
- generated Go and TypeScript clients
- Orval-generated TypeScript client and Zod validation schemas where useful
- contract tests showing REST and gRPC agree for key happy paths
- Hurl e2e tests for API health, project, flag, config validation, snapshot, evaluation, and audit endpoints
- generated-code check in CI, including SQLC
```

### 4. Kubernetes Operator and CRDs Plan

Prompt:

```text
Use ce-plan to plan the Kubernetes-native slice.

Source inspiration:
- /Users/wito/code/project-keat
- legacy Keat CRD/operator ideas from the ideation docs

Goal: add Kubernetes CRDs and an operator skeleton for feature flag configuration.

Quality bar: demo-quality. CRDs and reconciliation should look believable, with status updates and tests, but can use simplified reconciliation.

Include:
- Project, Environment, Flag, Segment, RolloutPolicy, FlagBinding, and FlagSnapshot CRDs
- Kubebuilder/controller-runtime operator structure
- sample manifests, Helm chart, and Kustomize overlay
- envtest or controller unit tests
- kind smoke test shape
- integration with the normalized config/release model
```

### 5. Dashboard, CLI, and SDKs Plan

Prompt:

```text
Use ce-plan to plan the client surfaces slice.

Goal: make the software look usable from a dashboard, CLI, and SDKs.

Quality bar: demo-quality. Prioritize believable workflows and visible screens/commands over complete behavior.

Include:
- Remix dashboard pages for projects, flags, config strategy selection, config validation, rollout snapshots, audit events, and evaluation trace
- Tailwind styling, Radix UI component primitives, Biome lint/format, and Vitest tests
- Commander-based CLI commands for login stub, project list, flag list, config validate, config save, snapshot export, and smoke-test
- Go SDK and TypeScript SDK with local snapshot evaluation
- OpenFeature-inspired provider shapes
- shared golden evaluation corpus
- browser tests and SDK conformance tests
```

### 6. MCP Server Plan

Prompt:

```text
Use ce-plan to plan the MCP server slice.

Goal: add an agent-facing MCP server that can manage and inspect feature flags through the control-plane API.

Quality bar: demo-quality. It should expose useful tools and pass tool-call tests, but does not need hardened auth.

Include:
- separate Go binary for platformcon-mcp
- tools for listing projects, listing flags, getting flag details, validating config, explaining evaluations, and searching audit logs
- gRPC or generated API client integration
- JSON schema/tool input validation
- tests for tool calls, error formatting, and permissions stubs
- container image and CI smoke test
```

### 7. Slow CI and Depot Acceleration Plan

Slice 7 is split into two halves so the slow baseline can land independently from the Depot-accelerated version. Slice 7a is the codified slow baseline (done by Claude). Slice 7b is the Depot acceleration story that the maintainer will land themselves in preparation for the demo.

#### 7a. Slow CI Baseline

Prompt:

```text
Use ce-plan to plan the slow baseline CI slice (7a only — do not plan Depot acceleration; that is slice 7b).

Goal: create a slow but credible baseline CI pipeline that resembles what you would actually find in the wild on a project of this shape. The pipeline should be slow because it performs real build/test work across many surfaces, not because it is artificially unoptimized.

Quality bar: demo-quality and realistic. Avoid artificial sleeps. Do not deliberately pessimize the workflow (no stripped caches, no forced cold starts, no contrived sequential steps that a sane team would parallelize). Use ordinary GitHub-hosted runners, ordinary actions/cache, and ordinary docker/buildx — the kind of setup a competent team ships before they hear about Depot.

Include:
- baseline GitHub Actions workflow on default GitHub-hosted runners
- docker-bake file for multiple images, built via standard buildx
- Go, TypeScript, generated code, Docker, Kubernetes, browser, SDK, config compiler, and backend jobs
- Buf, SQLC, controller-gen, and OpenAPI generation checks
- `golangci-lint`, `gotestsum`, Biome, Vitest, and Playwright jobs
- Hurl API e2e job against the API server
- timing summary artifact that slice 7b can later compare against
- docs covering how to read the timings and what slice 7b will replace

Explicitly out of scope for 7a:
- Depot runners, Depot Cache, Depot container builds
- the optimized workflow file
- before/after comparison docs (slice 7b owns these)
```

#### 7b. Depot Acceleration (deferred — owned by maintainer)

Not planned or implemented by Claude. The maintainer will land this directly in preparation for the conference talk.

Scope when picked up:
- Depot-optimized GitHub Actions workflow using Depot runners, Depot Cache, and Depot container builds
- timing artifact emitted in the same shape as 7a so the two can be diffed
- docs explaining how to run the comparison and what the demo audience should look at

### 8. Polish and Demo Script Plan

Prompt:

```text
Use ce-plan to plan the final demo polish slice.

Goal: make the repository easy to present at a conference.

Quality bar: demo-quality. Focus on the audience seeing the product shape and the CI acceleration story quickly.

Include:
- README quickstart
- architecture diagram
- demo script
- seed data
- screenshots or browser-verifiable dashboard path
- one-command local smoke test
- one-command CI comparison explanation
- cleanup of obvious broken commands
```

### 10. SQLite Backend Plan (self-hosted-tool CI weight)

Slices 10–12 deliberately add the kinds of cross-cutting friction common to self-hosted tools (Gitea, Mealie, Vaultwarden, Forgejo, Nextcloud, Immich, Paperless-ngx). The goal is *more authentic slow CI surface area to optimize*, not better engineering. They stack multiplicatively: dual backends × dual architectures × multiple deployment artifacts.

Slice 10 adds SQLite as a second storage engine alongside Postgres. Everything that currently runs once against Postgres now also runs against SQLite. This is the canonical self-hosted-tool pattern and is the single highest-impact way to broaden the "real-world data layer" surface that CI has to chew through.

Prompt:

```text
Use ce-plan to plan a SQLite storage backend alongside the existing Postgres backend.

Goal: turn the persistence layer into a dual-backend system that every self-hosted tool eventually grows. The product reads identically; CI and codegen now do roughly 2× the work.

Quality bar: demo-quality. The SQLite path must work end-to-end in compose and pass the same integration suite as Postgres, but it does not need to be performance-tuned or production-hardened.

Include:
- Parallel goose migrations under `db/migrations/sqlite/**` translating uuid→TEXT, jsonb→TEXT, timestamptz→TEXT, gen_random_uuid()→app-side, and rewriting `DISTINCT ON` / `now()` / `::type` casts.
- A second `sqlc.yaml` config (or `sqlc.sqlite.yaml`) emitting to `internal/db/sqlite/**` with `engine: sqlite`, `sql_package: database/sql`, and a SQLite-flavored copy of `db/queries/**`.
- A `Store` interface refactor in `internal/store/**` that hides `pgxpool.Pool`, removes the `Pool()` getter, and lets the API pick a backend from the DSN scheme (`postgres://` vs `sqlite://`).
- A shared domain-type conversion layer so callers never see `pgtype.UUID`/`pgtype.Timestamptz`; SQLite returns the same domain shapes.
- UUID generation moved to Go (`google/uuid`) for both backends.
- A second compose service variant (or a `compose.sqlite.yaml` overlay) booting the API with an embedded SQLite file in a named volume.
- Integration tests in `internal/store/integration_test.go` parametrized over both backends via subtests; the existing pgtestcontainer path stays, a new `t.TempDir()`-backed SQLite path runs alongside.
- Hurl smoke suite (`tests/hurl/**`) executed against both backends; `make smoke` either runs both sequentially or a new `make smoke-sqlite` companion target.
- CI matrix: every Go test job and every Hurl smoke job fans out over `{postgres, sqlite}`.
- Documentation: a self-hoster-flavored README section explaining "single binary + SQLite" vs "compose + Postgres" and when to pick which.

Owned paths:
- `db/migrations/sqlite/**`, `db/queries/sqlite/**` (or per-engine subfolders)
- `internal/db/sqlite/**`
- `internal/store/**` (interface refactor; coordinate with the Postgres implementation file split)
- `sqlc.yaml` (or new `sqlc.sqlite.yaml`)
- `compose.sqlite.yaml`
- `.github/workflows/ci.yml` (backend matrix axis)
- `tests/hurl/**` smoke-runner wrapper, `Makefile` smoke targets

Out of scope for this slice:
- SQLite-only optimizations (FTS5, virtual tables).
- Backup/restore tooling (slice 12 owns that).
- Migration parity checking — both schemas drift independently, by design.
```

### 11. Multi-Arch Container Images Plan

Slice 11 makes every container image we ship build for `linux/amd64` and `linux/arm64`. ARM support is table stakes for self-hosted tools because of the Raspberry Pi / Apple Silicon / cheap-ARM-VPS crowd. This roughly doubles container build time (QEMU emulation for cross-compiled steps and emulated smoke tests) and stresses the Dockerfile cache story.

Prompt:

```text
Use ce-plan to plan multi-arch container builds (linux/amd64 + linux/arm64) for every shipped image.

Goal: ship credible ARM support the way every self-hosted tool does, and add a second axis to the container-build CI surface so that the bake/push step becomes a meaningful optimization target.

Quality bar: demo-quality. ARM images must build, boot, and pass a smoke probe (`/healthz` or equivalent) under QEMU emulation. They do not need to be perf-tested or run a full Playwright pass on ARM.

Include:
- `docker-bake.hcl` updated so every target has `platforms = ["linux/amd64", "linux/arm64"]`.
- Go binaries cross-compiled via `GOOS`/`GOARCH` in `infra/Dockerfile` and `infra/Dockerfile.dashboard`, using a single multi-stage Dockerfile rather than per-arch forks.
- A `tonistiigi/binfmt` (or `docker/setup-qemu-action`) step in CI to register QEMU for arm64 emulation.
- An emulated smoke job per image: pull the arm64 variant, boot it under qemu, hit `/healthz` and one canonical RPC, assert.
- Manifest list publication so a single `:latest` tag resolves to the right arch automatically.
- `make bake` and `make bake-print` produce both arches by default; a `make bake-fast` shortcut emits amd64-only for local iteration.
- Helm `values.yaml` and Kubernetes manifests reference the multi-arch tag (no hardcoded arch in `image:`).
- Documentation: README quickstart mentions ARM support; demo script shows the manifest list resolving on both an amd64 and arm64 host (or a single `docker manifest inspect` for the audience).

Owned paths:
- `docker-bake.hcl`, `infra/Dockerfile`, `infra/Dockerfile.dashboard`
- `.github/workflows/ci.yml` (QEMU setup, per-arch smoke matrix, manifest push)
- `Makefile` (bake targets)
- `deploy/**` (Helm values and kustomize overlays as applicable)

Out of scope for this slice:
- Native ARM runners (slice 7b can revisit if Depot ARM runners materialize).
- Windows or darwin images.
- Performance comparison between architectures.
```

### 12. Deployment Artifact Matrix Plan

Slice 12 turns the deployment story into the maximalist self-hosted-tool shape: docker-compose, Helm chart, raw Kubernetes manifests, and a systemd unit file, each lint/render/validate-checked in CI, with backup/restore round-trips per backend. Every realistic self-host tool ends up shipping at least three of these, and the validation matrix is a slow, embarrassing-but-real CI cost.

Prompt:

```text
Use ce-plan to plan a deployment artifact matrix covering docker-compose, Helm, raw Kubernetes manifests, and a systemd unit, with per-backend variants and a backup/restore round-trip check.

Goal: ship deployment surfaces that look like a mature self-hosted tool's `deploy/` directory, and add a slow, matrix-shaped CI surface that exercises every artifact on every PR.

Quality bar: demo-quality. Each artifact must pass its native validator and produce a bootable stack in CI for at least one configuration, but does not need to be production-tuned or feature-complete (e.g. no PodSecurityPolicy, no NetworkPolicies, no PVC sizing optimization).

Include:
- `deploy/compose/` — production-flavored compose variants for {postgres, sqlite}, each validated by `docker compose config` and booted in CI for a smoke probe.
- `deploy/helm/falseflag/` — a Helm chart with `values.yaml`, `templates/**`, and a `values-{postgres,sqlite}.yaml` example pair. Validated with `helm lint`, `helm template`, and `kubeconform` / `kubeval` on the rendered output. Goldenfile snapshot tests under `deploy/helm/falseflag/tests/__snapshots__/**` so chart drift is visible in PRs.
- `deploy/kubernetes/` — raw manifests (no Helm) for users who don't want a templating engine. Same kubeconform pass; small kustomize overlays for the {postgres, sqlite} variants.
- `deploy/systemd/` — `falseflag-api.service`, `falseflag-proxy.service`, and an `EnvironmentFile` example, validated with `systemd-analyze verify` in CI.
- Backup/restore round-trip: per backend, a CI job that seeds → backs up (`pg_dump` for PG, `.backup` for SQLite) → wipes → restores → re-asserts the Hurl smoke suite. Slow, deliberately.
- `make deploy-lint` and `make deploy-render` targets that run the whole local validation locally so contributors can iterate.
- CI matrix expansion: a `deploy` job that fans out over `{compose, helm, kubernetes, systemd}` × `{postgres, sqlite}` where applicable.
- Documentation: a `docs/deploy.md` page enumerating the supported surfaces with one paragraph each, mirroring how Mealie/Immich/Forgejo document their deployment options.

Owned paths:
- `deploy/**` (new top-level structure; the existing operator/Helm chart from slice 4 moves under `deploy/helm/` if appropriate)
- `.github/workflows/ci.yml` (deploy-matrix job)
- `Makefile` (`deploy-lint`, `deploy-render`, backup/restore targets)
- `docs/deploy.md`

Out of scope for this slice:
- Operator CRD changes (slice 4 owns those).
- Real production hardening (no NetworkPolicies, no RBAC tightening, no Secrets-Operator integration).
- Cross-cloud Terraform/Pulumi modules — out of genre for self-hosted tools.
```

## Implementation Checklist

- [x] Plan slice 1: Foundation monorepo
- [x] Implement slice 1: Foundation monorepo
- [x] Verify slice 1 locally
- [x] Plan slice 2: Configuration strategies
- [x] Implement slice 2: Configuration strategies
- [x] Verify slice 2 locally
- [x] Plan slice 3: API, gRPC, and OpenAPI
- [x] Implement slice 3: API, gRPC, and OpenAPI
- [x] Verify slice 3 locally
- [x] Plan slice 4: Kubernetes operator and CRDs
- [x] Implement slice 4: Kubernetes operator and CRDs
- [x] Verify slice 4 locally
- [x] Plan slice 5: Dashboard, CLI, and SDKs
- [x] Implement slice 5: Dashboard, CLI, and SDKs
- [x] Verify slice 5 locally
- [x] Plan slice 6: MCP server
- [x] Implement slice 6: MCP server
- [x] Verify slice 6 locally
- [x] Plan slice 7a: Slow CI baseline
- [x] Implement slice 7a: Slow CI baseline
- [x] Verify slice 7a locally
- [ ] Slice 7b: Depot acceleration (deferred — maintainer owns this)
- [x] Plan slice 8: Server-side TS compile + view/edit source UX
- [x] Implement slice 8: Server-side TS compile + view/edit source UX
- [ ] Verify slice 8 locally — Hurl `14-typescript-publish.hurl` 7/7 ✓; view route renders the fallback IR pretty-print as expected; compose API container rebuilt with phases 1-8 (`docker compose up --build` on the new root-level `compose.yaml`). **Remaining:** human or Playwright round-trip through the Monaco edit route — the spec at `js/apps/dashboard/playwright/edit-flag.spec.ts` is currently a graceful-skip skeleton; replace `test.skip(true, ...)` with a real save+redirect assertion, then `make dashboard-e2e`.
- [ ] Plan slice 9: Polish and demo script
- [ ] Implement slice 9: Polish and demo script
- [ ] Verify slice 9 locally
- [x] Plan slice 10: SQLite backend (dual storage engines)
- [x] Implement slice 10: SQLite backend
- [x] Verify slice 10 locally
- [ ] Plan slice 11: Multi-arch container images (amd64 + arm64)
- [ ] Implement slice 11: Multi-arch container images
- [ ] Verify slice 11 locally
- [ ] Plan slice 12: Deployment artifact matrix (compose, Helm, k8s, systemd) + backup/restore
- [ ] Implement slice 12: Deployment artifact matrix
- [ ] Verify slice 12 locally

## When To Use LFG

Use `compound-engineering:lfg` after slice 1 exists and the requested task is bounded.

Good LFG prompts:

```text
Use compound-engineering:lfg to add the CEL configuration strategy end to end based on the existing plan. Keep the quality bar demo-quality, not production-quality.
```

```text
Use compound-engineering:lfg to add the MCP server slice based on docs/plans/<plan-file>. Keep the implementation broad and believable, with passing tests, but avoid production hardening.
```

Avoid LFG for:

- the very first greenfield scaffold
- broad "implement everything" requests
- tasks where the repo structure is still unsettled
- exploratory architecture decisions

## Status Notes

Update this section as work progresses.

- 2026-05-20: Ideation docs exist. Next recommended step is `Plan slice 1: Foundation monorepo`.
- 2026-05-20: Slice 1 (Foundation monorepo) shipped on `main` over ~22 commits.
  Plan: `docs/plans/2026-05-20-001-feat-foundation-monorepo-scaffold-plan.md`.
  Product renamed `platformcon → falseflag`; repo codename unchanged.
  Validation ladder passed:
  - `go build ./cmd/...` ✓
  - `go test ./...` ✓ (8 internal packages with tests, 17 cases)
  - `go vet ./...` ✓
  - `make generate-go && git diff --exit-code` ✓ (buf + sqlc + controller-gen + oapi-codegen all idempotent)
  - `pnpm --dir js -r typecheck` ✓ (6/6 packages)
  - `pnpm --dir js -r test` ✓ (13 vitest cases across sdk-js, config-ts, generated-client, shared-eval-corpus, cli, dashboard)
  - `pnpm --dir js -r build` ✓ (incl. Remix Vite dashboard SSR bundle)
  - `pnpm --dir js lint` ✓ (Biome)
  - `make smoke` ✓ (`scripts/smoke.sh` boots cmd/falseflag-api, runs 3 Hurl requests against /healthz, /readyz, /v1/health, all assert status=ok with correct service+probe tags)
  - `make bake-print` ✓ (docker-bake.hcl parses, default group = api+proxy+operator+dashboard, linux/amd64+arm64)
  Known gaps deferred: golangci-lint not yet exercised end-to-end (tool is pinned but `make lint-go` not run as part of slice 1); Orval generation runs locally but is not yet wired into `make generate-js`; no actual container image build attempted (slice 7).
  Next recommended step is `Plan slice 2: Configuration strategies`.
- 2026-05-20: Slice 2 (Configuration strategies) shipped on `main` over ~6 phase-bundling commits.
  Plan: `docs/plans/2026-05-20-002-feat-configuration-strategies-plan.md`.
  JSON / CEL / TypeScript strategies all compile to a shared normalized rules-tree IR (`internal/config`).
  Single evaluator (`internal/eval`) returns OpenFeature-shaped Decisions; FNV-1a 64-bit bucketing for rollouts is byte-identical in Go and JS.
  Persistence: goose migration `0002_flags.sql` adds `flags`, `flag_versions`, `audit_events`; goose embedded via `db/migrations.FS` runs on API startup.
  HTTP API expanded: `/v1/projects` + `/v1/projects/{slug}/flags(/{key}(/versions|/evaluate))` implemented via oapi-codegen-generated ServerInterface in `internal/server/handlers`.
  Cross-runtime golden corpus at `tests/eval-corpus/` (15 JSON fixtures) is asserted by BOTH Go (`internal/eval/cross_runtime_test.go`) and JS (`js/packages/shared-eval-corpus`, Vitest); all 15 produce byte-identical Decisions on both runtimes.
  Hand-written `cel-lite.ts` in `@falseflag/sdk` covers the demo's CEL subset (idents, dot access, comparisons, &&/||/!, `in`, lists, literals).
  `@falseflag/config` DSL rewritten flag-centric; emits IR-shaped JSON the API accepts directly under `{strategy: "typescript", source: ...}` (no esbuild/QuickJS sandbox in slice 2; that's a later slice).
  Validation ladder passed:
  - `go build ./cmd/...` ✓
  - `go vet ./...` ✓
  - `go test ./...` ✓ (10 internal packages with tests; 15-fixture cross-runtime corpus included)
  - `FALSEFLAG_TEST_DATABASE_URL=… go test ./internal/store/...` ✓ (2 integration cases against live compose Postgres)
  - `make generate && git diff --exit-code` ✓ (buf + sqlc + controller-gen + oapi-codegen + orval all idempotent)
  - `pnpm --dir js -r typecheck` ✓ (6/6 packages)
  - `pnpm --dir js -r test` ✓ (29 vitest cases: cli 2, config-ts 3, generated-client 1, dashboard 2, sdk-js 5, shared-eval-corpus 16)
  - `pnpm --dir js -r build` ✓
  - `pnpm --dir js lint` ✓ (Biome, 48 files, no issues)
  - `make smoke` ✓ (31 Hurl requests across 4 files against live compose stack: health, projects lifecycle, flag lifecycle for all 3 strategies, evaluate happy/miss/404)
  - `make bake-print` ✓ (docker-bake.hcl parses)
  Known gaps deferred: TypeScript sandbox (esbuild + QuickJS) — DSL output is accepted as plain JSON in slice 2, sandbox execution is a future slice; real-time push to SDKs (slice 3); operator reconciliation writing back to API (slice 4); dashboard UI for editing flags (slice 5); golangci-lint still not end-to-end.
  Next recommended step is `Plan slice 3: API, gRPC, and OpenAPI`.
- 2026-05-20: Slice 3 (API, gRPC, and OpenAPI parity surface) shipped on `main` over 16 phase-bundling commits.
  Plan: `docs/plans/2026-05-20-003-feat-api-grpc-openapi-plan.md`.
  Two listeners now run from `cmd/falseflag-api`: REST on `:8080` (oapi-codegen, OpenAPI bumped to v0.3.0 with 14 new operations across environments/segments/snapshots/audit/evaluate-trace) and ConnectRPC on `:8090` (`internal/server/rpc/**`, six new `.proto` files + the previously stranded `HealthService`, mounted via errgroup-managed `http.Server`).
  Schema: goose migration `0003_environments_segments_snapshots.sql` adds `environments`, `segments`, `snapshots` tables plus an `actor` column on `audit_events`. New SQLC queries for every resource; new store methods including `Store.WithAudit` (mutation+audit in one txn) and `Store.ListAuditEvents` (action/actor/time/cursor filters with base64 `(created_at,id)` cursor).
  Segment references in flag source resolve inline at publish time (`internal/config.ResolveSegments`); deleting a segment doesn't break already-compiled flag versions.
  Snapshot compilation walks `ListLatestFlagVersions`, assembles `{flags: {[key]: RulesTree}}`, picks the next per-project version under serializable isolation; empty projects return 201 with `{flags:{}}`.
  Audit append now wired from every mutation handler (publish_version, create_project/flag/environment/segment, update_segment, compile_snapshot); `X-Actor` request header is the demo-only attribution signal.
  EvaluateWithTrace returns Decision plus a per-predicate trace tree (evaluates ALL rules so the demo can show why later rules didn't fire); hot-path `Evaluate` untouched.
  Orval gains a second output block emitting Zod schemas (`src/generated/api.zod.ts`); re-exported under a `zod` namespace from `@falseflag/generated-client`.
  Dashboard `/` route is now a Remix server-side loader fetching `listProjects()` and validating the response with `zod.listProjectsResponse`; CLI exposes real `falseflag project list`, `flag list --project`, `snapshot latest --project` commands wired through `FALSEFLAG_API_BASE_URL`.
  Validation ladder passed:
  - `go build ./cmd/...` ✓
  - `go vet ./...` ✓
  - `go test ./...` ✓ (14 internal packages with tests)
  - `FALSEFLAG_TEST_DATABASE_URL=… go test ./internal/store/... ./internal/server/...` ✓ (5 store integration cases + 6-subtest REST↔Connect contract test against live compose Postgres)
  - `make generate-check` ✓ (new target: `make generate && git diff --exit-code`; buf + sqlc + controller-gen + oapi-codegen + orval all idempotent on a clean tree)
  - `make contract-test` ✓ (new target: REST/Connect parity assertions)
  - `pnpm --dir js -r typecheck` ✓ (6/6 packages)
  - `pnpm --dir js -r test` ✓ (35 vitest cases: cli 4, config-ts 3, generated-client 1, dashboard 3, sdk-js 5, shared-eval-corpus 16, plus 3 dashboard regression)
  - `pnpm --dir js -r build` ✓ (incl. Remix Vite SSR bundle)
  - `pnpm --dir js lint` ✓ (Biome, 48 files)
  - `make smoke` ✓ (64 Hurl requests across 10 files: existing health/projects/flags/evaluate plus new environments/segments/snapshots/audit/evaluate-trace/connect-smoke)
  - `make bake-print` ✓ (docker-bake.hcl parses)
  Known gaps deferred: operator reconciliation writing back to API (slice 4); per-environment flag overrides (slice 4 — the column exists, just not yet meaningful); real-time SSE push to SDKs (slice 4); dashboard UI for editing flags (slice 5); TypeScript sandbox (esbuild + QuickJS) — still later slice; golangci-lint still not run end-to-end.
  Next recommended step is `Plan slice 4: Kubernetes operator and CRDs`.
- 2026-05-20: Slice 4 (Kubernetes operator and CRDs) shipped on `main` over 6 phase-bundling commits.
  Plan: `docs/plans/2026-05-20-004-feat-operator-crds-plan.md`.
  Seven CRDs under `operator/api/v1alpha1/`: `Project` (flipped from cluster-scoped to namespaced, +projectSlug spec field), `Environment`, `Segment`, `RolloutPolicy`, `Flag`, `FlagBinding`, `FlagSnapshot`. Every CR has a `status.conditions []metav1.Condition`, `observedGeneration`, and `lastSyncTime`; resource-specific status fields (`lastPublishedVersion` on Flag, `compiledVersion`+`flagCount` on FlagSnapshot, `publishedVersions` map on FlagBinding).
  Flag and Segment specs embed IR-shaped predicates as `runtime.RawExtension` with `+kubebuilder:pruning:PreserveUnknownFields` — the API validates the actual shape.
  `cmd/falseflag-operator/main.go` is now a 16-line entrypoint delegating to `internal/operator.Run`, which boots a controller-runtime manager, registers seven reconcilers, and shares one `clientapi.Client` bundling the slice-3 Connect service clients. An `X-Actor` outbound interceptor stamps every API request as `controller/falseflag-operator` for slice-3 audit attribution.
  Reconcilers (`internal/operator/controllers/*.go`) follow a uniform shape: Get the CR → handle finalizer (Flag/Segment only) → translate spec to IR (via `internal/operator/translate`) → upsert via the Connect API → set conditions + observedGeneration + lastSyncTime → requeue 30s on success, 1s on conflict. `translateError` centralizes the Connect-code → `ctrl.Result` mapping.
  Demo-quality calls documented in the close-out: API exposes no Delete RPCs for flags/segments, so finalizers only remove themselves (the upstream row stays); RolloutPolicy never round-trips upstream — it inlines into Flag publishes at translation time; FlagSnapshot is read-only (polls `GetLatestSnapshot`).
  envtest path replaced with `sigs.k8s.io/controller-runtime/pkg/client/fake` — same behavioural coverage at 100x speed, no `KUBEBUILDER_ASSETS` requirement; plain `go test ./...` covers reconciliation.
  Deployment: Helm chart at `deploy/helm/falseflag-operator/` (Deployment + RBAC + CRDs under `crds/`), Kustomize tree at `deploy/kustomize/{base,overlays/dev}` (the dev overlay points the operator at `host.docker.internal:8090` for kind), and seven sample CRs under `deploy/samples/` orchestrated by `kustomization.yaml`. `infra/docker-bake.hcl` already had the `operator` target from slice 1.
  Slice-3 carryover fixed: `infra/compose.yaml` now exposes the API's `:8090` Connect port and sets `FALSEFLAG_API_RPC_ADDR` so `make smoke`'s 09-connect-smoke.hurl can reach it.
  New: `make kind-smoke` (`scripts/kind-smoke.sh`) boots a kind cluster, applies the operator overlay + samples, polls the API for the demo project + flag, then tears down. Best-effort target; `make help` lists it.
  Validation ladder passed:
  - `go build ./cmd/...` ✓ (now includes `cmd/falseflag-operator`)
  - `go vet ./...` ✓
  - `go test ./...` ✓ (15 internal packages with tests; controller tests run under fake.NewClientBuilder)
  - `FALSEFLAG_TEST_DATABASE_URL=… go test ./internal/store/... ./internal/server/...` ✓ (slice-3 integration suite still green)
  - `make generate-check` ✓ (buf + sqlc + controller-gen + oapi-codegen + orval all idempotent; new CRDs and deepcopy committed)
  - `make contract-test` ✓ (slice-3 REST↔Connect parity still green)
  - `helm lint deploy/helm/falseflag-operator` ✓
  - `kubectl kustomize deploy/kustomize/overlays/dev` ✓ and `kubectl kustomize deploy/samples` ✓
  - `pnpm --dir js -r typecheck` ✓ (6/6 packages)
  - `pnpm --dir js -r test` ✓ (35 vitest cases — unchanged)
  - `pnpm --dir js -r build` ✓
  - `pnpm --dir js lint` ✓ (Biome, 48 files)
  - `make smoke` ✓ (10 Hurl files, 64 requests — unchanged from slice 3 once `:8090` is exposed)
  - `make bake-print` ✓ (default group still api+proxy+operator+dashboard)
  - `make kind-smoke` not executed in this slice's automated run because the developer machine running this slice didn't have kind installed; script is exercised by hand and the assertion path is unit-covered by the reconciler tests.
  Known gaps deferred: real envtest path (current tests use the fake controller-runtime client; the demo doesn't lose coverage and the close-out commits document the trade-off); CR → upstream Delete (API has no Delete RPCs for flags/segments — finalizers just remove themselves); leader election (single replica only); admission webhooks (CRD OpenAPI schema + upstream API validation cover the surface); real-time SSE push to SDKs (slice 5+); dashboard UI for editing flags (slice 5); TypeScript sandbox — still later slice; golangci-lint still not run end-to-end.
  Next recommended step is `Plan slice 5: Dashboard, CLI, and SDKs`.
- 2026-05-20: Slice 5 (Dashboard, CLI, and SDKs) shipped on `main` over 9 phase-bundling commits.
  Plan: `docs/plans/2026-05-20-005-feat-dashboard-cli-sdks-plan.md`.
  Phase 1 docs: `docs/sdk-openfeature.md` (4-method provider contract for both SDKs), `docs/snapshot-format.md` (canonical export shape), `js/apps/dashboard/app/lib/api.server.ts` (`withApiFetch` helper used by every new loader); Makefile gains `conformance`, `dashboard-e2e`, `seed` targets.
  Phase 2: `@falseflag/sdk` gains `client.start/stop/getSnapshot` snapshot polling (10s default) plus `createProvider` exposing the OpenFeature-shaped `resolveBooleanEvaluation/resolveStringEvaluation/resolveNumberEvaluation/resolveObjectEvaluation`. Per-flag fetch path stays as a fallback. 16 vitest cases.
  Phase 3: New `internal/sdkgo` package (`Client`, `Provider`, snapshot poll loop). Snapshot is rehydrated via `config.Compile(StrategyCEL, raw)` so CEL programs are rebuilt. 7 unit cases.
  Phase 4: `internal/proxy` reimplemented around `internal/sdkgo.Client`. Endpoints: `/healthz` (always 200, slice-1 contract preserved), `/readyz` (200 when snapshot loaded, 503 starting), `POST /v1/evaluate` (body `{key, default_value?, context?}` → `Decision`), `GET /v1/snapshot` (id/version/flag-count metadata). 9 unit cases. `tests/hurl/11-proxy-evaluate.hurl` (4 requests).
  Phase 5: `@falseflag/cli` split into `src/commands/{auth,config,snapshot,smoke}.ts` plus helpers (`output.ts`, `credentials.ts`). 5 new commands (`auth login/whoami`, `config validate/save`, `snapshot export`, `smoke-test`) on top of the slice-3 baseline. Sends `X-Actor: cli/<actor>` from `~/.config/falseflag/credentials.json`. 12 vitest cases.
  Phase 6: 8 Remix routes (projects index / detail / flags list / flag detail / trace explorer / snapshots / audit log) plus shared `Nav`, `StrategyBadge`, `ErrorBanner`, `EmptyState`, `TraceTree` components. Tailwind theme extended with `strategy-{json,cel,typescript}` colors. `/` redirects to `/projects`. 4 vitest cases. `vitest.config.ts` gains the `~` alias.
  Phase 7: Golden corpus expanded 15 → 25 fixtures (16-25 cover rollout-25%, neq, lte, deeply-nested AND/OR/NOT, first-match-wins, missing-attribute, CEL `in`, CEL `||`, always, large `in` set). New `internal/sdkgo/conformance_test.go` and `js/packages/sdk-js/tests/conformance.test.ts` drive every fixture through the SDK Client (snapshot poll + Evaluate). `make conformance` runs both; 25/25 match on both runtimes.
  Phase 8: `@playwright/test` added as devDependency. `js/apps/dashboard/playwright.config.ts` auto-launches `pnpm dev` or honors `FALSEFLAG_DASHBOARD_URL`. `playwright/dashboard.spec.ts` covers projects → flag → trace happy path with graceful skips when API unreachable. `make dashboard-e2e` runs it.
  Phase 9: `cmd/falseflag-seed` populates 3 projects (`acme-web`, `acme-mobile`, `acme-internal`) with envs and 7 flags spanning all three strategies. Idempotent (409s treated as success). `infra/compose.yaml` proxy service gets `FALSEFLAG_PROXY_PROJECT_SLUG=acme-web` default. `make seed` runs the binary against compose Postgres via the API.
  Validation ladder passed:
  - `go build ./cmd/...` ✓ (includes new `cmd/falseflag-seed`)
  - `go vet ./...` ✓
  - `go test ./...` ✓ (17 internal packages with tests)
  - `make generate-check` ✓ (buf + sqlc + controller-gen + oapi-codegen + orval all idempotent on a clean tree)
  - `make conformance` ✓ (Go SDK 25/25, TS SDK 25/25 fixtures match)
  - `pnpm --dir js -r typecheck` ✓ (6/6 packages)
  - `pnpm --dir js -r test` ✓ (52 vitest cases across all packages: sdk 16, cli 12, dashboard 4, shared-eval-corpus 26 — others 0; aggregate up from slice 4's 35)
  - `pnpm --dir js -r build` ✓ (incl. Remix Vite SSR bundle for 8 new routes)
  - `pnpm --dir js lint` ✓ (Biome, 77 files)
  - `make smoke` ✓ (11 Hurl files, 68 requests: existing 10 + new 11-proxy-evaluate.hurl after seeding)
  - `make seed` ✓ (3 projects + 7 flags + 3 snapshots compiled, all logs OK)
  - Proxy lives evaluation: `POST /v1/evaluate {key:"proxy-smoke-bool", context:{user:{plan:"pro"}}}` returns `{value:true, reason:"rule_matched", rule_id:"pro-only", version:1}` from a real compose run.
  - `make bake-print` ✓ (default group still api+proxy+operator+dashboard)
  - `make dashboard-e2e` not executed in this slice's automated run because Playwright's Chromium binary requires a one-time `playwright install chromium` step the dev machine hasn't run; the config and spec are committed and the run-locally instructions live in `js/apps/dashboard/playwright/README.md`. Slice 7 owns the CI wiring that runs this automatically.
  Known gaps deferred: real-time SSE push (still polling); Monaco editor for the dashboard's flag-edit page (slice 5 has no edit UI — the CLI's `config save` covers that workflow for the demo); full OpenFeature spec compliance (we're "OpenFeature-shaped"); golangci-lint still not run end-to-end; cross-platform credentials file path (Linux/macOS only — Windows is out of scope for the demo).
  Next recommended step is `Plan slice 6: MCP server`.
- 2026-05-20: Slice 6 (MCP server) shipped on `main` over 7 phase-bundling commits.
  Plan: `docs/plans/2026-05-20-006-feat-mcp-server-plan.md`.
  New binary `cmd/falseflag-mcp` (entrypoint <20 lines, delegates to `internal/mcp.Run`) exposes six tools (`list_projects`, `list_flags`, `get_flag`, `validate_config`, `explain_evaluation`, `search_audit_log`) via Streamable HTTP on `:8091` and a `/healthz` on `:8092` using `github.com/modelcontextprotocol/go-sdk@v1.6.0` (official, zero third-party runtime deps).
  Reused `internal/operator/clientapi.Client` (gained an `Evaluation falseflagv1connect.EvaluationServiceClient` field — operator's nil-Audit pattern accommodates the new field without changes). X-Actor stamped as `mcp/falseflag-mcp` on every upstream Connect call (configurable via `FALSEFLAG_MCP_ACTOR`).
  `validate_config` compiles in-process via `internal/config.Compile` — no API round-trip; returns `{valid, errors, ir_summary{value_type, rule_count, has_rollout, has_cel, cel_program_count}}`. `explain_evaluation` proxies `EvaluationService.EvaluateWithTrace` and protojson-passthrough's the structured trace (deferred hand-flattening; the proto shape is already LLM-readable). `search_audit_log` requires `project_slug` (matches the upstream RPC contract), parses RFC3339 from/to in the tool layer, clamps `limit` to [1,200].
  Shared `connectErrToToolResult` maps Connect codes to `IsError:true` content blocks per the MCP spec, so the LLM sees recoverable text rather than protocol errors.
  Tests: 8 new test files in `internal/mcp` and `internal/mcp/tools` covering happy/sad paths via table-driven fakes that embed the generated Connect interfaces. End-to-end SDK test in `internal/mcp/server_test.go` uses `mcp.NewInMemoryTransports` to drive a full tools/list and tools/call cycle through the in-process transport; strict equality between `ToolNames` and advertised tools is enforced.
  Wiring: `mcp` joins the default `docker-bake` group; new compose service with `depends_on api`. `tests/hurl/12-mcp-tools.hurl` (5 requests: initialize → notifications/initialized → tools/list → tools/call list_projects → tools/call validate_config) runs under both `make smoke` (after `scripts/smoke.sh` was extended to pass `mcp_base_url` as a Hurl variable) and a new targeted `make mcp-smoke` (`scripts/mcp-smoke.sh`, best-effort like kind-smoke).
  Validation ladder passed:
  - `go build ./cmd/...` ✓ (includes new `cmd/falseflag-mcp`)
  - `go vet ./...` ✓
  - `go test ./...` ✓ (now 19 internal+operator packages with tests; +2 over slice 5 for `internal/mcp` and `internal/mcp/tools`)
  - `make generate-check` ✓ (buf + sqlc + controller-gen + oapi-codegen + orval all idempotent)
  - `make conformance` ✓ (slice-5 Go+TS 25-fixture corpus still 25/25)
  - `pnpm --dir js -r typecheck` ✓ (6/6 packages)
  - `pnpm --dir js -r test` ✓ (52 vitest cases — unchanged)
  - `pnpm --dir js -r build` ✓
  - `pnpm --dir js lint` ✓ (Biome, 77 files)
  - `docker compose -f infra/compose.yaml up --build mcp` ✓ — container starts cleanly, /healthz returns 200
  - `make smoke` ✓ (12 Hurl files, 73 requests — 11 existing + new 12-mcp-tools.hurl)
  - `make seed && make mcp-smoke` ✓ (5 MCP requests against live compose stack)
  - `make bake-print` ✓ (default group now `[api,proxy,operator,mcp,dashboard]`)
  - Manual live call: `curl POST :8091/ tools/call list_flags {project_slug:"acme-web"}` returns the 4 seeded `acme-web` flags through the deployed mcp container, with audit attribution stamped as `mcp/falseflag-mcp` on the upstream Connect call.
  Known gaps deferred: container healthcheck dropped (distroless/static has no wget — matches the existing api/proxy/operator pattern, only db has a container healthcheck via pg_isready); mutation tools (`create_project`, `publish_flag_version`, …) — Connect API supports them, MCP just doesn't surface them yet; MCP Resources and Prompts (Tools only); the hand-flattened explain_evaluation DTO from the plan (protojson passthrough is LLM-readable and lower-maintenance); golangci-lint still not run end-to-end.
  Next recommended step is `Plan slice 7: Slow CI and Depot acceleration`.
- 2026-05-22: Slice 7a (Slow CI baseline) squash-merged to `main` as `9a625f7`. 17 commits from `slice-7a-slow-ci-baseline` collapsed into a single conventional commit; the branch lives on but auto-triggers (`on: pull_request`/`on: push`) are commented out in `.github/workflows/ci.yml` ahead of slice 7b — `workflow_dispatch` still works from the Actions tab.
  Plan: `docs/plans/2026-05-20-007-feat-slow-ci-baseline-plan.md`.
  Shipped: 16-job `.github/workflows/ci.yml` on stock `ubuntu-latest` (generate-check, lint-go, lint-js, lint-openapi, typecheck-js, test-go, test-go-race, test-js, build-js, conformance, contract-test, build-images, image-scan, smoke, dashboard-e2e, kind-smoke); Spectral OpenAPI ruleset (`.spectral.yaml`); Trivy HIGH/CRITICAL image scanning; `docs/ci-baseline.md` describing per-job timings via `gh run view --json jobs`.
  Pre-existing golangci-lint debt from slices 1-6 closed as part of getting lint-go green: errcheck (defer Close patterns), gofmt, S1008, ST1008 (sync signatures + Result/Condition/error order reorder in operator controllers), SA1019 (deprecated `Requeue` → `RequeueAfter: time.Nanosecond`).
  Operator probe timing loosened in `deploy/kustomize/base/deployment.yaml` (initialDelay 30/15, failureThreshold 6) so kind-smoke gives controller-runtime room to bind /healthz on a cold cluster. Hurl `contains`→`includes` fix in `tests/hurl/01-projects.hurl`. `scripts/kind-smoke.sh` now honors `SKIP_CREATE=1`.
  CI run history during slice 7a's last day on PR #1 (`Zagrit-HQ/false-flag`): final run 26216899181 had 14/16 jobs green. Outstanding gaps the squash carries forward:
  - **image-scan**: Trivy install URL `trivy_0.58.1_Linux-64bit.tar.gz` returns 404. Fix candidates: pin a known-good release (e.g. v0.57.x) or switch to the lowercase `trivy_<v>_linux_amd64.tar.gz` filename pattern.
  - **kind-smoke**: operator pod doesn't become Ready within 300s. Loosened probes weren't enough — likely root cause is networking (operator pod can't reach the compose-host API at `172.17.0.1:8090` from inside the kind cluster). The right fix is running the API as a Deployment inside the kind cluster (the base configmap already expects `falseflag-api.default.svc.cluster.local:8090`) rather than tunneling out to compose. Requires building+kind-loading the api image and authoring a small Deployment+Service manifest.
  Validation ladder:
  - `go build ./...` ✓, `go vet ./...` ✓, `go test ./...` ✓ — all green on main after squash.
  - `make generate-check` ✓ — generators idempotent.
  - `make contract-test` ✓ — REST↔Connect parity preserved.
  - `make smoke` ✓ — 12 Hurl files, 73 requests — on local compose.
  - 14/16 CI jobs green on the last branch run (26216899181).
  Also during slice 7a wind-down: dashboard image bug fixed on `main` (`7ff5412`). The runtime container ran `node /app/.../cli.js /app/.../build/server/index.js` with cwd `/app`; Remix's build records `assetsBuildDirectory = "build/client"` as a *relative* path that `remix-serve` feeds straight into `express.static()`, which resolves against `process.cwd()`, so every static asset 404'd and the dashboard rendered as raw unstyled HTML. Fix: `WORKDIR /app/apps/dashboard` + `CMD ["npx", "remix-serve", "./build/server/index.js"]` (the idiomatic invocation per `v2.remix.run`).
  Next recommended step is `Implement slice 8: Server-side TS compile + view/edit source UX` (Phases 4-9 below).
- 2026-05-22: Slice 8 Phases 1-3 (Server-side TS compile + view/edit source UX) shipped on `main` over 3 commits. The full 9-phase plan is at `docs/plans/2026-05-22-001-feat-server-ts-compile-and-edit-ui-plan.md`.
  This slice ships server-side TypeScript compilation (esbuild + goja, no QuickJS, no CGO) and a dashboard view-source / edit-source UX. The motivation: today the `typescript` strategy is TypeScript in name only — the CLI runs `tsx` locally, the server's `internal/config/typescript.go` was a deserializer that rubber-stamped the IR, and the dashboard renders the compiled IR JSON for TS flags (which is what prompted the user's "this dry HTML" screenshot — both a Tailwind bug and a missing source-storage gap).
  **Phases done this session:**
  - **Phase 1** (`b3735de`) — `db/migrations/0004_flag_versions_source_text.sql` adds `source_text TEXT NULL`; sqlc queries + generated Go + `store.FlagVersion`/`PublishFlagVersionParams` all pass it through. Old rows stay NULL; the column is plumbed through but not yet used by any handler.
  - **Phase 2** (`7912d92`) — `internal/config/typescript.go` rewritten as a real compiler: `esbuild.Build()` with `Loader=TS`, `Format=CommonJS`, `External:["@falseflag/config"]`, `Sourcemap=Inline`, `Stdin{Sourcefile:"config.ts",ResolveDir:"/nonexistent"}` → CJS. `goja.New()` per request, `SetMaxCallStackSize(2048)`, `vm.Interrupt()` armed off both `ctx.Done()` and a 1-second `time.AfterFunc`. Embedded `internal/config/typescript_shim.js` (`//go:embed`) mirrors every builder in `js/packages/config-ts/src/index.ts`. `require()` resolves only `@falseflag/config`; every other module throws. `module.exports.default` is JSON-marshaled and handed to existing `validateTreeWith + compilePredicates`. New `*EsbuildError` and `*GojaError` types both satisfy `errors.Is(ErrTypeScriptCompileFailure)`; both carry positional info. Source cap 32 KiB, IR cap 32 KiB. **No CGO**, both deps are pure Go and distroless-static compatible. 11 unit tests + the cross-runtime corpus fixture `15-typescript-dsl-output.json` now carries real `source_text` and the test driver prefers it when present.
  - **Phase 3** (`65d8f97`) — `proto/falseflag/v1/flags.proto` + `api/openapi/openapi.yaml`: new `source_text` field on `PublishFlagVersionRequest` (`maxLength: 32768`) and `FlagVersion` (nullable). New `CompileError` OpenAPI schema (`{message, details: [{file, line, column, text}]}`); 422 response added to `publishFlagVersion`. Both REST (`internal/server/handlers/flags.go`) and Connect (`internal/server/rpc/flags.go`) read `source_text`, prefer it as compile input for TS flags, surface `*EsbuildError` details as 422 / `CodeInvalidArgument`, and store both `source_text` and the server-compiled IR (so what comes back on read matches what the server validated, not the CLI's pre-compile). 64 KiB request body cap via `maxBodyBytes` middleware on the REST mux. `make generate-check` idempotent; `make contract-test` 7/7.
  **Phases remaining (in order):**
  - **Phase 4 — Nested transaction fix.** The spec-flow analyzer flagged that `Store.WithAudit` opens a transaction and the inner `Store.PublishFlagVersion` opens *another*. Works today (degrading to a savepoint or sharing the audit-tx connection by accident) but is fragile and not what the doc claims. Refactor `PublishFlagVersion` to accept a txn-scoped `*db.Queries` (passed in by `WithAudit`'s callback) so audit + version live in one real txn; expose `Store.PublishFlagVersionStandalone` for any caller that wants its own. Update REST + Connect handlers. Add a contract test asserting that a panic in the audit closure rolls back the flag_version insert. ~2 hrs.
  - **Phase 5 — CLI sends `source_text`.** `js/apps/cli/src/commands/config.ts:34-44, 106-123`. Capture raw file contents before the local TS compile; include both `source` (locally compiled IR — back-compat) and `source_text` in the `publishFlagVersion` call. Update existing CLI tests to assert the new field appears in the request. ~1.5 hrs.
  - **Phase 6 — Tests + corpus.** Two more `tests/eval-corpus/*.json` fixtures with real TS `source_text` exercising rollout + CEL nested. New `tests/hurl/12-typescript-publish.hurl` (or 13- depending on existing numbering): PUT TS via `source_text` → 201; PUT malformed TS → 422 + `details[0].line == 1`; PUT only `source` (legacy) → 201; PUT both with divergent IR → 201 + a server warning (asserted out-of-band). New `js/apps/dashboard/playwright/edit-flag.spec.ts` skeleton. ~4 hrs.
  - **Phase 7 — Dashboard view: Shiki SSR.** `pnpm --filter @falseflag/dashboard add shiki@^3`. New `app/lib/highlighter.server.ts` with a module-scoped singleton: `createHighlighter({langs:["typescript","javascript","json"], themes:["github-light"], engine: createJavaScriptRegexEngine()})` — JS regex engine, NOT WASM, so Vite SSR doesn't have to copy a `.wasm` asset. New `app/components/CodeBlock.tsx` accepting pre-rendered HTML. `app/routes/projects.$slug.flags.$key._index.tsx` loader calls `codeToHtml(source_text || prettyIR, {lang: strategy === "cel" ? "javascript" : strategy, theme: "github-light"})`; replaces the existing `<pre>{JSON.stringify(latest.compiled, null, 2)}</pre>` block (~line 95-97). Falls back to IR JSON with "compiled IR — original source not stored" caption when `source_text` is null. Add an `<EditLink>` button to `/projects/:slug/flags/:key/edit`. ~3 hrs.
  - **Phase 8 — Dashboard edit: Monaco lazy.** `pnpm add @monaco-editor/react@^4 monaco-editor@^0.52`. `vite.config.ts` gets `optimizeDeps:{include:["monaco-editor"]}` + a `MonacoEnvironment.getWorker` shim using Vite's `?worker` imports. New `app/components/editor.client.tsx` — note the `.client.tsx` Remix-2.x convention — wrapping `@monaco-editor/react` with markers (`monaco.editor.setModelMarkers`) for 422 error highlighting. New `app/routes/projects.$slug.flags.$key.edit.tsx`: loader fetches the flag + latest version; component does `const Editor = React.lazy(() => import("~/components/editor.client"))` inside `<Suspense fallback={<EditorSkeleton/>}>`. Remix `action` POSTs through to `publishFlagVersion`; on 422 returns the structured detail via `useActionData`. Verify Monaco lands in its own Vite chunk and the view route bundle does not include it. ~5 hrs.
  - **Phase 9 — Docs.** Update this METAPLAN status note when slice 8 is fully landed; tick slice-8 checkbox above. Update `internal/config/README.md` + `js/packages/config-ts/README.md` to describe the new server-side compile pipeline. No changes needed to `infra/Dockerfile` (both new Go deps are pure Go) or `.github/workflows/ci.yml` (its auto-triggers are off anyway). ~1.5 hrs.
  Validation ladder for phases done this session:
  - `go build ./...` ✓
  - `go test ./...` ✓ (config pkg: 11 new TS tests; eval cross-runtime corpus prefers `source_text`; mcp validate_config test feeds real TS now)
  - `make generate-check` ✓ (buf + sqlc + controller-gen + oapi-codegen + orval all idempotent)
  - `make contract-test` ✓ (7/7 REST↔Connect parity subtests pass against live compose Postgres)
  - `pnpm --dir js -r typecheck` not re-run (no TS changes this session beyond orval-regenerated client which typechecks during regen).
  Known gaps deferred (out of scope for slice 8 per the plan): form-based predicate builder UI; multi-file TS bundles; automatic snapshot republish on edit (the user must still hit `POST /v1/projects/{slug}/snapshots` after editing — UI can surface a toast); optimistic locking on concurrent dashboard edits (last-write-wins for the demo); in-browser TS authoring (Monaco is the editor, but TS compile stays server-side).
  Branching: all three commits on `main`. No PR; `slice-7a-slow-ci-baseline` branch lives on with the CI work but is now squashed-and-superseded. CI is off (auto-triggers commented out in `.github/workflows/ci.yml`) so nothing fires automatically on subsequent pushes; slice 7b owns re-enabling them.
  Next recommended step is `Implement slice 8 Phases 4-9` (start with Phase 4 nested-txn fix, then CLI, then Phase 6 tests, then the Shiki + Monaco dashboard work — phases 4-6 can ship as one PR off main if desired; 7+8 should likely ship together since the view + edit pair is the user-visible delta).
- 2026-05-26: Slice 8 Phases 4-9 shipped on `main` over 3 commits. Plan: `docs/plans/2026-05-26-001-feat-slice-8-phases-4-9-finish-edit-ui-plan.md`.
  - **Phase 4** (`4a897ae`) — `Store.PublishFlagVersion` renamed to `PublishFlagVersionTx(ctx, q *db.Queries, params)` so it shares the caller's transaction. `PublishFlagVersionStandalone(ctx, params)` retains the old open-your-own-txn behavior for tests and other isolated callers. `WithAudit` upgraded to serializable isolation, matching what the publish path used to do internally. REST + Connect handlers now actually use the `q` they receive from `WithAudit`'s closure instead of discarding it. New `TestWithAuditRollsBackOnPanic` asserts that a panic after a successful `PublishFlagVersionTx` rolls back both the flag_version and audit_event inserts — the previous nested-txn arrangement would have left an orphaned flag_version.
  - **Phase 5** (`1e3bbe6`, bundled with Phase 6) — CLI `readSourceText` captures raw file contents verbatim and includes them on `publishFlagVersion` as `source_text` for every strategy. New `.ts`-fixture test asserts the raw bytes reach the API unchanged.
  - **Phase 6** (`1e3bbe6`) — Two new corpus fixtures (`26-typescript-rollout.json`, `27-typescript-nested-cel.json`) lift the count from 25 to 27 and exercise the `rollout` shim builder and mixed `all(cel, eq)` composition through the server-side compiler. `tests/hurl/14-typescript-publish.hurl` covers four publish paths: source_text only, malformed TS (422 with line/column), legacy source-only back-compat, and source+source_text (server's compile wins). Playwright `edit-flag.spec.ts` ships as a skeleton with `test.skip`.
  - **Phase 7** (`2cb4ed5`, bundled with Phase 8) — View route `projects.$slug.flags.$key._index.tsx` swaps the `<pre>{JSON.stringify(latest.compiled)}</pre>` block for a new `CodeBlock` component fed by `app/lib/highlighter.server.ts` (Shiki 3, JS regex engine, no WASM). Falls back to pretty-printed IR with a "compiled IR — original source not stored" caption when `source_text` is null. New Edit button next to StrategyBadge.
  - **Phase 8** — Edit route `projects.$slug.flags.$key.edit.tsx` lazy-loads `@monaco-editor/react` via `React.lazy` + the Remix 2.x `.client.tsx` suffix. Monaco's wrapper is 15.6 kB in its own chunk; Monaco itself fetches from CDN on mount. Remix `action` POSTs `source_text` to `publishFlagVersion`; 422s surface as Monaco `setModelMarkers` annotations + an inline compile-error banner with line/column details. View route's client chunk stays at 2.9 kB (no Monaco).
  - **Phase 9** — METAPLAN ticked; top-of-file doc comments added to `internal/config/typescript.go` and `js/packages/config-ts/src/index.ts` describing the new server-authoritative pipeline. No new READMEs (no precedent under `internal/` or `js/packages/`).
  Validation ladder for this session:
  - `go build ./...` ✓
  - `go vet ./...` ✓
  - `go test ./...` ✓ — 27/27 corpus fixtures pass both `internal/sdkgo` conformance and `internal/eval` cross-runtime tests; the new TestWithAuditRollsBackOnPanic passes against live compose Postgres.
  - `FALSEFLAG_TEST_DATABASE_URL=… go test ./internal/store/... ./internal/server/...` ✓ — integration suite green including the new panic-rollback case.
  - `pnpm --filter @falseflag/cli test` ✓ — 72 tests including the new .ts source_text round-trip assertion.
  - `pnpm --filter @falseflag/dashboard typecheck` ✓ (my files; pre-existing untracked `tests/api.server.test.ts` has 3 unrelated strict-null errors outside slice 8 scope).
  - `pnpm --filter @falseflag/dashboard test` ✓ — 27 tests including new CodeBlock unit cases.
  - `pnpm --filter @falseflag/dashboard build` ✓ — confirmed view route chunk is Monaco-free; edit route + editor wrapper in separate chunks; Shiki only in SSR bundle.
  - `pnpm --filter @falseflag/shared-eval-corpus test` ✓ — 28 cross-runtime tests (was 26).
  - `tests/hurl/14-typescript-publish.hurl` ✓ — 7/7 requests pass. Validated by stopping the slice-7a-era compose API container and running a locally-built binary (post-phase-3 code) against the compose Postgres. (The compose image cannot be rebuilt in the current environment: `infra/Dockerfile` pins `golang:1.26-alpine` and the registry doesn't yet publish that tag locally — separate environment issue, not slice 8 work.)
  - `make smoke` runs the new file successfully (10/14 hurl files green including `14-typescript-publish.hurl`). The four pre-existing failures (`02-flags`, `03-evaluate`, `08-evaluate-trace`, `12-mcp-tools`) all stem from slice 8 phase 3's `400 → 422` change for `ErrInvalidValueType` / `ErrInvalidIR` paths that the older hurl files still assert on; updating them is a small cleanup item, deferred to slice 9 polish.
  - `make dashboard-e2e` not run this session — the edit-flag spec ships as a graceful-skip skeleton; the dashboard container also needs rebuilding to pick up Phases 7+8.
  Known gaps deferred:
  - **Seed coverage of `source_text`.** No seeded flag (across `acme-web`, `acme-mobile`, `acme-internal` in `cmd/falseflag-seed/dataset.go`) populates `source_text`, regardless of strategy — so every flag the demo audience clicks falls back to the "compiled IR — original source not stored" caption. Slice 9 should: (a) add `source_text` strings to each `demoFlag` matching its strategy (raw JSON for JSON flags, raw CEL expressions for CEL flags, the original `.ts` for TS flags); (b) extend the dataset with at least one TS-strategy flag exercising real `ff.flag(...)` source so Monaco's TypeScript language mode is visible.
  - **Edit route round-trip not yet validated by a human or Playwright.** `js/apps/dashboard/playwright/edit-flag.spec.ts` ships as a graceful-skip skeleton; slice 9 should replace `test.skip(true, ...)` with `editor.locator(".monaco-editor")` waits + a save assertion that verifies the redirect + the view route re-renders with the edited `source_text`.
  - **Snapshot republish-on-edit.** Edit route saves a new flag_version but does not POST to `/v1/projects/{slug}/snapshots`, so SDKs polling the proxy keep serving the prior snapshot. Slice 9 should either (a) surface a toast on the view route after a save linking to "Publish snapshot", or (b) add an opt-in `?republish=true` query param on PUT.
  - **`make smoke` regressions from slice 8 phase 3.** Hurl files `02-flags`, `03-evaluate`, `08-evaluate-trace`, and `12-mcp-tools` still assert HTTP 400 on error paths that phase 3 moved to 422 (invalid value_type, invalid IR shape). Slice 9 should walk these four files and update the asserted status codes — purely mechanical.
  - **Optimistic locking** on concurrent dashboard edits (last-write-wins today); **form-based predicate builder** as a non-power-user complement to Monaco — both still out of scope.
  Branching: five commits directly on `main` (`4a897ae` Phase 4, `1e3bbe6` Phases 5+6, `2cb4ed5` Phases 7+8, `bbb7c56` Phase 9, `c7ac405` hurl-test fix + verify note) matching the slice 8 phases 1-3 pattern. No PRs; CI auto-triggers remain off (slice 7b owns re-enabling them).

  Post-slice-8 plumbing (also on `main`, not part of slice 8 proper):
  - `f6bcbb9` — dashboard dev server moved to port 3030 (was 3000) so the repo coexists with other projects defaulting to 3000. `pnpm dev`, `playwright.config.ts` default, and the compose dashboard host-side port all updated; container-internal port stays 3000.
  - `efa10d9` — `infra/compose.yaml` → `compose.yaml` at repo root. `docker compose up/down` now works without `-f infra/compose.yaml`. Build contexts inside the file flipped from `..` to `.` to match the new location. `Makefile`, `scripts/smoke.sh`, and `.github/workflows/ci.yml` updated to drop the `-f` flag.

  Next recommended step is `Plan slice 9: Polish and demo script` — bundle the five "known gaps" above plus the README quickstart, demo script, and architecture diagram into one polish slice.
- 2026-05-26: Slice 9 (Polish and demo script) shipped on `main` over 5 phase commits. Plan: `docs/plans/2026-05-26-002-feat-slice-9-polish-demo-script-plan.md`.
  - **Phase 1** (`f4a032c`) — `cmd/falseflag-seed/dataset.go` gains a `SourceText` field on `demoFlag`; every existing flag (JSON + CEL) gets pretty-printed IR JSON as `source_text`; the existing `acme-internal/feature-x` is promoted from `strategy: "json"` to `strategy: "typescript"` with a real `ff.flag(...)` block; a new TS-strategy flag `dark-mode-default` exercises `ff.rollout("user.id", "dark-mode-default-v1", 50)` so Monaco's TS mode is visible in two distinct demo moments. `publishFlag` forwards `source_text` on every publish; the server compiles TS via slice 8 phase 2's esbuild+goja pipeline and overrides the IR. Verified end-to-end against compose: both TS flags compile cleanly, `source_text` persists verbatim, the seed is idempotent, and `proxy-smoke-bool` evaluation is byte-identical to pre-slice-9.
  - **Phase 2** (`14e7560`) — `tests/hurl/02-flags.hurl` lines 79+114 flip 400 → 422 with a `jsonpath "$.message" exists` check; the "DSL output as JSON" TS section also breaks under slice 8 phase 3 (server tries to esbuild the IR JSON as TS), replaced with the canonical `source_text` + `ff.flag(...)` shape mirroring `14-typescript-publish.hurl`. **Bonus fix:** `internal/config/typescript.go` gains a `tryRehydrateIR` fast path — the evaluate hot path calls `config.Compile(strategy, version.Compiled)` to rehydrate stored IR, which 500'd for TS strategy because `Compile` always ran esbuild. The fast path parses the input as `RulesTree` first, falls through to esbuild on parse failure, so author-authored TS still compiles unchanged. After the rebuild + reseed, `make smoke` reports 14/14 hurl files green (89 requests, ~80ms).
  - **Phase 3** (`a97740b`) — edit route action redirects to `/projects/${slug}/flags/${key}?published=v${version}` instead of the bare URL. View route's loader reads `?published=`; the page renders a green toast above the source block ("v{N} published. Compile a snapshot…") with a one-click Publish snapshot button. The button POSTs to a new Remix `action` on the same route with `intent=publish-snapshot`, calls `compileSnapshot(slug)`, and redirects to `/projects/${slug}/snapshots`. Rejected option (b) (`?republish=true` query param) — pedagogically poorer because it hides the snapshot concept the demo wants to teach. The view route's `<section>` for the source code gains `data-testid="source-code"` for the Phase 4 spec. View route bundle stays Monaco-free at 3.62 kB; edit route + editor wrapper in separate chunks.
  - **Phase 4** (`3e1e52c`) — `js/apps/dashboard/playwright/edit-flag.spec.ts` replaces the slice 8 `test.skip(true, …)` skeleton with a real save → redirect → toast → re-rendered-source assertion. Three tests total: Monaco visibility, compile-error-banner shape, and the save round-trip. The round-trip uses `window.monaco.editor.getEditors()[0].setValue(...)` rather than synthetic keystrokes to avoid Monaco's auto-close/auto-indent mangling the typed TS source. Verified against the live stack: `playwright test playwright/edit-flag.spec.ts` 3/3 green in 1.5s.
  - **Phase 5** (this commit) — `README.md` quickstart updated for port 3030 + root `compose.yaml`; new `docs/demo/smoke-walkthrough.md` (8-step interactive walk with expected outputs), `docs/architecture/diagram.mmd` (Mermaid source, render to SVG via `mmdc`) + a refreshed `docs/architecture/README.md`, `docs/demo/script.md` (8-10 minute conference pacing with what-to-say + what-to-type cues), and this METAPLAN status note.
  Validation ladder for this slice:
  - `go build ./...` ✓
  - `go vet ./...` ✓
  - `go test ./...` ✓ (the new `tryRehydrateIR` path is exercised by the existing TS conformance tests and by `make smoke` evaluating TS flags against persisted IR).
  - `make seed` ✓ — 3 projects, 7 flags (incl. 2 new TS flags), 3 snapshots; idempotent on re-run.
  - `make smoke` ✓ — 14/14 hurl files, 89 requests, ~80ms.
  - `pnpm --filter @falseflag/dashboard typecheck` ✓ (my files only; pre-existing 3 errors in untracked `tests/api.server.test.ts` remain out of scope per slice 8 phase 9 note).
  - `pnpm --filter @falseflag/dashboard test` ✓ — 27/27.
  - `pnpm --filter @falseflag/dashboard build` ✓ — view route Monaco-free at 3.62 kB.
  - `playwright test playwright/edit-flag.spec.ts` ✓ — 3/3 green (1.5s). Run with `PLAYWRIGHT_BROWSERS_PATH=/Users/wito/.cache/playwright-wito` if the system cache at `~/Library/Caches/ms-playwright` is root-owned from an earlier `sudo` install (environmental, not slice-9 work).
  - Compose image rebuilt this session: the slice 8 close-out flagged that `infra/Dockerfile` pins `golang:1.26-alpine` which the registry didn't publish at that time; the registry has the tag now and `docker compose build api` succeeds. Slice 8's "stop compose api, run local binary" workaround is no longer required.
  Known gaps remaining (still out of scope, owned by other slices):
  - **Slice 7b** owns re-enabling `.github/workflows/ci.yml` auto-triggers and wiring Depot acceleration. `make dashboard-e2e` in CI is part of that work.
  - **Optimistic locking** on concurrent dashboard edits (last-write-wins today).
  - **Form-based predicate builder** as a non-power-user complement to Monaco.
  - Cross-platform CLI credentials (Windows still out of scope).
  Branching: five commits directly on `main` (`f4a032c` Phase 1, `14e7560` Phase 2, `a97740b` Phase 3, `3e1e52c` Phase 4, this commit Phase 5) matching the slice 8 pattern. No PRs; CI auto-triggers remain off.

  With slice 9 landed, the demo is end-to-end story-complete: click any flag → see real source → edit it → publish a snapshot → the proxy serves the edited value. The next recommended step is slice 7b (Depot acceleration + CI re-enable), maintainer-owned.
- 2026-05-27: Slice 10 (SQLite backend alongside Postgres) shipped on `main` over 8 phase commits. Plan: `docs/plans/2026-05-27-001-feat-sqlite-backend-plan.md`. Handoff: `docs/plans/2026-05-27-001-feat-sqlite-backend-handoff.md`.
  - **Phases 1 + 2** (pre-handoff: `8f053d9`, `3992ac0`) — `internal/store` lost the `Pool()` and `Queries()` getters and grew a `parseBackend(dsn)` dispatcher; every `Create*` store method moved UUID generation to Go (`uuid.New()`) and the Postgres migrations dropped `gen_random_uuid()` defaults so SQLite can adopt the same shape.
  - **Phase 3** (`75fe1c6`) — `sqlc.yaml` declares both engines; SQLite migrations under `db/migrations/sqlite/**` (uuid → TEXT, timestamptz → TEXT with `strftime` default, jsonb → TEXT) and queries under `db/queries/sqlite/**` (`?` placeholders, `DISTINCT ON` → `ROW_NUMBER` window, row-value cursor expanded, all `::cast` syntax dropped). `UpdateSegment`'s `now()` moved to Go for both backends. `make generate` produces `internal/db/**` and `internal/db/sqlite/**`; both expose a sqlc `Querier` interface.
  - **Phase 4** (`41c3d68`) — SQLite implementation via `modernc.org/sqlite v1.50.1` (pure Go, registered as `"sqlite"`). The package layout went single-package with `pg_*.go` / `sqlite_*.go` file prefixes rather than the plan's `internal/store/postgres/` subpackage — subpackages would have required a separate `storetypes` shim package or moving `Open` out of `store`, both more invasive than the encapsulation goal warranted. The DSN gets `_pragma=journal_mode(WAL)`, `_pragma=synchronous(NORMAL)`, `_pragma=foreign_keys(ON)`, `_pragma=busy_timeout(5000)`, `_txlock=immediate` appended; `MaxOpenConns(1)` keeps the writer serialized; `withImmediateTx` retries on `SQLITE_BUSY` (5) and `SQLITE_BUSY_SNAPSHOT` (517). The Tx interface was expanded beyond the handoff's recommended PublishFlagVersion-only shape to cover every operation WithAudit callbacks actually issue (CreateProject/Flag/Environment/Segment, UpdateSegment, PublishFlagVersion, CompileSnapshot); routing callback writes through tx makes the audit row genuinely atomic with the mutation — which the Postgres impl had *not* been (the callback wrote on a separate pool connection) — and avoids the second-connection deadlock SQLite's single-writer model would otherwise produce. `store.IsConflict` now recognizes both `pgconn.PgError` SQLSTATEs (23505 / 23503) and modernc `sqlite.Error` codes (SQLITE_CONSTRAINT_UNIQUE / PRIMARYKEY / FOREIGNKEY).
  - **Phase 5** (`3d50ba3`) — `internal/store/integration_test.go` parametrized over both backends via `forEachBackend(t, fn)`. The SQLite subtest runs unconditionally using `t.TempDir()`; the Postgres subtest is gated on `FALSEFLAG_TEST_DATABASE_URL`. `go test ./internal/store/...` is now hermetic out of the box.
  - **Phase 6** (`80f6352`) — `compose.sqlite.yaml` is a standalone stack (distinct project name `falseflag-sqlite`) with no db service. An `init-data` Alpine sidecar chowns the SQLite volume to UID 65532 (distroless `nonroot`) on every `up`; without it the api container crash-loops with `SQLITE_CANTOPEN`. `scripts/smoke.sh` learns `FALSEFLAG_BACKEND={postgres,sqlite}`, seeds the demo dataset and restarts the proxy so its first poll lands the freshly seeded `acme-web` (this also fixes a pre-existing race that affected the Postgres smoke). `make smoke` and `make smoke-sqlite` both land 14/14 hurl files green.
  - **Phase 7** (`ecf7564`) — `.github/workflows/ci.yml` fans `test-go-race`, `contract-test`, `smoke`, and `dashboard-e2e` over `backend: [postgres, sqlite]` with `fail-fast: false`. GitHub Actions services aren't matrix-conditional, so the Postgres container starts in both branches; the SQLite entry skips the wait-for-postgres step and uses a `sqlite://` DSN. `kind-smoke` stays postgres-only with a TODO pointing at slice 12 (deployment-artifact matrix owns the StatefulSet/PVC story for SQLite). Workflow YAML validates clean.
  - **Phase 8** (this commit) — README gains a "Storage backends" section near the top; METAPLAN ticks slice 10.
  Validation ladder for this slice:
  - `go build ./...` ✓; `go vet ./...` ✓.
  - `go test ./...` ✓ (SQLite half of the store integration suite runs hermetically).
  - `FALSEFLAG_TEST_DATABASE_URL=postgres://… go test ./...` ✓ (both backends green, including `TestRESTConnectParity` for both).
  - `go test -race ./internal/store/... ./internal/server/...` ✓ for both backends.
  - `make smoke` (Postgres compose) ✓ — 14/14 hurl files green.
  - `make smoke-sqlite` (SQLite compose) ✓ — 14/14 hurl files green.
  - `make seed` idempotent against both stacks (409-on-rerun swallowed).
  - `.github/workflows/ci.yml` parses (`python3 -c "import yaml; yaml.safe_load(...)"`); manual `gh workflow run ci-baseline.yml` not exercised this session (CI auto-triggers still off; slice 7b owns re-enablement).
  Known gaps remaining (still out of scope for this slice):
  - **Kubernetes SQLite path** — `kind-smoke` stays Postgres-only. The Helm chart + manifests assume a separate db deployment; the StatefulSet/PVC story for SQLite is slice 12.
  - **Backup / restore tooling** — slice 12 owns this for both backends.
  - **Migration parity checking** — by design the two schemas drift independently; no CI job compares them.
  - **Performance comparison** — no concurrent-writer benchmark; the slice is demo quality, not high-concurrency tuning.
  Branching: six phase commits directly on `main` (`8f053d9` Phase 1, `3992ac0` Phase 2, `75fe1c6` Phase 3, `41c3d68` Phase 4, `3d50ba3` Phase 5, `80f6352` Phase 6, `ecf7564` Phase 7, this commit Phase 8), no PRs, CI auto-triggers remain off (slice 7b territory). The next recommended step is `Plan slice 11: Multi-arch container images` — modernc.org/sqlite is pure Go, so the arm64 cross-compile pre-work is zero.
