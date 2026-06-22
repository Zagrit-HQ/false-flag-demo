package tools

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/encoding/protojson"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

// ListFlagsInput is the input for list_flags. ProjectSlug is required
// (no omitempty) so the SDK rejects empty-input calls at the schema layer.
type ListFlagsInput struct {
	ProjectSlug string `json:"project_slug" jsonschema:"slug of the project whose flags to list"`
}

// GetFlagInput is the input for get_flag.
type GetFlagInput struct {
	ProjectSlug string `json:"project_slug" jsonschema:"slug of the project that owns the flag"`
	FlagKey     string `json:"flag_key" jsonschema:"key of the flag within the project"`
}

func listFlags(client *clientapi.Client) mcp.ToolHandlerFor[ListFlagsInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ListFlagsInput) (*mcp.CallToolResult, any, error) {
		if client == nil || client.Flags == nil {
			return toolError("flags client unavailable"), nil, nil
		}
		resp, err := client.Flags.ListFlags(ctx, connect.NewRequest(&pb.ListFlagsRequest{ProjectSlug: in.ProjectSlug}))
		if err != nil {
			return connectErrToToolResult(err), nil, nil
		}
		return marshalProto(resp.Msg, "flags")
	}
}

// getFlag returns flag metadata plus the latest version. When the
// flag exists but has never been published, the LatestVersion field
// is naturally omitted by the upstream API and we surface a friendly
// note alongside the data so an LLM can interpret it correctly.
func getFlag(client *clientapi.Client) mcp.ToolHandlerFor[GetFlagInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in GetFlagInput) (*mcp.CallToolResult, any, error) {
		if client == nil || client.Flags == nil {
			return toolError("flags client unavailable"), nil, nil
		}
		resp, err := client.Flags.GetFlag(ctx, connect.NewRequest(&pb.GetFlagRequest{
			ProjectSlug: in.ProjectSlug,
			Key:         in.FlagKey,
		}))
		if err != nil {
			return connectErrToToolResult(err), nil, nil
		}

		// Hand-shaped output: pair the protojson body with an
		// explicit `published_version` discriminator so an LLM
		// doesn't have to infer "no version exists" from an absent
		// field. The flag payload itself is passed through as raw
		// protojson — no DTO drift.
		flagJSON, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp.Msg.GetFlag())
		if err != nil {
			return toolError("failed to marshal flag: " + err.Error()), nil, nil
		}
		out := map[string]any{
			"flag":              json.RawMessage(flagJSON),
			"published_version": nil,
			"note":              "flag has never been published",
		}
		if v := resp.Msg.GetLatestVersion(); v != nil {
			versionJSON, vErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(v)
			if vErr != nil {
				return toolError("failed to marshal flag version: " + vErr.Error()), nil, nil
			}
			out["published_version"] = json.RawMessage(versionJSON)
			delete(out, "note")
		}
		body, err := json.Marshal(out)
		if err != nil {
			return toolError("failed to marshal get_flag response: " + err.Error()), nil, nil
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}, nil, nil
	}
}
