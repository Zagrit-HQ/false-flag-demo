package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/depot/falseflag/internal/mcp/tools"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

// ToolNames is the canonical list of tools the MCP server registers.
// Keep in sync with tools.Register.
var ToolNames = []string{
	"list_projects",
	"list_flags",
	"get_flag",
	"validate_config",
	"explain_evaluation",
	"search_audit_log",
}

// registerTools is the single registration site, delegating to the
// tools subpackage. Phases 3-5 fill in the production tool set; the
// canonical list above will match by the end of phase 5.
func registerTools(s *mcp.Server, client *clientapi.Client) {
	tools.Register(s, client)
}
