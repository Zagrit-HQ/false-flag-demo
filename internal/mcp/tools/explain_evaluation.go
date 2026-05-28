package tools

import (
	"context"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

// ExplainEvaluationInput is the input for explain_evaluation.
// Context is a free-form attribute bag that mirrors what the SDK
// passes to Evaluate — keys like "user", "request", "environment"
// each mapping to nested values.
type ExplainEvaluationInput struct {
	ProjectSlug string         `json:"project_slug" jsonschema:"slug of the project that owns the flag"`
	FlagKey     string         `json:"flag_key" jsonschema:"key of the flag to evaluate"`
	Context     map[string]any `json:"context,omitempty" jsonschema:"evaluation context attributes (e.g. user, request, environment)"`
}

func explainEvaluation(client *clientapi.Client) mcp.ToolHandlerFor[ExplainEvaluationInput, any] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, in ExplainEvaluationInput) (*mcp.CallToolResult, any, error) {
		if client == nil || client.Evaluation == nil {
			return toolError("evaluation client unavailable"), nil, nil
		}
		var ctxStruct *structpb.Struct
		if len(in.Context) > 0 {
			s, err := structpb.NewStruct(in.Context)
			if err != nil {
				return toolError("invalid context: " + err.Error()), nil, nil
			}
			ctxStruct = s
		}
		resp, err := client.Evaluation.EvaluateWithTrace(ctx, connect.NewRequest(&pb.EvaluateWithTraceRequest{
			ProjectSlug: in.ProjectSlug,
			Key:         in.FlagKey,
			Context:     ctxStruct,
		}))
		if err != nil {
			return connectErrToToolResult(err), nil, nil
		}
		// Passthrough: the proto trace is already structured
		// (rule_id, matched, predicate tree, error per rule) and
		// the LLM can consume protojson directly. Hand-flattening
		// adds maintenance surface without improving readability
		// for the demo. Documented deviation from the plan.
		return marshalProto(resp.Msg, "evaluate-with-trace")
	}
}
