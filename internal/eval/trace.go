package eval

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/depot/falseflag/internal/config"
)

// Trace is the explained-evaluation view returned by EvaluateWithTrace.
type Trace struct {
	EvaluatedRules []TraceRule `json:"evaluated_rules"`
	DefaultUsed    bool        `json:"default_used"`
	MatchedRuleID  string      `json:"matched_rule_id,omitempty"`
}

// TraceRule captures one rule's outcome plus its evaluated predicate
// tree. Rules later than the first match are still recorded, so the
// demo can show why they didn't fire.
type TraceRule struct {
	RuleID    string    `json:"rule_id"`
	Matched   bool      `json:"matched"`
	Predicate TraceNode `json:"predicate"`
	Error     string    `json:"error,omitempty"`
}

// TraceNode mirrors config.Predicate but with the additional fields the
// demo needs to explain the outcome.
type TraceNode struct {
	Kind           string      `json:"kind"`
	Attr           string      `json:"attr,omitempty"`
	AttrValue      any         `json:"attr_value,omitempty"`
	Expected       any         `json:"expected,omitempty"`
	ExpectedValues []any       `json:"expected_values,omitempty"`
	Pattern        string      `json:"pattern,omitempty"`
	Salt           string      `json:"salt,omitempty"`
	Percent        int         `json:"percent,omitempty"`
	Bucket         int         `json:"bucket,omitempty"`
	Source         string      `json:"source,omitempty"`
	Result         bool        `json:"result"`
	Children       []TraceNode `json:"children,omitempty"`
}

// EvaluateWithTrace returns the same Decision as Evaluate plus a trace
// recording every rule and the predicate evaluation for that rule. All
// rules are evaluated, even after a match, so the trace explains why
// later rules didn't apply too.
func EvaluateWithTrace(c *config.Compiled, ctx map[string]any, version int) (Decision, Trace, error) {
	if c == nil || c.IR == nil {
		return Decision{}, Trace{}, errors.New("eval: nil compiled or IR")
	}
	defValue, err := decodeValue(c.IR.Default)
	if err != nil {
		return Decision{}, Trace{}, fmt.Errorf("decoding default: %w", err)
	}

	trace := Trace{EvaluatedRules: make([]TraceRule, 0, len(c.IR.Rules))}
	var winner *config.Rule
	winnerIdx := -1
	for i := range c.IR.Rules {
		r := &c.IR.Rules[i]
		node, ok, err := traceMatch(r.When, ctx, c.CELPrograms)
		tr := TraceRule{
			RuleID:    r.ID,
			Matched:   ok && err == nil,
			Predicate: node,
		}
		if err != nil {
			tr.Matched = false
			tr.Error = err.Error()
		}
		trace.EvaluatedRules = append(trace.EvaluatedRules, tr)
		if winner == nil && tr.Matched {
			winner = r
			winnerIdx = i
		}
	}

	if winner == nil {
		trace.DefaultUsed = true
		return Decision{
			Value:   defValue,
			Reason:  ReasonDefault,
			Version: version,
		}, trace, nil
	}
	trace.MatchedRuleID = winner.ID

	v, err := decodeValue(winner.Value)
	if err != nil || !valueMatchesType(v, c.IR.ValueType) {
		return Decision{
			Value:   defValue,
			Reason:  ReasonTypeMismatch,
			RuleID:  winner.ID,
			Version: version,
		}, trace, nil
	}
	_ = winnerIdx // (kept for future-trace position hints)
	return Decision{
		Value:   v,
		Reason:  reasonForRule(winner),
		RuleID:  winner.ID,
		Version: version,
	}, trace, nil
}

// traceMatch is the trace-recording counterpart of match. It returns
// the trace node along with the boolean outcome. The two are kept
// separate so the hot-path Evaluate stays simple.
func traceMatch(p *config.Predicate, ctx map[string]any, programs map[string]config.CELProgram) (TraceNode, bool, error) {
	if p == nil {
		return TraceNode{Kind: "nil"}, false, nil
	}
	node := TraceNode{Kind: string(p.Kind)}
	switch p.Kind {
	case config.PredAlways:
		node.Result = true
		return node, true, nil
	case config.PredEq, config.PredNeq:
		node.Attr = p.Attr
		actual, _ := lookup(ctx, p.Attr)
		node.AttrValue = actual
		var expected any
		_ = json.Unmarshal(p.Value, &expected)
		node.Expected = expected
		ok, err := match(p, ctx, programs)
		node.Result = ok
		return node, ok, err
	case config.PredIn:
		node.Attr = p.Attr
		actual, _ := lookup(ctx, p.Attr)
		node.AttrValue = actual
		expected := make([]any, 0, len(p.Values))
		for _, raw := range p.Values {
			var v any
			_ = json.Unmarshal(raw, &v)
			expected = append(expected, v)
		}
		node.ExpectedValues = expected
		ok, err := match(p, ctx, programs)
		node.Result = ok
		return node, ok, err
	case config.PredGt, config.PredGte, config.PredLt, config.PredLte:
		node.Attr = p.Attr
		actual, _ := lookup(ctx, p.Attr)
		node.AttrValue = actual
		var expected any
		_ = json.Unmarshal(p.Value, &expected)
		node.Expected = expected
		ok, err := match(p, ctx, programs)
		node.Result = ok
		return node, ok, err
	case config.PredMatches:
		node.Attr = p.Attr
		actual, _ := lookupString(ctx, p.Attr)
		node.AttrValue = actual
		node.Pattern = p.Pattern
		ok, err := match(p, ctx, programs)
		node.Result = ok
		return node, ok, err
	case config.PredRollout:
		node.Attr = p.Attr
		node.Salt = p.Salt
		node.Percent = p.Percent
		actual, found := lookupString(ctx, p.Attr)
		node.AttrValue = actual
		if found {
			node.Bucket = rolloutBucket(p.Salt, actual)
		}
		ok, err := match(p, ctx, programs)
		node.Result = ok
		return node, ok, err
	case config.PredAll:
		all := true
		var firstErr error
		for _, c := range p.Of {
			child, ok, err := traceMatch(c, ctx, programs)
			node.Children = append(node.Children, child)
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if !ok {
				all = false
			}
		}
		node.Result = all && firstErr == nil
		return node, node.Result, firstErr
	case config.PredAny:
		any := false
		var firstErr error
		for _, c := range p.Of {
			child, ok, err := traceMatch(c, ctx, programs)
			node.Children = append(node.Children, child)
			if err != nil && firstErr == nil {
				firstErr = err
			}
			if ok {
				any = true
			}
		}
		node.Result = any && firstErr == nil
		return node, node.Result, firstErr
	case config.PredNot:
		child, ok, err := traceMatch(p.OfOne, ctx, programs)
		node.Children = []TraceNode{child}
		node.Result = !ok && err == nil
		return node, node.Result, err
	case config.PredCEL:
		node.Source = p.Source
		ok, err := match(p, ctx, programs)
		node.Result = ok
		return node, ok, err
	}
	return node, false, fmt.Errorf("unknown predicate kind %q", p.Kind)
}
