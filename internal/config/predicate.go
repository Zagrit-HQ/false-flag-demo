package config

import "fmt"

// ValidatePredicate exposes the predicate validator for callers
// outside the strategy compilers — segment write handlers in
// particular need to validate a standalone predicate before storing
// it. allowCEL toggles whether `kind: "cel"` is permitted.
//
// The internal implementation is unchanged from slice 2; this is
// purely a re-export to keep the package boundary clean.
func ValidatePredicate(p *Predicate, allowCEL bool) error {
	return validatePredicate(p, allowCEL)
}

// SegmentResolver looks up a project's stored segment predicate by
// key. It is implemented at the handler boundary so internal/config
// stays free of any persistence dependency.
type SegmentResolver interface {
	ResolveSegment(key string) (*Predicate, error)
}

// ResolveSegments walks every Predicate in tree and replaces every
// `{kind: "segment", key: "<x>"}` node with the inlined predicate
// returned by resolver.ResolveSegment("<x>"). The replacement is
// recursive: the inlined predicate is itself walked so nested
// segment references resolve. Cycle detection is intentionally not
// implemented — segment definitions can't reference other segments
// in slice 3 (the segment write handler rejects them).
//
// Mutates tree in place. Returns ErrInvalidPredicate wrapping the
// resolver's error if a segment can't be resolved.
func ResolveSegments(tree *RulesTree, resolver SegmentResolver) error {
	for i := range tree.Rules {
		if tree.Rules[i].When == nil {
			continue
		}
		resolved, err := resolveOne(tree.Rules[i].When, resolver)
		if err != nil {
			return fmt.Errorf("rule %q: %w", tree.Rules[i].ID, err)
		}
		tree.Rules[i].When = resolved
	}
	return nil
}

func resolveOne(p *Predicate, resolver SegmentResolver) (*Predicate, error) {
	if p == nil {
		return nil, nil
	}
	if p.Kind == PredSegment {
		seg, err := resolver.ResolveSegment(p.SegmentKey)
		if err != nil {
			return nil, fmt.Errorf("%w: segment %q: %s", ErrInvalidPredicate, p.SegmentKey, err)
		}
		if seg == nil {
			return nil, fmt.Errorf("%w: segment %q resolved to nil", ErrInvalidPredicate, p.SegmentKey)
		}
		// Recurse in case the segment definition itself contains a
		// segment reference (currently rejected at write time, but
		// future-proof).
		return resolveOne(seg, resolver)
	}
	for i, child := range p.Of {
		resolved, err := resolveOne(child, resolver)
		if err != nil {
			return nil, err
		}
		p.Of[i] = resolved
	}
	if p.OfOne != nil {
		resolved, err := resolveOne(p.OfOne, resolver)
		if err != nil {
			return nil, err
		}
		p.OfOne = resolved
	}
	return p, nil
}
