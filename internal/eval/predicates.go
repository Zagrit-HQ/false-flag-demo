package eval

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/depot/falseflag/internal/config"
)

// match evaluates p against ctx using programs for any CEL leaves.
// Returns (matched, error). Errors propagate; a nil error with
// matched=false means "predicate did not fire" rather than a failure.
func match(p *config.Predicate, ctx map[string]any, programs map[string]config.CELProgram) (bool, error) {
	if p == nil {
		return false, nil
	}
	switch p.Kind {
	case config.PredAlways:
		return true, nil
	case config.PredEq:
		return cmpEq(p, ctx)
	case config.PredNeq:
		ok, err := cmpEq(p, ctx)
		return !ok, err
	case config.PredIn:
		return cmpIn(p, ctx)
	case config.PredGt, config.PredGte, config.PredLt, config.PredLte:
		return cmpOrd(p, ctx)
	case config.PredMatches:
		return cmpMatches(p, ctx)
	case config.PredRollout:
		v, ok := lookupString(ctx, p.Attr)
		if !ok {
			return false, nil
		}
		return inBucket(p.Salt, v, p.Percent), nil
	case config.PredAll:
		for _, c := range p.Of {
			ok, err := match(c, ctx, programs)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	case config.PredAny:
		for _, c := range p.Of {
			ok, err := match(c, ctx, programs)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	case config.PredNot:
		ok, err := match(p.OfOne, ctx, programs)
		return !ok, err
	case config.PredCEL:
		prog, ok := programs[p.Source]
		if !ok {
			return false, fmt.Errorf("cel program missing for source %q", p.Source)
		}
		return config.EvalCEL(prog, ctx)
	default:
		return false, fmt.Errorf("unknown predicate kind %q", p.Kind)
	}
}

func cmpEq(p *config.Predicate, ctx map[string]any) (bool, error) {
	actual, ok := lookup(ctx, p.Attr)
	if !ok {
		return false, nil
	}
	var expected any
	if err := json.Unmarshal(p.Value, &expected); err != nil {
		return false, fmt.Errorf("decoding eq value: %w", err)
	}
	return jsonEqual(actual, expected), nil
}

func cmpIn(p *config.Predicate, ctx map[string]any) (bool, error) {
	actual, ok := lookup(ctx, p.Attr)
	if !ok {
		return false, nil
	}
	for _, raw := range p.Values {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return false, fmt.Errorf("decoding in values: %w", err)
		}
		if jsonEqual(actual, v) {
			return true, nil
		}
	}
	return false, nil
}

func cmpOrd(p *config.Predicate, ctx map[string]any) (bool, error) {
	actualRaw, ok := lookup(ctx, p.Attr)
	if !ok {
		return false, nil
	}
	a, ok := toFloat(actualRaw)
	if !ok {
		return false, nil
	}
	var expRaw any
	if err := json.Unmarshal(p.Value, &expRaw); err != nil {
		return false, fmt.Errorf("decoding ord value: %w", err)
	}
	b, ok := toFloat(expRaw)
	if !ok {
		return false, nil
	}
	switch p.Kind {
	case config.PredGt:
		return a > b, nil
	case config.PredGte:
		return a >= b, nil
	case config.PredLt:
		return a < b, nil
	case config.PredLte:
		return a <= b, nil
	}
	return false, nil
}

func cmpMatches(p *config.Predicate, ctx map[string]any) (bool, error) {
	actual, ok := lookupString(ctx, p.Attr)
	if !ok {
		return false, nil
	}
	re, err := regexp.Compile(p.Pattern)
	if err != nil {
		return false, fmt.Errorf("compiling pattern: %w", err)
	}
	return re.MatchString(actual), nil
}

// lookup walks a dotted path ("user.country") through nested maps.
// Returns (value, true) on hit, (nil, false) on miss.
func lookup(ctx map[string]any, path string) (any, bool) {
	if path == "" {
		return nil, false
	}
	parts := strings.Split(path, ".")
	var cur any = ctx
	for _, part := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func lookupString(ctx map[string]any, path string) (string, bool) {
	v, ok := lookup(ctx, path)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	return s, true
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

// jsonEqual compares two values after JSON-style normalization: both
// must be the same kind (string, number, bool, null, map, slice) and
// numerically/structurally equal.
func jsonEqual(a, b any) bool {
	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case nil:
		return b == nil
	default:
		// Number-ish or structural — round-trip both through JSON for
		// safe comparison without writing a full deepEqual.
		ab, _ := json.Marshal(a)
		bb, _ := json.Marshal(b)
		return string(ab) == string(bb)
	}
}
