// Command falseflag-mcp is the FalseFlag agent-facing MCP server.
// It exposes read-and-inspect tools for the control-plane API over
// Streamable HTTP, plus a /healthz listener on a separate port for
// compose orchestration.
//
// All logic lives in internal/mcp; main stays a thin entrypoint per
// the AGENTS.md <50-line rule.
package main

import (
	"os"

	"github.com/depot/falseflag/internal/buildinfo"
	"github.com/depot/falseflag/internal/mcp"
)

func main() {
	os.Exit(buildinfo.WithGracefulShutdown("mcp", mcp.Run))
}
