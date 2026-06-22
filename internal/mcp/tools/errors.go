// Package tools holds the MCP tool implementations registered onto
// the falseflag-mcp server. Each file in this package owns one or
// more tools plus their input/output schemas. The shared error
// helper here keeps all Connect-to-MCP error translation consistent.
package tools

import (
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolError builds a tool-level error result. Per MCP spec, tool
// failures must round-trip to the LLM as a successful JSON-RPC
// response with IsError:true and a human-readable Content block — a
// protocol-level error would be hidden from the model.
func toolError(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}

// connectErrToToolResult maps a Connect client error into a
// tool-level error result with a user-facing message. Unrecognized
// codes fall through to a generic "upstream error" body so callers
// always see structured output, never a raw RPC code.
func connectErrToToolResult(err error) *mcp.CallToolResult {
	if err == nil {
		return nil
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		return toolError(fmt.Sprintf("upstream error: %s", err.Error()))
	}
	switch connectErr.Code() {
	case connect.CodeNotFound:
		return toolError("not found: " + connectErr.Message())
	case connect.CodeInvalidArgument:
		return toolError("invalid argument: " + connectErr.Message())
	case connect.CodePermissionDenied:
		return toolError("permission denied: " + connectErr.Message())
	case connect.CodeUnavailable:
		return toolError("upstream unavailable: " + connectErr.Message())
	case connect.CodeUnauthenticated:
		return toolError("unauthenticated: " + connectErr.Message())
	default:
		return toolError(fmt.Sprintf("upstream error (%s): %s", connectErr.Code(), connectErr.Message()))
	}
}
