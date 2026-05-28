package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/depot/falseflag/internal/operator/clientapi"
)

// Register attaches every production MCP tool to s. The tool list is
// canonical here; mcp.ToolNames mirrors this set for use by smoke
// tests and docs.
func Register(s *mcp.Server, client *clientapi.Client) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_projects",
		Description: "List every FalseFlag project. No inputs.",
	}, listProjects(client))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_flags",
		Description: "List the feature flags owned by a project. Input: project_slug.",
	}, listFlags(client))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_flag",
		Description: "Fetch a flag's metadata and its latest published version. Input: project_slug, flag_key. When the flag has never been published, published_version is null and a 'note' field explains.",
	}, getFlag(client))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "validate_config",
		Description: "Compile a FalseFlag config blob (json, cel, or typescript) in-process and report whether it's valid. On success, ir_summary describes the compiled rules tree (value_type, rule_count, has_rollout, cel_program_count).",
	}, validateConfig())

	mcp.AddTool(s, &mcp.Tool{
		Name:        "explain_evaluation",
		Description: "Evaluate a flag for a context and return the decision plus a per-rule trace showing which predicates matched. Input: project_slug, flag_key, context (free-form attribute bag).",
	}, explainEvaluation(client))

	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_audit_log",
		Description: "Search the audit log for a project. Filters: action, actor, from/to (RFC3339), limit (1-200, default 50), cursor. Note: the MCP server itself records actions with actor 'mcp/falseflag-mcp' so an agent can observe its own prior actions.",
	}, searchAuditLog(client))
}
