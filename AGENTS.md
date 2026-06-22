# Project Instructions

## Product Goal

Build a believable, intentionally large feature flag platform for a conference demo about accelerating slow CI/CD with Depot. The software should look like it works, but it does not need production-grade hardening. Prefer broad, credible surface area and passing commands over deep implementation detail.

Use these docs as the main planning references:

- `docs/ideation/2026-05-20-synthetic-feature-flag-platform-depot-demo-ideation.md`
- `docs/ideation/2026-05-20-moonconfig-historical-reference.md`

## Chosen Stack

Backend and platform:

- Go for backend services, operator, proxy, SDK, and MCP server.
- Keep Go implementation code under `internal/**`; do not introduce a Go `pkg/**` tree.
- Follow `/Users/wito/code/project-depot/registry` as a Go repository shape reference: `cmd/**` entrypoints, `internal/**` packages, `internal/db` SQLC output, root `proto`, and root `sqlc.yaml`.
- ConnectRPC or gRPC with Buf-managed protobuf definitions.
- `oapi-codegen` for OpenAPI Go generation.
- `pgxpool` + SQLC + goose for relational persistence.
- `go-redis` for Redis integration.
- OpenTelemetry + Prometheus for observability.
- Standard `log/slog` for logging. Do not use zap or zerolog unless explicitly requested.
- `golangci-lint` and `gotestsum` for Go quality/test tooling.
- Kubebuilder/controller-runtime for the Kubernetes operator.
- Helm and Kustomize for Kubernetes packaging.

Frontend and TypeScript:

- Remix for the dashboard.
- Tailwind for styling.
- Radix UI for custom component primitives.
- Biome for TypeScript linting/formatting.
- Vitest for TypeScript tests.
- pnpm workspaces + Turborepo for the TypeScript monorepo.
- Zod for schemas and validation.
- Orval for OpenAPI TypeScript client generation.
- Commander for the TypeScript CLI.
- Playwright for browser tests.
- Keep all TypeScript code and TypeScript workspace files under `js/**`.

Code generation and CI:

- Include Buf, SQLC, controller-gen, OpenAPI generation, and Orval generation in generated-code checks.
- CI should have credible slow surfaces: Go tests, TypeScript builds, generated code checks, Docker builds, browser tests, Kubernetes/operator tests, backend matrices, and config compiler tests.

## Local Ports

The compose stack uses the following ports; keep new binaries on a contiguous range.

| Service | Ports | Notes |
|---|---|---|
| api | 8080, 8090 | REST + ConnectRPC |
| proxy | 8081 | Local snapshot evaluation |
| operator | 8082, 8083 | Metrics + health probe |
| mcp | 8091, 8092 | Streamable HTTP MCP surface + `/healthz` |
| dashboard | 3000 | Remix SSR |
| db | 5432 | Postgres |

`cmd/falseflag-mcp` exposes six agent-facing tools — `list_projects`,
`list_flags`, `get_flag`, `validate_config`, `explain_evaluation`,
`search_audit_log` — via the official `modelcontextprotocol/go-sdk`.
See `cmd/falseflag-mcp/README.md`.

## Implementation Bias

- Demo-quality is the target. Stub internals when needed, but keep commands, APIs, UI routes, and tests believable.
- Every implementation slice should end with relevant compile/build checks, API or runtime checks, and demo-path checks when applicable.
- Configuration is project-scoped: a project chooses one active strategy at a time (`json`, `cel`, or `typescript`), and each strategy compiles to the same normalized release snapshot.
- Runtime SDKs and the evaluation proxy should consume static JSON/rules, not execute user-submitted TypeScript.
- Use parallel subagents only when write scopes are clearly separate.

## Inspiration Projects

Reference material for this project lives outside this workspace at:

```text
/Users/wito/code/project-keat
```

Use those projects as read-only inspiration when designing or implementing this repository. In particular, inspect the Keat server, SDK, release tooling, Kubernetes manifests, CRDs, dashboard, and git history when they are relevant to feature flag architecture, Kubernetes-native workflows, or CI/CD demo design.

Read-only Git history inspection is allowed and encouraged, for example:

```bash
git -C /Users/wito/code/project-keat/keat-server log --oneline
git -C /Users/wito/code/project-keat/keat-server show <commit>
git -C /Users/wito/code/project-keat/keat-server blame <file>
```

Do not edit files under `/Users/wito/code/project-keat` unless the user explicitly asks for changes there and grants the necessary workspace access.
