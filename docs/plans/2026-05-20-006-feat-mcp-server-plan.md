---
title: "feat: MCP server slice for FalseFlag demo"
type: feat
status: active
date: 2026-05-20
---

# Slice 6 — MCP server (`cmd/falseflag-mcp`)

## Overview

Add an agent-facing Model Context Protocol (MCP) server as a sixth top-level Go binary in the FalseFlag demo monorepo. The server uses the official `github.com/modelcontextprotocol/go-sdk` v1.x, listens via Streamable HTTP on `:8091`, and exposes six tools that wrap the existing Connect RPC control-plane (running on `:8090` of the `api` service):

| Tool | Backing call | Notes |
|---|---|---|
| `list_projects` | `ProjectsService.ListProjects` | `protojson` passthrough |
| `list_flags` | `FlagsService.ListFlags` | input: `{ project_slug }` |
| `get_flag` | `FlagsService.GetFlag` | input: `{ project_slug, flag_key }`; surfaces `published_version: null` for unpublished flags |
| `validate_config` | in-process `config.Compile` | input: `{ strategy, source }`; **does not** hit the API |
| `explain_evaluation` | `EvaluationService.EvaluateWithTrace` | input: `{ project_slug, flag_key, context }`; flattens trace tree into MCP-friendly DTO |
| `search_audit_log` | `AuditService.ListAuditEvents` | `project_slug` **required**; filters: `action?`, `actor?`, `from?`, `to?`, `limit?` |

The binary stub already exists (`cmd/falseflag-mcp/main.go`, ~34 lines, currently a no-op `run()` from slice 1). Slice 6 fills in the implementation, wires the compose stack and `docker-bake` default group, and adds Hurl smoke coverage and an in-process tool-call test suite.

Quality bar is **demo-quality**, matching slices 2–5. No bearer-token auth on the MCP listener; `X-Actor: mcp/<agent>` stamping carries audit attribution only.

## Motivation

The conference demo pitches FalseFlag as a believably complete platform. An MCP server is the agent-native control surface that lets Claude (and any MCP-aware client) drive the same workflows the dashboard and CLI already cover. It is also a credible slow surface for CI (slice 7) — `go test ./internal/mcp/...`, a Hurl smoke job, and a `docker-bake` image build all light up.

The metaplan calls this out under Fan-Out D (prerequisite: API contract skeleton — shipped in slice 3) and describes the same six tools.

## Decisions already locked in by research

These come from research run at the top of this plan and should not be re-litigated during implementation.

1. **SDK: `github.com/modelcontextprotocol/go-sdk` v1.x** (official, stable since late 2025, currently v1.5.0 as of 2026-04-07). Reasons: zero third-party runtime deps (stdlib + internal `jsonschema`), the project's stated bias toward mainstream/official libraries (`AGENTS.md`), and the in-process test transport (`mcp.NewInMemoryTransports`) matches the slice-4 controller-runtime fake pattern.
2. **Transport: Streamable HTTP only on `:8091`.** Stdio would force `docker compose run` per session and is awkward for the demo. Streamable HTTP turns the MCP server into an always-on HTTP service reachable by service name from sibling containers and by `claude mcp add --transport http` from a developer laptop.
3. **Separate `/healthz` on `:8092`.** Same dual-port pattern the API uses (`:8080` REST + `:8090` Connect). Keeps the streamable-HTTP handler routing on `:8091` clean and gives compose `healthcheck:` something concrete to probe.
4. **`internal/operator/clientapi.Client` is the reuse target, not a parallel package.** Add an `Evaluation falseflagv1connect.EvaluationServiceClient` field to the existing struct. The operator never calls evaluate so it sets that field to `nil` in tests (mirroring the existing `Audit: nil` precedent at `internal/operator/controllers/suite_test.go:103`). No new client package.
5. **`validate_config` is in-process** — calls `config.Compile(strategy, source)` from `internal/config/strategy.go:56` directly. No API round-trip, no new RPC.
6. **Tool output shapes:** `protojson` passthrough for `list_projects`, `list_flags`, `get_flag`, `search_audit_log` (proto field names are already `snake_case` and agent-friendly). Hand-shaped DTOs only for `validate_config` (synthetic `ir_summary`) and `explain_evaluation` (flatten the trace tree).
7. **Tool errors return `isError: true` content blocks**, not protocol-level errors. Per MCP spec (2025-11-25 ed.) — protocol errors are swallowed by the client framework and never reach the LLM. Shared helper `connectErrToMCPContent(err error) *mcp.CallToolResult` enforces consistency.
8. **`search_audit_log` requires `project_slug`** because the underlying `AuditService.ListAuditEvents` RPC unconditionally calls `store.GetProjectBySlug`. Matching the RPC reality is simpler than adding cross-project paths for the demo.
9. **`search_audit_log` does NOT suppress `mcp/*` actors.** The demo moment of "agent inspecting its own prior actions" is exactly the point. Tool description names `mcp/falseflag-mcp` so the LLM can interpret entries correctly.

## Acceptance Criteria

### Compile & generate

- [ ] `go build ./cmd/...` builds `cmd/falseflag-mcp` to a working binary (no panics on `--help` or `-h` if a flag set exists, otherwise a successful clean exit on SIGTERM).
- [ ] `go vet ./...` clean.
- [ ] `go test ./...` passes including new `./internal/mcp/...` test packages.
- [ ] `make generate-check` still passes (no codegen drift).

### Behavioural (per tool)

- [ ] `list_projects` returns the seeded three projects when run against the compose stack after `make seed`.
- [ ] `list_flags` returns 7 flags for `acme-web`, errors with `isError: true` and message `project not found: <slug>` for unknown slugs.
- [ ] `get_flag` returns flag + `published_version` for a published flag; returns `published_version: null` (not error) for a created-but-unpublished flag; returns `isError: true` for a missing flag key.
- [ ] `validate_config` returns `{ valid: true, ir_summary: { flag_count, rule_count, has_rollout } }` for valid JSON/CEL/TypeScript samples; returns `{ valid: false, errors: [...] }` for malformed source; returns `isError: true` for unknown strategy values; returns `{ valid: false, errors: ["empty source"] }` for empty source rather than panicking inside `config.Compile`.
- [ ] `explain_evaluation` returns the same `Decision` value that `EvaluationService.Evaluate` would return, plus a flattened `trace: [{rule_id, matched, reason, predicate_path}]` list.
- [ ] `search_audit_log` returns paginated events; `project_slug` is required and missing-project returns `isError: true`.

### Audit attribution

- [ ] Every outbound Connect call from the MCP server carries `X-Actor: mcp/<actor>` (default `mcp/falseflag-mcp`, configurable via `FALSEFLAG_MCP_ACTOR`). Verified by a clientapi-level integration test that hits a real compose API and reads `audit_events.actor`.

### Tests

- [ ] `internal/mcp/server_test.go` boots the MCP server with `mcp.NewInMemoryTransports()`, connects an in-process client, and table-drives all six tools through happy + sad paths against fakes that follow the slice-4 `fakeAPI` pattern (`internal/operator/controllers/suite_test.go:65`).
- [ ] `internal/mcp/errors_test.go` covers `connectErrToMCPContent` for `not_found`, `invalid_argument`, `unavailable`, and unknown codes.
- [ ] `internal/mcp/tools/validate_config_test.go` covers the three strategies plus unknown-strategy and empty-source guards.

### Smoke & wiring

- [ ] `infra/compose.yaml` has an `mcp` service using the `*go-build` anchor, exposing `8091:8091` and `8092:8092`, `depends_on: api: condition: service_started`, with a `healthcheck:` probing `:8092/healthz`.
- [ ] `infra/docker-bake.hcl` default group includes `"mcp"`.
- [ ] `tests/hurl/12-mcp-tools.hurl` POSTs the streamable-HTTP `initialize` → `tools/list` → `tools/call list_projects` sequence and asserts a 200 + `result.content[0].type == "text"` + a known project slug in the payload.
- [ ] `make mcp-smoke` runs that Hurl file against a running compose stack. Implemented as `./scripts/mcp-smoke.sh` parallel to `scripts/smoke.sh`.
- [ ] `make smoke` continues to pass (the MCP service is wired but the existing 11 Hurl files don't depend on it).

### Documentation

- [ ] `cmd/falseflag-mcp/README.md` updated with: tool list, how to point `claude mcp add` at the running compose instance, env vars, port assignments.
- [ ] `AGENTS.md` gains one paragraph under the architecture map noting the MCP surface and port layout.
- [ ] Full README quickstart polish is deferred to slice 8 — call out explicitly in status notes.

## Phased implementation plan

Six phases, each ending with a small, reviewable commit set. Sequential by default (the slice is small enough that fan-out parallelism isn't warranted; phase 1 unblocks everything else).

### Phase 1 — Foundation

Owned paths: `go.mod`, `go.sum`, `internal/operator/clientapi/client.go`, `internal/appconfig/appconfig.go`, `cmd/falseflag-mcp/main.go`, `internal/mcp/run.go` (new, empty skeleton).

1. `go get github.com/modelcontextprotocol/go-sdk@latest` and verify the import path under `mcp` (`github.com/modelcontextprotocol/go-sdk/mcp`). Commit `go.mod` + `go.sum` separately for review clarity.
2. Add `Evaluation falseflagv1connect.EvaluationServiceClient` field to `clientapi.Client`. Wire in `New()` as `Evaluation: falseflagv1connect.NewEvaluationServiceClient(httpClient, baseURL, opts...)`. The operator tests at `internal/operator/controllers/suite_test.go:103` already use the `field: nil` pattern — leave them alone; they will compile because `nil` interface fields are fine.
3. Add `appconfig.MCPConfig` and `appconfig.LoadMCP()` mirroring `LoadOperator()` in `internal/appconfig/appconfig.go`:
   ```go
   type MCPConfig struct {
       Addr        string // FALSEFLAG_MCP_ADDR, default ":8091"
       HealthAddr  string // FALSEFLAG_MCP_HEALTH_ADDR, default ":8092"
       APIBaseURL  string // FALSEFLAG_API_RPC_ADDR, default "http://localhost:8090"
       Actor       string // FALSEFLAG_MCP_ACTOR, default "mcp/falseflag-mcp"
       LogLevel    string // shared with logging.New
   }
   ```
4. Replace `cmd/falseflag-mcp/main.go` `run()` body with a delegation to `mcp.Run(ctx)`. Keep the file under 50 lines per `AGENTS.md`.
5. Create empty `internal/mcp/run.go` with a `Run(ctx context.Context) error` stub that just calls `logging.New("mcp")`, loads config, and returns — fills in over the next phases.

**Validation:** `go build ./cmd/...`, `go vet ./...`, `go test ./internal/operator/...` (existing).

### Phase 2 — MCP server skeleton + health

Owned paths: `internal/mcp/run.go`, `internal/mcp/health.go` (new), `internal/mcp/server.go` (new).

1. `internal/mcp/server.go`: a `newServer(cfg MCPConfig, client *clientapi.Client) *mcp.Server` factory that constructs the `mcp.Server` with `&mcp.Implementation{Name: "falseflag-mcp", Version: buildinfo.Version()}` and returns it without yet registering tools.
2. `internal/mcp/health.go`: a tiny `http.Handler` returning `{"status":"ok","service":"falseflag-mcp","probe":"liveness"}` on `/healthz`, matching the API's `/v1/health` JSON shape. Bound to `:8092`.
3. `internal/mcp/run.go`: assemble both listeners under an `errgroup` (same pattern as `internal/server/server.go` which runs REST + Connect together). Pass `mcp.NewStreamableHTTPHandler(...)` to the `:8091` listener.
4. Verify by running `go run ./cmd/falseflag-mcp` locally: `curl localhost:8092/healthz` returns 200; `curl localhost:8091/` returns the MCP Streamable HTTP greeting (without tools yet it'll return an empty `tools/list`).

**Validation:** `go build`, `go vet`, manual `curl` of both ports.

### Phase 3 — Read-only tools (list_projects, list_flags, get_flag)

Owned paths: `internal/mcp/tools/` (new directory), `internal/mcp/tools/projects.go`, `internal/mcp/tools/flags.go`, `internal/mcp/errors.go`, `internal/mcp/errors_test.go`.

1. `internal/mcp/errors.go`: define
   ```go
   func connectErrToMCPContent(err error) *mcp.CallToolResultFor[any] {
       code := connect.CodeOf(err)
       msg := userFacingMessage(code, err)
       return &mcp.CallToolResultFor[any]{
           IsError: true,
           Content: []mcp.Content{&mcp.TextContent{Text: msg}},
       }
   }
   ```
   with a `userFacingMessage` switch covering `connect.CodeNotFound`, `CodeInvalidArgument`, `CodeUnavailable`, `CodePermissionDenied`, and a default. Cover with `errors_test.go`.
2. `tools/projects.go`: input struct `ListProjectsInput struct{}`, handler calls `client.Projects.ListProjects(ctx, connect.NewRequest(&pb.ListProjectsRequest{}))`, marshals via `protojson.Marshal` into a `TextContent`. Register via `mcp.AddTool(server, &mcp.Tool{Name: "list_projects", Description: "..."}, listProjectsHandler)`.
3. `tools/flags.go`: two handlers (`list_flags`, `get_flag`). Input structs:
   ```go
   type ListFlagsInput struct {
       ProjectSlug string `json:"project_slug" jsonschema:"required,description=slug of the project"`
   }
   type GetFlagInput struct {
       ProjectSlug string `json:"project_slug" jsonschema:"required"`
       FlagKey     string `json:"flag_key" jsonschema:"required"`
   }
   ```
   For `get_flag`, when `version.published_at` is zero, emit a non-error result with `published_version: null` and a `note` field rather than returning an error — this is a deliberately distinct state from "flag not found."
4. Register all three in `internal/mcp/server.go`'s `RegisterTools(s *mcp.Server, client *clientapi.Client)` exported function called from `Run`.

**Validation:** `go test ./internal/mcp/...` exercises happy + sad paths against `fakeAPI`. Live-stack manual smoke: `curl -X POST http://localhost:8091/ -d '{"jsonrpc":"2.0",...}'` returns the seeded projects.

### Phase 4 — Config validation + evaluation explanation

Owned paths: `internal/mcp/tools/validate_config.go`, `internal/mcp/tools/explain_evaluation.go`, `internal/mcp/tools/validate_config_test.go`.

1. `validate_config`: input struct
   ```go
   type ValidateConfigInput struct {
       Strategy string `json:"strategy" jsonschema:"required,enum=json,enum=cel,enum=typescript"`
       Source   string `json:"source" jsonschema:"required"`
   }
   type ValidateConfigOutput struct {
       Valid     bool                    `json:"valid"`
       Errors    []string                `json:"errors,omitempty"`
       IRSummary *ValidateConfigSummary  `json:"ir_summary,omitempty"`
   }
   type ValidateConfigSummary struct {
       FlagCount    int  `json:"flag_count"`
       RuleCount    int  `json:"rule_count"`
       HasRollout   bool `json:"has_rollout"`
   }
   ```
   Handler maps the SDK enum to `config.Strategy`, returns `{valid:false}` for empty source, calls `config.Compile(strategy, []byte(source))`, walks the returned `*config.Compiled` to populate `IRSummary`. Errors from `Compile` go into `Errors[]`. Unknown strategies (shouldn't reach here because of the enum, but defensively) return `isError`.
2. `explain_evaluation`: input struct
   ```go
   type ExplainEvaluationInput struct {
       ProjectSlug string         `json:"project_slug" jsonschema:"required"`
       FlagKey     string         `json:"flag_key" jsonschema:"required"`
       Context     map[string]any `json:"context"`
   }
   ```
   Encode `Context` to a `structpb.Struct` (helper already in repo — search `internal/server/rpc/evaluation.go` for the existing pattern), call `client.Evaluation.EvaluateWithTrace(...)`, flatten the response into:
   ```go
   type ExplainEvaluationOutput struct {
       Value   any           `json:"value"`
       Reason  string        `json:"reason"`
       RuleID  string        `json:"rule_id,omitempty"`
       Version int32         `json:"version"`
       Trace   []TraceStep   `json:"trace"`
   }
   type TraceStep struct {
       RuleID    string `json:"rule_id"`
       Matched   bool   `json:"matched"`
       Reason    string `json:"reason"`
       Predicate string `json:"predicate_path"` // e.g. "and[0].eq(user.plan,pro)"
   }
   ```
   The flattening walks the existing trace tree (`internal/server/rpc/evaluation.go` for the producing shape).
3. `validate_config_test.go`: table-driven across JSON/CEL/TypeScript with one valid and one invalid fixture each, plus unknown-strategy and empty-source guards. Fixtures inline as Go string literals (no separate files — small enough).

**TypeScript caveat:** TypeScript validation currently runs through `internal/config/typescript.go` which produces an IR-shaped JSON output expected on the wire (per slice 2 status notes — there's no esbuild/QuickJS sandbox yet). The `validate_config` tool exercises the same in-process compile path, so no extra subprocess is required. If TypeScript compilation later moves to a sandbox, this tool may grow a `runtime_unavailable` branch; for slice 6 it does not.

**Validation:** `go test ./internal/mcp/...`, live-stack `claude mcp` call to `validate_config`.

### Phase 5 — Audit search + tool registry finalization

Owned paths: `internal/mcp/tools/audit.go`, `internal/mcp/server.go` (RegisterTools finalization).

1. `search_audit_log`: input
   ```go
   type SearchAuditLogInput struct {
       ProjectSlug string  `json:"project_slug" jsonschema:"required"`
       Action      string  `json:"action,omitempty" jsonschema:"description=optional action filter, e.g. publish_version"`
       Actor       string  `json:"actor,omitempty"`
       From        string  `json:"from,omitempty" jsonschema:"description=RFC3339 timestamp"`
       To          string  `json:"to,omitempty"`
       Limit       int32   `json:"limit,omitempty" jsonschema:"minimum=1,maximum=200"`
       Cursor      string  `json:"cursor,omitempty"`
   }
   ```
   Parse `From`/`To` as RFC3339 to `timestamppb.Timestamp` pointers (tool-layer guards return `isError` for malformed timestamps rather than passing them through). Limit defaults to 50 when 0. Wraps `client.Audit.ListAuditEvents(...)`. Passthrough output via `protojson`. Tool description names `mcp/falseflag-mcp` as the MCP server's own actor string so an LLM can interpret matching entries.
2. `RegisterTools` becomes the single canonical registration point: `mcp.AddTool` for all six tools with descriptions sized for an LLM context window (one-sentence purpose + one-sentence input expectation per tool). Drift between tool names in code and tool names in docs/Hurl is prevented by referencing a single `package tools` exported constant slice `Tools = []string{...}`.

**Validation:** `go test ./internal/mcp/...` full suite.

### Phase 6 — Wiring: compose, bake, Hurl, Makefile, README

Owned paths: `infra/compose.yaml`, `infra/docker-bake.hcl`, `tests/hurl/12-mcp-tools.hurl` (new), `scripts/mcp-smoke.sh` (new), `Makefile`, `cmd/falseflag-mcp/README.md`, `AGENTS.md`.

1. `infra/compose.yaml`: new service `mcp:` block, `<<: *go-build`, `SERVICE: falseflag-mcp`, `ports: ["8091:8091", "8092:8092"]`, `depends_on: api: condition: service_started`, env: `FALSEFLAG_API_RPC_ADDR=http://api:8090`, `FALSEFLAG_MCP_ACTOR=mcp/falseflag-mcp`, `healthcheck: test: ["CMD-SHELL", "wget -qO- http://localhost:8092/healthz | grep -q ok"]`, interval 5s, retries 5.
2. `infra/docker-bake.hcl`: add `"mcp"` to `group "default" targets`. The `target "mcp"` block already exists, no other changes.
3. `tests/hurl/12-mcp-tools.hurl`: three requests:
   - POST `/mcp` with `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{...}}` → assert 200 + `result.protocolVersion`.
   - POST `/mcp` with `tools/list` → assert 200 + jsonpath `$.result.tools[*].name` contains all six tool names.
   - POST `/mcp` with `tools/call list_projects` → assert 200 + `$.result.content[0].text` contains `"acme-web"`.
   - Base URL: `{{base_url}}` env-substituted by `scripts/mcp-smoke.sh`.
4. `scripts/mcp-smoke.sh`: parallel to `scripts/smoke.sh`. Defaults `FALSEFLAG_MCP_BASE_URL=http://localhost:8091`. Runs `hurl --test --variable base_url=$FALSEFLAG_MCP_BASE_URL tests/hurl/12-mcp-tools.hurl`. Best-effort — exits 0 with a warning if `hurl` not on `$PATH`, matching how `kind-smoke.sh` handles missing `kind`.
5. `Makefile`: add `mcp-smoke: ## Run MCP smoke checks against the running compose stack` invoking `./scripts/mcp-smoke.sh`. Add to `make help` output. Do **not** add to the umbrella `make smoke` target so existing CI doesn't suddenly depend on compose having the MCP service running.
6. `cmd/falseflag-mcp/README.md`: full rewrite. Sections: what it is, six tools (one-line description each), running it locally (`docker compose up mcp`), pointing `claude mcp add --transport http http://localhost:8091/` at it, env vars, ports.
7. `AGENTS.md`: one paragraph appended to the architecture map under existing binary list, naming `cmd/falseflag-mcp` and noting port 8091/8092.

**Validation:** Full ladder (see below).

## Validation Ladder

Run all of these at the end of slice 6 before updating METAPLAN status notes.

1. `go build ./cmd/...` ✓
2. `go vet ./...` ✓
3. `go test ./...` ✓ (now 18 internal packages with tests, +1 over slice 5)
4. `make generate-check` ✓
5. `make conformance` ✓ (regression — slice 5 contract unchanged)
6. `pnpm --dir js -r typecheck && pnpm --dir js -r test && pnpm --dir js -r build && pnpm --dir js lint` ✓ (regression — no TS touched)
7. `docker compose up --build mcp` ✓ — service comes up, `/healthz` returns ok.
8. `make smoke` ✓ — existing 11 Hurl files still pass.
9. `make mcp-smoke` ✓ — new file passes against compose stack.
10. `make bake-print` ✓ — default group now `["api","proxy","operator","mcp","dashboard"]`.
11. Manual: `claude mcp add --transport http falseflag http://localhost:8091/` then `claude mcp call falseflag list_projects` returns the seeded list. Record the transcript in commit message or status notes.

If `claude mcp` CLI isn't available on the dev machine, the Hurl smoke covers the same protocol surface — record this as a known gap in status notes (consistent with how slice 5 deferred Playwright Chromium install).

## System-Wide Impact

### Interaction graph

`MCP client (Claude / curl) → :8091 Streamable HTTP handler → mcp.Server.dispatchToolCall → tool handler → clientapi.Client → connect.UnaryInterceptor (X-Actor) → :8090 Connect RPC → store/<x> queries → audit_events row (for mutations, not for the six read-only tools)`.

No new write paths are introduced. All six tools are read-only against the upstream API. The only audit_events writes come incidentally — none of the six MCP tools call a mutating RPC. This means the X-Actor `mcp/...` stamps never appear in the audit log during normal slice-6 usage. The wiring is still mandatory for forward compatibility (slice 8 or beyond may add `publish_flag_version` to the tool set).

### Error propagation

Three error tiers:
1. **Transport-level** (HTTP 4xx/5xx, malformed JSON-RPC) — surfaced by the MCP SDK as protocol errors. Returned by handlers via `return nil, err`.
2. **Tool-level** (Connect call failed, validation failed) — returned in the tool result with `IsError: true`. Mapped consistently by `connectErrToMCPContent`. The LLM sees this and can self-correct.
3. **Application-level** (e.g., `validate_config` returns `{valid: false, errors: [...]}`) — a successful tool call returning a structured negative result. Distinct from tier 2 because the agent's expected behavior differs ("the input was invalid, here's why" vs "the tool itself failed").

### State lifecycle risks

None — all six tools are read-only against the API.

### API surface parity

The MCP tools are a strict subset of the Connect RPC surface plus the in-process `config.Compile`. There is no MCP-only state. Anything an MCP agent can do, the operator and CLI can already do via Connect. No drift risk.

### Integration test scenarios

Three scenarios not covered by unit tests with fakes:

1. **MCP → live API → live Postgres.** `tests/hurl/12-mcp-tools.hurl` exercises `list_projects` against the seeded compose stack. Catches: protojson mis-marshaling, Connect interceptor not stamping X-Actor, port misconfiguration.
2. **X-Actor end-to-end attribution.** If a future tool mutates state, run a manual test that calls the tool and then verifies an `audit_events` row with `actor = 'mcp/falseflag-mcp'`. Out of scope for this slice's automated tests since no mutating tools are exposed, but documented in `cmd/falseflag-mcp/README.md` as the expected behavior for future tools.
3. **MCP server resilience to API restart.** Compose `restart: unless-stopped` (already on other services — add to mcp). Manual test: stop and restart `api`, confirm subsequent `tools/call list_projects` succeeds without restarting MCP. The Connect client uses HTTP/1.1 with default keepalive; this should "just work" but is worth confirming.

## Dependencies & Risks

| Risk | Mitigation |
|---|---|
| Official Go SDK API changes between v1.5.0 and whatever is current at implementation time | Pin a specific version in `go.mod` (`go get github.com/modelcontextprotocol/go-sdk@v1.5.0` or current at implementation). Re-pin only if a security release is required. |
| `EvaluationService` is missing from `clientapi.Client` and any other internal code that constructs `clientapi.Client{}` literally (rather than via `New`) will break | Phase 1 step 2 audits for direct struct literal construction. `grep -r 'clientapi.Client{' .` in the foundation step. The operator test pattern uses `&clientapi.Client{Projects: ...}` which means adding the new field with no initializer means `nil` — safe because the field is an interface. Confirmed safe but worth grepping. |
| Hurl JSON-RPC assertion brittleness — protocol responses include negotiated session IDs in headers | The Hurl file should assert only on response body jsonpath, not headers. Pin assertions to `result.content[0].text` substring matches, not exact equality. |
| `docker compose up mcp` may race with `api` startup if `api` has no healthcheck stanza | The metaplan + slice 4 status note that `api` has no healthcheck yet. Phase 6 adds an `api` healthcheck stanza too — small adjacent change but cleanly scoped. If reviewers push back on touching `api` config in this slice, fall back to a startup-poll loop inside `internal/mcp.Run` that pings the Connect health service until ready. Preferred order: add `api` healthcheck (one-line YAML), only fall back to startup-poll if the healthcheck causes other regressions. |
| `claude mcp add` CLI may not be installed on dev/CI machines | Hurl smoke is the primary automation; `claude mcp` is documented as a manual verification path. Same precedent as Playwright Chromium in slice 5. |
| TypeScript `validate_config` could grow to need a real esbuild/QuickJS sandbox later | Out of scope for slice 6. Slice 2 status notes already defer this; slice 6 inherits the same boundary. |

## Out of Scope

Explicitly deferred to later slices:

- **Authentication / authorization on the MCP listener.** No bearer tokens, no API keys, no per-tool permission checks. Slice 8 may add a stub.
- **Mutating tools.** `create_project`, `publish_flag_version`, `update_segment`, etc. — the slice exposes only read/inspect tools. The mutation surface exists in the Connect API and could be added in a future slice without re-architecting.
- **MCP Resources & Prompts.** The SDK supports MCP "resources" (file-like content) and "prompts" (parameterized prompt templates). Slice 6 ships tools only — these are not part of the demo story.
- **Streaming tool responses.** Streamable HTTP supports server-sent events for long-running tool calls. None of the six tools are long-running; all are synchronous Connect calls.
- **Cross-runtime conformance.** Slice 5's golden corpus is a Go-vs-TS evaluation contract. There is no TS MCP server, so no equivalent. The MCP tools indirectly inherit the corpus via the Connect API they call.
- **Slice 7 CI integration** (golangci-lint job for MCP, image build, Hurl job). Slice 7 owns CI; this slice ensures the artifacts (image target, Hurl file, Makefile target) exist for slice 7 to wire.
- **Slice 8 README polish** (top-level README quickstart that mentions MCP, demo script, screenshot). Slice 8 owns the README; this slice updates only `cmd/falseflag-mcp/README.md` and one `AGENTS.md` paragraph.

## File touch map (estimate)

New files (~14):
- `internal/mcp/run.go`
- `internal/mcp/server.go`
- `internal/mcp/health.go`
- `internal/mcp/errors.go`
- `internal/mcp/errors_test.go`
- `internal/mcp/server_test.go`
- `internal/mcp/tools/projects.go`
- `internal/mcp/tools/flags.go`
- `internal/mcp/tools/validate_config.go`
- `internal/mcp/tools/validate_config_test.go`
- `internal/mcp/tools/explain_evaluation.go`
- `internal/mcp/tools/audit.go`
- `tests/hurl/12-mcp-tools.hurl`
- `scripts/mcp-smoke.sh`

Modified files (~7):
- `go.mod`, `go.sum`
- `cmd/falseflag-mcp/main.go` (replace `run()` body)
- `cmd/falseflag-mcp/README.md` (full rewrite)
- `internal/operator/clientapi/client.go` (+ `Evaluation` field)
- `internal/appconfig/appconfig.go` (+ `MCPConfig`, `LoadMCP`)
- `infra/compose.yaml` (+ `mcp` service, + `api` healthcheck)
- `infra/docker-bake.hcl` (`mcp` joins default group)
- `Makefile` (+ `mcp-smoke` target)
- `AGENTS.md` (+ one paragraph)

Total: ~21 files, single autonomous implementation pass should finish in a similar shape to slice 4 (6 phase-bundling commits).

## Status note draft (for METAPLAN after implementation)

To be filled in during implementation and pasted into `docs/METAPLAN.md` Status Notes. Skeleton:

> 2026-MM-DD: Slice 6 (MCP server) shipped on `main` over N phase-bundling commits. Plan: `docs/plans/2026-05-20-006-feat-mcp-server-plan.md`. New binary `cmd/falseflag-mcp` exposes 6 tools (`list_projects`, `list_flags`, `get_flag`, `validate_config`, `explain_evaluation`, `search_audit_log`) via Streamable HTTP on `:8091` and a `/healthz` on `:8092` using the official `github.com/modelcontextprotocol/go-sdk` v1.x. Reuses `internal/operator/clientapi` (now with `Evaluation` field) for Connect calls; X-Actor stamped as `mcp/falseflag-mcp`. Tests use `mcp.NewInMemoryTransports()` with fake Connect clients mirroring slice-4 operator pattern. New Hurl file `tests/hurl/12-mcp-tools.hurl` exercises initialize/tools-list/tools-call sequence. Compose stack now includes `mcp` service; bake default group adds `mcp`. Validation ladder passed: [list]. Known gaps deferred: [list]. Next recommended step is `Plan slice 7: Slow CI and Depot acceleration`.

## Sources & References

### Internal

- `docs/METAPLAN.md` — slice 6 prompt under "Recommended ce-plan Sequence" and Fan-Out D notes
- `docs/ideation/2026-05-20-synthetic-feature-flag-platform-depot-demo-ideation.md` — demo framing
- `docs/plans/2026-05-20-004-feat-operator-crds-plan.md` — operator pattern that MCP reuses
- `docs/plans/2026-05-20-005-feat-dashboard-cli-sdks-plan.md` — most recent shipped slice + phase-bundling pattern
- `cmd/falseflag-operator/main.go` — entrypoint template (16 lines)
- `internal/operator/clientapi/client.go` — Connect client + X-Actor interceptor; add `Evaluation` field
- `internal/operator/controllers/suite_test.go:65-104` — fake Connect client pattern for tests
- `internal/config/strategy.go:56` — `config.Compile` entry point for `validate_config`
- `internal/server/rpc/evaluation.go` — `EvaluateWithTrace` shape (for trace flattening)
- `internal/appconfig/appconfig.go` — `LoadOperator()` template for `LoadMCP()`
- `internal/buildinfo/buildinfo.go:40-62` — `WithGracefulShutdown` helper
- `internal/logging/slog.go:26` — `logging.New(suffix)` setup
- `infra/compose.yaml` — `*go-build` anchor and existing service definitions
- `infra/docker-bake.hcl` — existing `mcp` target, default group definition
- `tests/hurl/09-connect-smoke.hurl` — closest template for JSON-RPC-over-HTTP Hurl assertions
- `scripts/smoke.sh` and `scripts/kind-smoke.sh` — smoke script convention
- `AGENTS.md` — repo conventions; `<50-line main.go` rule, `log/slog` mandate, no `pkg/**`

### External

- [`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) — official SDK, v1.5.0 (2026-04-07)
- [pkg.go.dev: `go-sdk/mcp`](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp) — current API reference
- [MCP Tools spec, 2025-11-25 edition](https://modelcontextprotocol.io/specification/2025-11-25/server/tools) — `isError: true` content-block convention
- [MCP Streamable HTTP transport spec](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports) — transport choice
