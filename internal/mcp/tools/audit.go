package tools

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

// SearchAuditLogInput is the input for search_audit_log. ProjectSlug
// is required because the upstream RPC scopes all audit lookups by
// project. Time filters use RFC3339 strings to avoid timezone
// ambiguity from the LLM side; they're parsed and normalized to
// timestamppb.Timestamp before the upstream call.
type SearchAuditLogInput struct {
	ProjectSlug string `json:"project_slug" jsonschema:"slug of the project to search"`
	Action      string `json:"action,omitempty" jsonschema:"action filter, e.g. publish_version, create_flag"`
	Actor       string `json:"actor,omitempty" jsonschema:"actor filter; the MCP server itself is mcp/falseflag-mcp"`
	From        string `json:"from,omitempty" jsonschema:"RFC3339 lower bound, inclusive"`
	To          string `json:"to,omitempty" jsonschema:"RFC3339 upper bound, exclusive"`
	Limit       int32  `json:"limit,omitempty" jsonschema:"max events to return (1-200, default 50)"`
	Cursor      string `json:"cursor,omitempty" jsonschema:"opaque pagination cursor from a prior response"`
}

func searchAuditLog(client *clientapi.Client) mcp.ToolHandlerFor[SearchAuditLogInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in SearchAuditLogInput) (*mcp.CallToolResult, any, error) {
		if client == nil || client.Audit == nil {
			return toolError("audit client unavailable"), nil, nil
		}
		req := &pb.ListAuditEventsRequest{
			ProjectSlug: in.ProjectSlug,
			Limit:       normalizeLimit(in.Limit),
		}
		if in.Action != "" {
			a := in.Action
			req.Action = &a
		}
		if in.Actor != "" {
			a := in.Actor
			req.Actor = &a
		}
		if in.From != "" {
			ts, err := parseRFC3339(in.From)
			if err != nil {
				return toolError("invalid from timestamp: " + err.Error()), nil, nil
			}
			req.From = ts
		}
		if in.To != "" {
			ts, err := parseRFC3339(in.To)
			if err != nil {
				return toolError("invalid to timestamp: " + err.Error()), nil, nil
			}
			req.To = ts
		}
		if in.Cursor != "" {
			c := in.Cursor
			req.Cursor = &c
		}

		resp, err := client.Audit.ListAuditEvents(ctx, connect.NewRequest(req))
		if err != nil {
			return connectErrToToolResult(err), nil, nil
		}
		return marshalProto(resp.Msg, "audit-events")
	}
}

// normalizeLimit clamps to [1, 200], defaulting unset/zero to 50.
func normalizeLimit(n int32) int32 {
	if n <= 0 {
		return 50
	}
	if n > 200 {
		return 200
	}
	return n
}

func parseRFC3339(s string) (*timestamppb.Timestamp, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return timestamppb.New(t), nil
}
