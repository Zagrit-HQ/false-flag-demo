package config

import (
	"encoding/json"
	"fmt"
	"regexp"
)

type jsonCompiler struct{}

func (jsonCompiler) Strategy() Strategy { return StrategyJSON }

// Compile parses source as a RulesTree and validates it structurally.
// JSON is the "no-op" strategy — the source IS the IR — but we still
// walk the tree to reject malformed predicates and unsupported value
// types.
func (jsonCompiler) Compile(source []byte) (*Compiled, error) {
	var tree RulesTree
	if err := json.Unmarshal(source, &tree); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidIR, err)
	}
	if err := validateTree(&tree); err != nil {
		return nil, err
	}
	return &Compiled{Strategy: StrategyJSON, IR: &tree, CELPrograms: nil}, nil
}

// validateTree enforces the IR invariants:
//   - value_type is one of the known kinds
//   - default is present (json.RawMessage non-nil)
//   - every predicate is well-formed
//
// CEL sources are NOT validated here — that's the cel compiler's job.
// JSON-strategy flags may not contain CEL predicates; we enforce that.
func validateTree(tree *RulesTree) error {
	switch tree.ValueType {
	case ValueTypeBoolean, ValueTypeString, ValueTypeNumber, ValueTypeObject:
	default:
		return fmt.Errorf("%w: %q", ErrInvalidValueType, tree.ValueType)
	}
	if len(tree.Default) == 0 {
		return fmt.Errorf("%w: missing default", ErrInvalidIR)
	}
	for i := range tree.Rules {
		r := &tree.Rules[i]
		if r.ID == "" {
			return fmt.Errorf("%w: rule %d missing id", ErrInvalidIR, i)
		}
		if len(r.Value) == 0 {
			return fmt.Errorf("%w: rule %q missing value", ErrInvalidIR, r.ID)
		}
		if r.When == nil {
			return fmt.Errorf("%w: rule %q missing when", ErrInvalidIR, r.ID)
		}
		if err := validatePredicate(r.When, false); err != nil {
			return fmt.Errorf("rule %q: %w", r.ID, err)
		}
	}
	return nil
}

func validatePredicate(p *Predicate, allowCEL bool) error {
	if p == nil {
		return fmt.Errorf("%w: nil predicate", ErrInvalidPredicate)
	}
	switch p.Kind {
	case PredEq, PredNeq, PredGt, PredGte, PredLt, PredLte:
		if p.Attr == "" {
			return fmt.Errorf("%w: %s missing attr", ErrInvalidPredicate, p.Kind)
		}
		if len(p.Value) == 0 {
			return fmt.Errorf("%w: %s missing value", ErrInvalidPredicate, p.Kind)
		}
	case PredIn:
		if p.Attr == "" {
			return fmt.Errorf("%w: in missing attr", ErrInvalidPredicate)
		}
		if len(p.Values) == 0 {
			return fmt.Errorf("%w: in needs non-empty values", ErrInvalidPredicate)
		}
	case PredMatches:
		if p.Attr == "" || p.Pattern == "" {
			return fmt.Errorf("%w: matches missing attr or pattern", ErrInvalidPredicate)
		}
		if _, err := regexp.Compile(p.Pattern); err != nil {
			return fmt.Errorf("%w: matches pattern: %s", ErrInvalidPredicate, err)
		}
	case PredRollout:
		if p.Attr == "" || p.Salt == "" {
			return fmt.Errorf("%w: rollout missing attr or salt", ErrInvalidPredicate)
		}
		if p.Percent < 0 || p.Percent > 100 {
			return fmt.Errorf("%w: rollout percent out of [0,100]", ErrInvalidPredicate)
		}
	case PredAll, PredAny:
		if len(p.Of) == 0 {
			return fmt.Errorf("%w: %s requires of[]", ErrInvalidPredicate, p.Kind)
		}
		for _, c := range p.Of {
			if err := validatePredicate(c, allowCEL); err != nil {
				return err
			}
		}
	case PredNot:
		if p.OfOne == nil {
			return fmt.Errorf("%w: not requires of_one", ErrInvalidPredicate)
		}
		if err := validatePredicate(p.OfOne, allowCEL); err != nil {
			return err
		}
	case PredCEL:
		if !allowCEL {
			return fmt.Errorf("%w: cel predicate not permitted in this strategy", ErrInvalidPredicate)
		}
		if p.Source == "" {
			return fmt.Errorf("%w: cel missing source", ErrInvalidPredicate)
		}
	case PredAlways:
		// no fields required
	case PredSegment:
		// Standalone validation accepts segment references; they are
		// expected to have been resolved away before Compile runs. If
		// a segment reference survives into Compile it'll fail in
		// compilePredicates with an explanatory error.
		if p.SegmentKey == "" {
			return fmt.Errorf("%w: segment missing key", ErrInvalidPredicate)
		}
	default:
		return fmt.Errorf("%w: unknown kind %q", ErrInvalidPredicate, p.Kind)
	}
	return nil
}
