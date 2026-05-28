package tools

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/depot/falseflag/internal/config"
)

// ValidateConfigInput is the input for validate_config. Strategy is
// enum-constrained at the SDK schema layer; Source is the raw config
// blob in the strategy's source format.
type ValidateConfigInput struct {
	Strategy string `json:"strategy" jsonschema:"one of json, cel, typescript"`
	Source   string `json:"source" jsonschema:"raw config source in the strategy's format"`
}

// ValidateConfigOutput is the hand-shaped result. Valid + Errors are
// the application-level signal; IRSummary is present on success and
// surfaces the headline shape of the compiled rules tree so an agent
// can sanity-check intent.
type ValidateConfigOutput struct {
	Valid     bool                   `json:"valid"`
	Strategy  string                 `json:"strategy"`
	Errors    []string               `json:"errors,omitempty"`
	IRSummary *ValidateConfigSummary `json:"ir_summary,omitempty"`
}

// ValidateConfigSummary surfaces a few headline facts about the
// compiled IR. flag_count is dropped: validate_config compiles one
// flag at a time, so reporting "1" everywhere would be noise.
type ValidateConfigSummary struct {
	ValueType   string `json:"value_type"`
	RuleCount   int    `json:"rule_count"`
	HasRollout  bool   `json:"has_rollout"`
	HasCEL      bool   `json:"has_cel"`
	CELPrograms int    `json:"cel_program_count"`
}

func validateConfig() mcp.ToolHandlerFor[ValidateConfigInput, any] {
	return func(_ context.Context, _ *mcp.CallToolRequest, in ValidateConfigInput) (*mcp.CallToolResult, any, error) {
		strategy := config.Strategy(in.Strategy)
		if !strategy.Valid() {
			return toolError("unknown strategy: " + in.Strategy + " (valid: json, cel, typescript)"), nil, nil
		}
		if in.Source == "" {
			return jsonResult(ValidateConfigOutput{Valid: false, Strategy: in.Strategy, Errors: []string{"empty source"}})
		}
		compiled, err := config.Compile(strategy, []byte(in.Source))
		if err != nil {
			return jsonResult(ValidateConfigOutput{Valid: false, Strategy: in.Strategy, Errors: []string{err.Error()}})
		}
		summary := summarize(compiled)
		return jsonResult(ValidateConfigOutput{Valid: true, Strategy: in.Strategy, IRSummary: summary})
	}
}

// summarize walks compiled.IR shallowly to produce the IR summary.
func summarize(c *config.Compiled) *ValidateConfigSummary {
	if c == nil {
		return &ValidateConfigSummary{}
	}
	s := &ValidateConfigSummary{
		CELPrograms: len(c.CELPrograms),
	}
	s.HasCEL = s.CELPrograms > 0
	if c.IR != nil {
		s.ValueType = string(c.IR.ValueType)
		s.RuleCount = len(c.IR.Rules)
		for _, r := range c.IR.Rules {
			if predicateHasKind(r.When, config.PredRollout) {
				s.HasRollout = true
				break
			}
		}
	}
	return s
}

// predicateHasKind walks p (and its children) looking for kind.
func predicateHasKind(p *config.Predicate, kind config.PredicateKind) bool {
	if p == nil {
		return false
	}
	if p.Kind == kind {
		return true
	}
	for _, child := range p.Of {
		if predicateHasKind(child, kind) {
			return true
		}
	}
	return predicateHasKind(p.OfOne, kind)
}

// jsonResult marshals v into a single TextContent block.
func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return toolError("failed to marshal result: " + err.Error()), nil, nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(body)}}}, nil, nil
}
