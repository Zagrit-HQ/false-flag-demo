---
date: 2026-05-20
topic: synthetic-feature-flag-platform-depot-demo
focus: conference demo codebase for accelerating slow CI/CD with Depot
mode: repo-grounded
---

# Ideation: Synthetic Feature Flag Platform for a Depot Demo

## Grounding Context

This should be a deliberately large, credible feature flag platform rather than a real startup product. The goal is to make CI/CD slow in familiar ways, then make the improvement obvious with Depot.

`legacy-keat` is the seed. Its README frames Keat as a Kubernetes-native feature management tool where users define an `Application` CRD and the server watches Kubernetes resources, streams changes, and serves a small dashboard. The current CRD keeps `spec` schemaless with `x-kubernetes-preserve-unknown-fields`, and the TypeScript Fastify server has a synchronizer abstraction with `in-memory` and Kubernetes watching modes. The README already says the next production step is splitting the current server into `keat-proxy` and `keat-server`.

External grounding:

- Depot GitHub Actions runners are positioned as drop-in runner replacements with faster compute, faster cache orchestration, and runner labels like `depot-ubuntu-24.04`.
- Depot Cache supports remote caching for tools relevant here, including Go, Turborepo, and other build systems.
- Depot container builds support remote BuildKit, persistent layer cache, and native multi-platform builds, which matches a demo with many images.
- OpenFeature defines typed flag evaluation APIs, and OFREP defines a vendor-neutral remote evaluation protocol. This gives the synthetic platform recognizable standards without inventing every client contract.

## Topic Axes

- Flag control plane and management workflows
- Runtime evaluation data plane and SDKs
- Kubernetes-native configuration and operator behavior
- Persistence, synchronization, and auditability
- CI/CD complexity designed for acceleration

## Recommended Architecture

Build this as a monorepo with Go for the backend/platform layer and TypeScript isolated under `js/**` for the dashboard, CLI, config DSL, generated clients, and one SDK. Follow the Go repository shape from `/Users/wito/code/project-depot/registry`: `cmd/**` for binaries, `internal/**` for implementation packages, root-level contract/tooling files, and `tests/hurl/**` for API smoke tests. Do not create a Go `pkg/**` tree.

```text
cmd/
  platformcon-api/
  platformcon-proxy/
  platformcon-operator/
  platformcon-mcp/
  platformcon-loadgen/
internal/
  audit/
  config/
  db/                  SQLC-generated Go query package
  eval/
  flags/
  httpapi/
  observability/
  projects/
  proxy/
  redis/
  sdkgo/
  server/
proto/
api/
  openapi/
operator/
  api/                 CRD Go types
  controllers/
js/
  apps/
    dashboard/         Remix dashboard with Tailwind and Radix UI
    cli/               Commander CLI
  packages/
    config-ts/
    generated-client-ts/
    sdk-js/
    shared-eval-corpus/
deploy/
  helm/
  kustomize/
  manifests/
infra/
  docker-bake.hcl
  compose.yaml
test/
  hurl/
  e2e/
  fixtures/
  golden/
```

Core services:

- **Control-plane API:** owns projects, environments, flags, segments, targeting rules, API keys, audit events, approvals, and snapshots. Go service, OpenAPI exposed, gRPC optional internally.
- **Evaluation proxy:** horizontally scalable Go service optimized for read-heavy flag evaluation. It keeps a warm local cache, subscribes to updates, serves OFREP-like endpoints, and exposes SSE/WebSocket streams for SDK refreshes.
- **Kubernetes operator:** reconciles feature flag CRDs into API state, writes status back to Kubernetes, and supports GitOps workflows.
- **Dashboard:** Remix web app under `js/apps/dashboard` for flag management, segment editing, rollout monitoring, audit review, evaluation traces, and environment comparisons. Use Tailwind for styling, Radix UI for accessible custom components, Biome for linting/formatting, and Vitest for tests.
- **CLI:** TypeScript command-line client under `js/apps/cli` for `login`, `flag create`, `flag diff`, `flag promote`, `flag validate`, `seed`, `snapshot export`, and `smoke-test`.
- **SDKs:** Go SDK under `internal/sdkgo` and TypeScript SDK under `js/packages/sdk-js`; both evaluate flags locally from proxy snapshots, fall back to remote evaluation, emit telemetry, and expose OpenFeature-compatible providers.
- **Database layer:** use SQLC to generate typed Go query packages from SQL migrations and query files. Keep Postgres as the main relational target, with SQLite support where practical for local demo/test mode.
- **Backend logging:** use Go's standard `log/slog` package across Go services, CLIs, operator controllers, and tests rather than zap or zerolog.

Chosen technology stack:

- **RPC:** ConnectRPC or gRPC with Buf-managed protobuf definitions.
- **REST/OpenAPI:** `oapi-codegen` for Go server/client generation.
- **Database:** `pgxpool` for Postgres connections, SQLC for query generation, and goose for migrations.
- **Cache/pubsub:** `go-redis`.
- **Observability:** OpenTelemetry instrumentation and Prometheus metrics.
- **Logging:** Go standard `log/slog`.
- **Go quality tooling:** `golangci-lint` and `gotestsum`.
- **TypeScript workspace:** pnpm workspaces with Turborepo.
- **TypeScript validation and clients:** Zod for schemas and Orval for OpenAPI client generation.
- **CLI:** Commander.
- **Browser tests:** Playwright.
- **HTTP e2e tests:** Hurl for API smoke and end-to-end requests.
- **Kubernetes:** Kubebuilder/controller-runtime with Helm and Kustomize.
- **Code generation in CI:** Buf, SQLC, controller-gen, and OpenAPI generation.

Configuration should be project-scoped. A `Project` chooses exactly one configuration strategy at a time, and the rest of the control plane treats every strategy as a compiler that produces the same normalized release state. That keeps the demo understandable while still adding meaningful implementation and CI complexity.

Supported configuration strategies:

- **Simple JSON config:** flags and variants are stored as static JSON through Postgres/SQLite, Redis, or Kubernetes CRDs. This is the baseline mode and should be easiest to seed in tests.
- **CEL config:** targeting rules are authored in CEL, validated server-side, and compiled into the normalized rule model used by the proxy and SDKs.
- **TypeScript config:** users write a code-first config with a pure `@keat/config` DSL. The API bundles it, evaluates it in a sandboxed WASM runtime, validates the resulting data, and publishes only static JSON/rules to runtime systems.

## Ranked Ideas

### 1. Split Control Plane, Evaluation Proxy, and Operator

**Description:** Make the platform three main Go runtime binaries: `api`, `proxy`, and `operator`. The API handles mutation-heavy administrative workflows, the proxy handles high-volume evaluation traffic, and the operator reconciles Kubernetes resources into platform state.

**Axis:** Runtime evaluation data plane and SDKs

**Basis:** `direct:` `legacy-keat` already combines API, dashboard, Kubernetes watch, and proxy routes in one server, while its README says it is ready to split into `keat-proxy` and `keat-server`.

**Rationale:** This is the cleanest architecture for the demo: multiple deployable images, multiple test suites, realistic scaling story, and a direct evolution from the seed project.

**Downsides:** Adds service boundaries, generated clients, auth between services, and test orchestration.

**Confidence:** 95%

**Complexity:** Medium

**Status:** Unexplored

### 2. Make Kubernetes CRDs a First-Class Flag Source

**Description:** Expand the single schemaless `Application` CRD into typed resources: `Project`, `Environment`, `Flag`, `Segment`, `RolloutPolicy`, `FlagBinding`, and `FlagSnapshot`. The operator watches these resources, validates references, updates status, and syncs them into the control plane.

**Axis:** Kubernetes-native configuration and operator behavior

**Basis:** `direct:` `legacy-keat/k8s/manifests/crd.yaml` defines an `applications.keat.io` CRD, and the README positions Keat as Kubernetes-native.

**Rationale:** CRDs make the system more credible and create valuable CI weight: code generation, controller tests, Kubernetes manifests, envtest, Helm/Kustomize validation, and kind e2e flows.

**Downsides:** CRD versioning and status conditions can become noisy if over-modeled.

**Confidence:** 90%

**Complexity:** High

**Status:** Unexplored

### 3. Support Four Persistence and Sync Backends

**Description:** Implement storage adapters for Postgres, SQLite, Kubernetes CRDs, and Redis. Postgres is the production control-plane store and should use SQLC-generated typed Go queries. SQLite powers local demos and tests where practical, Kubernetes CRDs support GitOps mode, and Redis is used for proxy cache, pub/sub invalidation, and emergency overrides.

**Axis:** Persistence, synchronization, and auditability

**Basis:** `direct:` the user asked for multiple database backends, specifically Postgres, SQLite, and Redis, and `legacy-keat` already uses Kubernetes as a persistence/synchronization source.

**Rationale:** Backend matrices are a reliable way to create meaningful CI load without fake loops. SQLC adds a credible generated-code path for database access, and the adapters create real design questions around transactions, migrations, cache consistency, and snapshot rebuilds.

**Downsides:** Requires strict repository interfaces and a shared conformance test suite to avoid adapter drift.

**Confidence:** 88%

**Complexity:** High

**Status:** Unexplored

### 4. Adopt OpenFeature and OFREP as Compatibility Targets

**Description:** Make the Go and TypeScript SDKs expose OpenFeature providers, and make the proxy serve OFREP-inspired endpoints for single and bulk evaluation. Keep a native SDK API too, but use OpenFeature semantics for typed boolean, number, string, and object flags.

**Axis:** Runtime evaluation data plane and SDKs

**Basis:** `external:` OpenFeature requires typed flag evaluation methods, and OFREP describes a standard remote evaluation API between OpenFeature providers and flag management systems.

**Rationale:** This avoids making the SDK shape arbitrary. It also creates contract tests, generated OpenAPI clients, and cross-language golden fixtures.

**Downsides:** Full compliance may be more than needed, so the demo should claim compatibility target rather than certification.

**Confidence:** 85%

**Complexity:** Medium

**Status:** Unexplored

### 5. Build a Rich Rule Engine With Shared Golden Evaluation Tests

**Description:** Implement targeting rules for segments, prerequisites, percentage rollouts, variants, schedules, semantic versions, regex matches, geography, and JSON config values. Keep one language-agnostic golden corpus that both SDKs, the proxy, and the API use for conformance tests.

**Axis:** Flag control plane and management workflows

**Basis:** `reasoned:` feature flag systems become credible when evaluation rules are deterministic and cross-SDK behavior matches. Shared fixtures turn that credibility into real Go and TypeScript test work.

**Rationale:** This creates lots of tests that are legitimate: property tests, fuzz tests, golden tests, cross-language conformance tests, and benchmark suites.

**Downsides:** Rule engines can sprawl; keep the grammar explicit and generated from JSON Schema or protobuf.

**Confidence:** 86%

**Complexity:** High

**Status:** Unexplored

### 6. Add Project-Scoped Configuration Strategies

**Description:** Let each project choose one configuration strategy: simple JSON, CEL, or TypeScript config-as-code. Each strategy compiles into the same normalized release snapshot, so the proxy and SDKs only consume static data and never care how the configuration was authored.

**Axis:** Flag control plane and management workflows

**Basis:** `direct:` the user asked for simple JSON, CEL targeting rules, and a TypeScript/WASM strategy inspired by the historical MoonConfig work in `/Users/wito/code/project-keat/keat-release`.

**Rationale:** This adds real product complexity without forcing all strategies to interoperate at runtime. It creates separate validators, compilers, persistence paths, fixtures, CLI commands, API endpoints, and test matrices while preserving a single edge-safe evaluation model.

**Downsides:** The boundary must stay strict. If SDKs or the proxy start executing CEL or TypeScript directly, the data plane becomes harder to secure and reason about.

**Confidence:** 91%

**Complexity:** High

**Status:** Unexplored

### 7. Reintroduce MoonConfig as TypeScript Config-as-Code

**Description:** Reintroduce the abandoned MoonConfig concept as a deliberate TypeScript configuration strategy. Users author environments, stages, typed args, features, and rules in TypeScript; the API bundles the source, evaluates the default export in a sandboxed WASM runtime, validates the output, and stores compiled static release JSON.

**Axis:** Persistence, synchronization, and auditability

**Basis:** `direct:` commit `07b6918` in `/Users/wito/code/project-keat/keat-release` added `apps/api/src/utils/moonConfig.ts`, `apps/api/src/utils/esbuild.ts`, `MutationAppConfigure`, and `MutationAppRelease`. The old implementation used `quickjs-emscripten` to evaluate bundled config and then wrote feature rules to database and edge KV.

**Rationale:** This is highly demo-friendly complexity: TypeScript DSL package, esbuild bundling, sandbox runtime, import allow-listing, diagnostics, deterministic execution policy, validation, source storage, bundle storage, and compilation to edge-safe static rules. It also adds a distinctive story from the real Keat history.

**Downsides:** Sandbox security, timeouts, memory limits, deterministic execution, and source mutation are real risks. The MVP should validate and save config first, then defer automatic code mutation/release editing.

**Confidence:** 87%

**Complexity:** High

**Status:** Unexplored

### 8. Create a Dashboard That Surfaces Operational Complexity

**Description:** The dashboard should be a Remix application that manages projects, environments, flags, segments, approval workflows, audit logs, rollout status, evaluation traces, and SDK connection health. Use Tailwind for the styling system, Radix UI for custom components, Biome for linting/formatting, and Vitest for unit/component tests. It should include dense admin views rather than a marketing-style UI.

**Axis:** Flag control plane and management workflows

**Basis:** `direct:` `legacy-keat/ui` already has a React dashboard for applications and features; the user explicitly wants a dashboard web application.

**Rationale:** A dashboard creates TypeScript build/test weight, browser e2e coverage, generated API clients, visual complexity, and a clear conference demo surface.

**Downsides:** Easy to over-invest in polish. Build enough UI to exercise real workflows and tests.

**Confidence:** 82%

**Complexity:** Medium

**Status:** Unexplored

### 9. Make CI Slow in Realistic, Depot-Friendly Ways

**Description:** Design the baseline CI to run serially across Go, TypeScript, Docker, Kubernetes, browser, and integration-test jobs, then provide a Depot-optimized workflow that uses Depot runners, Depot Cache for Go/TypeScript build artifacts, and Depot remote container builds via `docker-bake.hcl`.

**Axis:** CI/CD complexity designed for acceleration

**Basis:** `external:` Depot documents GitHub Actions runner label swaps, remote cache support for Go/Turborepo-style workloads, and remote container builds with persistent layer cache and native multi-platform support.

**Rationale:** This is the conference payoff. The system should not merely have slow code; it should be slow for reasons Depot can visibly improve: container builds, dependency installs, test parallelism, cacheable compilation, and multi-arch images.

**Downsides:** Avoid artificial sleeps. The contrast should be credible enough that attendees believe the acceleration.

**Confidence:** 94%

**Complexity:** Medium

**Status:** Unexplored

## Suggested CI Shape

Baseline workflow:

- `go test ./...` for API, proxy, operator, SDK, repositories, rule engine.
- `go test -race ./...` for shared evaluation and repository packages.
- `pnpm install`, `pnpm lint`, `pnpm typecheck`, `pnpm test`, `pnpm build` for the Remix dashboard, CLI, SDK, config DSL, and generated client packages. TypeScript lint/format should run through Biome, and unit tests should run through Vitest.
- Generate OpenAPI, protobuf, CRDs, SQLC query packages, clients, mocks, and docs through `oapi-codegen`, Buf, controller-gen, SQLC, and Orval, then fail on dirty diff.
- Build images for `api`, `proxy`, `operator`, `dashboard`, `synthetic-client`, and `loadgen`.
- Run Postgres, SQLite, Redis, and Kubernetes-backed integration suites.
- Run kind e2e tests for Helm/Kustomize/operator.
- Run Hurl API e2e tests against the API server.
- Run Playwright dashboard tests.
- Run SDK conformance tests against the shared golden corpus.
- Run config compiler suites for JSON, CEL, and TypeScript/WASM modes.
- Run sandbox safety tests for disallowed imports, infinite loops, memory pressure, deterministic output, and diagnostic mapping.
- Run benchmarks and publish evaluation latency artifacts.

Depot-optimized workflow:

- Change GitHub runner labels to Depot runners.
- Use Depot Cache for Go and TypeScript build/test caches.
- Use `depot bake` for all container images and multi-platform builds.
- Split matrix jobs aggressively: backend adapters, browser tests, operator tests, SDK conformance, and images run in parallel.
- Add CI timing dashboards so the before/after improvement is visible during the talk.

## Rejection Summary

| # | Idea | Reason Rejected |
|---|------|-----------------|
| 1 | Full billing/subscription system | Too far from the feature flag/platform demo and would distract from CI acceleration. |
| 2 | Mobile SDKs | Useful in a real product, but outside the stated Go/TypeScript constraint. |
| 3 | ML rollout optimizer | Interesting but too expensive and not needed to demonstrate slow builds. |
| 4 | Global active-active database replication | Creates distributed-systems complexity that does not help the Depot story enough. |
| 5 | Terraform provider | Plausible later client, but lower demo value than dashboard, CLI, SDKs, and Kubernetes operator. |
| 6 | Artificial sleeps in CI | Below the ambition floor; the slow pipeline should be credible and acceleration-friendly. |

## Sources

- Depot GitHub Actions runners: https://depot.dev/docs/github-actions/overview
- Depot Cache: https://depot.dev/docs/cache/overview
- Depot container builds: https://depot.dev/docs/container-builds/overview
- OpenFeature flag evaluation: https://openfeature.dev/specification/sections/flag-evaluation/
- OpenFeature Remote Evaluation Protocol: https://openfeature.dev/docs/reference/other-technologies/ofrep/openapi/
