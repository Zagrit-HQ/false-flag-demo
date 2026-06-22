# falseflag-mcp

Agent-facing MCP (Model Context Protocol) server for the FalseFlag
demo. Built on the official
[`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk)
(pinned at v1.6.0) and serves a Streamable HTTP listener on `:8091`
plus a `/healthz` probe on `:8092`.

## Tools

| Tool | Input | Output |
|---|---|---|
| `list_projects` | none | All FalseFlag projects (protojson passthrough) |
| `list_flags` | `project_slug` | Flags in that project |
| `get_flag` | `project_slug`, `flag_key` | Flag metadata + `published_version` (null + a note when the flag has never been published) |
| `validate_config` | `strategy` (`json`/`cel`/`typescript`), `source` | `{valid, errors?, ir_summary?}` — compiles in-process via `internal/config.Compile`, no API round-trip |
| `explain_evaluation` | `project_slug`, `flag_key`, `context` (object) | Decision + per-rule evaluation trace |
| `search_audit_log` | `project_slug`, optional `action`/`actor`/`from`/`to`/`limit`/`cursor` | Paginated audit events |

Every tool maps Connect-level errors (`not_found`, `invalid_argument`,
`unavailable`, …) to MCP tool-level errors (`isError: true` with a
TextContent body). This matches the MCP spec: tool failures must round
trip to the LLM so it can self-correct, rather than being swallowed as
protocol errors.

## Configuration

| Env var | Default | Purpose |
|---|---|---|
| `FALSEFLAG_MCP_ADDR` | `:8091` | Streamable HTTP listener for MCP traffic |
| `FALSEFLAG_MCP_HEALTH_ADDR` | `:8092` | `/healthz` liveness probe |
| `FALSEFLAG_API_RPC_ADDR` | `http://localhost:8090` | Upstream FalseFlag Connect RPC endpoint |
| `FALSEFLAG_MCP_ACTOR` | `mcp/falseflag-mcp` | `X-Actor` header on every outbound API request (demo-only attribution) |
| `LOG_LEVEL` | `info` | Standard `log/slog` level |

## Running locally

The compose stack includes the `mcp` service:

```bash
make up        # builds + starts api, proxy, operator, mcp, dashboard
make seed      # populates 3 projects + 7 flags
```

To run the binary directly without compose:

```bash
FALSEFLAG_API_RPC_ADDR=http://localhost:8090 go run ./cmd/falseflag-mcp
```

## Pointing Claude at this server

```bash
claude mcp add --transport http falseflag http://localhost:8091/
claude mcp call falseflag list_projects
```

The `--transport http` flag is required: stdio transport would need
Claude to spawn the binary as a subprocess, which doesn't compose with
a long-lived service in docker.

## Audit attribution

Because the MCP server stamps `X-Actor: mcp/falseflag-mcp` (or whatever
`FALSEFLAG_MCP_ACTOR` is set to) on every upstream call, any mutation
an agent makes through a future MCP tool will appear in
`audit_events.actor` with that value. The current six tools are all
read-only, so today the only place to observe this is by running the
`search_audit_log` tool with `actor: "mcp/falseflag-mcp"` after a
mutating tool is added in a later slice.

## Non-goals

This slice is intentionally demo-quality. Out of scope:

- bearer-token or OAuth auth on the MCP listener
- mutation tools (`create_project`, `publish_flag_version`, …) — the
  Connect API has them, the MCP just doesn't surface them yet
- MCP Resources and Prompts (only Tools are exposed)
- streaming long-running tool responses (none of the six are
  long-running; they're synchronous Connect calls)
