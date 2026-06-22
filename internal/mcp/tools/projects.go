package tools

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

// ListProjectsInput has no fields — list_projects is parameterless.
// The empty struct still gives the SDK a valid object schema.
type ListProjectsInput struct{}

func listProjects(client *clientapi.Client) mcp.ToolHandlerFor[ListProjectsInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ ListProjectsInput) (*mcp.CallToolResult, any, error) {
		if client == nil || client.Projects == nil {
			return toolError("projects client unavailable"), nil, nil
		}
		resp, err := client.Projects.ListProjects(ctx, connect.NewRequest(&pb.ListProjectsRequest{}))
		if err != nil {
			return connectErrToToolResult(err), nil, nil
		}
		return marshalProto(resp.Msg, "projects")
	}
}

// marshalProto serializes msg with protojson and wraps it as a single
// TextContent block. proto field names are already snake_case which
// is LLM-friendly, so passthrough is preferred over hand DTOs.
func marshalProto(msg interface{ Reset() }, kind string) (*mcp.CallToolResult, any, error) {
	m, ok := msg.(protoMessage)
	if !ok {
		return toolError(fmt.Sprintf("internal: %s response is not a proto message", kind)), nil, nil
	}
	body, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(m)
	if err != nil {
		return toolError(fmt.Sprintf("failed to marshal %s response: %v", kind, err)), nil, nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}, nil, nil
}
