# mcp-tools.hurl — Streamable HTTP MCP surface on :8091. The server
# enforces the spec's initialize-then-call lifecycle: every session
# must do `initialize` first, then `notifications/initialized`, and
# subsequent calls must carry the `Mcp-Session-Id` header that the
# server returned. We capture it and reuse it through the file.

# 1. Initialize. Captures the session id for the rest of the file.
POST {{mcp_base_url}}/
Content-Type: application/json
Accept: application/json, text/event-stream
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-06-18",
    "capabilities": {},
    "clientInfo": {"name": "hurl-smoke", "version": "0.0.1"}
  }
}
HTTP 200
[Captures]
session_id: header "Mcp-Session-Id"
[Asserts]
header "Content-Type" contains "text/event-stream"
body contains "\"protocolVersion\":\"2025-06-18\""
body contains "\"name\":\"falseflag-mcp\""

# 2. The spec requires the client to signal it has finished
# initialization before issuing any other calls.
POST {{mcp_base_url}}/
Content-Type: application/json
Accept: application/json, text/event-stream
Mcp-Session-Id: {{session_id}}
{
  "jsonrpc": "2.0",
  "method": "notifications/initialized"
}
HTTP 202

# 3. tools/list. Every registered tool must appear; failure here
# means tool registration drift between code and the canonical list.
POST {{mcp_base_url}}/
Content-Type: application/json
Accept: application/json, text/event-stream
Mcp-Session-Id: {{session_id}}
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}
HTTP 200
[Asserts]
body contains "list_projects"
body contains "list_flags"
body contains "get_flag"
body contains "validate_config"
body contains "explain_evaluation"
body contains "search_audit_log"

# 4. tools/call list_projects. Must succeed and return at least one
# project. The body is double-encoded (JSON-RPC result contains a
# text field which is itself JSON-encoded), so we assert on
# unambiguous slug tokens. `hurl-projects-a` is created by
# 01-projects.hurl which runs first under make smoke; on a stack
# that ran `make seed` instead, `acme-web` will appear.
POST {{mcp_base_url}}/
Content-Type: application/json
Accept: application/json, text/event-stream
Mcp-Session-Id: {{session_id}}
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "list_projects",
    "arguments": {}
  }
}
HTTP 200
[Asserts]
body matches "hurl-projects-a|acme-web"

# 5. tools/call validate_config with a malformed source. The tool
# returns a successful response with `valid: false` — distinct from
# IsError, which signals a tool-level failure (e.g. unknown
# strategy). This is the spec's application-vs-protocol distinction.
POST {{mcp_base_url}}/
Content-Type: application/json
Accept: application/json, text/event-stream
Mcp-Session-Id: {{session_id}}
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "validate_config",
    "arguments": {"strategy": "json", "source": "this is not JSON"}
  }
}
HTTP 200
[Asserts]
# Tool output is JSON nested inside the MCP `text` field, so quotes
# around keys are backslash-escaped. We match the raw substring.
body contains "valid\\\":false"
body contains "invalid"
