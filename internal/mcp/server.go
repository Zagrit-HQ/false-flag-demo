package mcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/depot/falseflag/internal/buildinfo"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

// newServer builds an MCP Server with the FalseFlag implementation
// metadata. Tool registration is intentionally a separate step
// (RegisterTools) so test suites can register only what they need.
func newServer() *mcp.Server {
	return mcp.NewServer(&mcp.Implementation{
		Name:    "falseflag-mcp",
		Version: buildinfo.Version,
	}, nil)
}

// RegisterTools wires every production tool onto s using client for
// upstream calls. Wraps a single registration site so the canonical
// tool list lives in one place (tools.go).
func RegisterTools(s *mcp.Server, client *clientapi.Client) {
	registerTools(s, client)
}
