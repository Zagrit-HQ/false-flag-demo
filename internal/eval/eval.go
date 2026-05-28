package eval

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/depot/falseflag/internal/config"
)

// Reason enumerates the OpenFeature-shaped resolution reasons. Keep
// the string values in sync with js/packages/sdk-js/src/evaluator.ts.
const (
	ReasonDefault            = "default"
	ReasonRuleMatched        = "rule_matched"
	ReasonRolloutInBucket    = "rollout_in_bucket"
	ReasonRolloutOutOfBucket = "rollout_out_of_bucket"
	ReasonTypeMismatch       = "type_mismatch"
	ReasonError              = "error"
)

// Decision is the result of evaluating a flag.
type Decision struct {
	Value   any    `json:"value"`
	Reason  string `json:"reason"`
	RuleID  string `json:"rule_id,omitempty"`
	Version int    `json:"version"`
}

// Evaluate walks the compiled IR and returns a Decision. Errors only
// propagate for genuine internal failures (missing CEL program, etc.).
// Soft mismatches (missing attribute, type coercion failure) produce a
// Decision with Reason=type_mismatch and Value=default.
//
// The optional `version` is stored on the Decision so callers can
// audit which flag_versions row produced it; the evaluator itself
// doesn't load anything.
func Evaluate(c *config.Compiled, ctx map[string]any, version int) (Decision, error) {
	if c == nil || c.IR == nil {
		return Decision{}, errors.New("eval: nil compiled or IR")
	}
	defValue, err := decodeValue(c.IR.Default)
	if err != nil {
		return Decision{}, fmt.Errorf("decoding default: %w", err)
	}
	for i := range c.IR.Rules {
		r := &c.IR.Rules[i]
		ok, err := match(r.When, ctx, c.CELPrograms)
		if err != nil {
			return Decision{
				Value:   defValue,
				Reason:  ReasonError,
				RuleID:  r.ID,
				Version: version,
			}, nil
		}
		if !ok {
			continue
		}
		v, err := decodeValue(r.Value)
		if err != nil {
			return Decision{
				Value:   defValue,
				Reason:  ReasonTypeMismatch,
				RuleID:  r.ID,
				Version: version,
			}, nil
		}
		if !valueMatchesType(v, c.IR.ValueType) {
			return Decision{
				Value:   defValue,
				Reason:  ReasonTypeMismatch,
				RuleID:  r.ID,
				Version: version,
			}, nil
		}
		return Decision{
			Value:   v,
			Reason:  reasonForRule(r),
			RuleID:  r.ID,
			Version: version,
		}, nil
	}
	return Decision{
		Value:   defValue,
		Reason:  ReasonDefault,
		Version: version,
	}, nil
}

func decodeValue(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return v, nil
}

func valueMatchesType(v any, t config.ValueType) bool {
	switch t {
	case config.ValueTypeBoolean:
		_, ok := v.(bool)
		return ok
	case config.ValueTypeString:
		_, ok := v.(string)
		return ok
	case config.ValueTypeNumber:
		_, ok := toFloat(v)
		return ok
	case config.ValueTypeObject:
		_, ok := v.(map[string]any)
		return ok
	}
	return false
}

func reasonForRule(r *config.Rule) string {
	if r.When != nil && r.When.Kind == config.PredRollout {
		return ReasonRolloutInBucket
	}
	return ReasonRuleMatched
}
