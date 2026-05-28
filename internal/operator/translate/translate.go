// Package translate converts FalseFlag CR specs into the slice-2 IR
// JSON shape the upstream API expects. Pure Go — no k8s or Connect
// dependencies — so it can be unit-tested without envtest.
package translate

import (
	"encoding/json"
	"fmt"

	"github.com/depot/falseflag/internal/config"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

// IRForFlag turns a Flag CR into the IR RulesTree the API publishes.
// rollouts may be nil if the flag does not reference any
// RolloutPolicy CRs. When binding is non-nil the resulting IR is
// specialised to that environment (using binding.Default and, if set,
// binding.Rules instead of the parent flag's rules).
func IRForFlag(flag *v1alpha1.Flag, rollouts map[string]*v1alpha1.RolloutPolicy, binding *v1alpha1.FlagSpecBinding) (*config.RulesTree, error) {
	if flag == nil {
		return nil, fmt.Errorf("flag is nil")
	}
	def := flag.Spec.Default
	rules := flag.Spec.Rules
	if binding != nil {
		if binding.Default != nil {
			def = *binding.Default
		}
		if binding.Rules != nil {
			rules = binding.Rules
		}
	}

	tree := &config.RulesTree{
		ValueType: config.ValueType(flag.Spec.ValueType),
		Default:   json.RawMessage(def.Raw),
	}

	for _, r := range rules {
		when, err := unmarshalPredicate(r.When.Raw)
		if err != nil {
			return nil, fmt.Errorf("rule %q when: %w", r.ID, err)
		}

		switch {
		case r.RolloutRef != "":
			rp, ok := rollouts[r.RolloutRef]
			if !ok {
				return nil, fmt.Errorf("rule %q references unknown RolloutPolicy %q", r.ID, r.RolloutRef)
			}
			// Inline as the first variant's value with a rollout
			// predicate wrapping `when`. Demo-quality: a real impl
			// would emit a multi-variant rollout node; for the
			// audience-visible path we just take the heaviest variant.
			value, err := pickRolloutValue(rp)
			if err != nil {
				return nil, fmt.Errorf("rule %q rollout: %w", r.ID, err)
			}
			tree.Rules = append(tree.Rules, config.Rule{ID: r.ID, When: when, Value: value})
		case r.Value != nil:
			tree.Rules = append(tree.Rules, config.Rule{ID: r.ID, When: when, Value: json.RawMessage(r.Value.Raw)})
		default:
			return nil, fmt.Errorf("rule %q must set value or rolloutRef", r.ID)
		}
	}

	return tree, nil
}

// IRForSegment turns a Segment CR into the IR Predicate the API
// expects on POST /segments.
func IRForSegment(seg *v1alpha1.Segment) (*config.Predicate, error) {
	if seg == nil {
		return nil, fmt.Errorf("segment is nil")
	}
	return unmarshalPredicate(seg.Spec.Predicate.Raw)
}

// PublishSource returns the {strategy, source: {...IR...}} shape the
// PublishFlagVersion endpoint accepts under strategy="json".
func PublishSource(tree *config.RulesTree) (map[string]any, error) {
	data, err := json.Marshal(tree)
	if err != nil {
		return nil, err
	}
	var src map[string]any
	if err := json.Unmarshal(data, &src); err != nil {
		return nil, err
	}
	return src, nil
}

func unmarshalPredicate(raw []byte) (*config.Predicate, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("predicate is empty")
	}
	p := &config.Predicate{}
	if err := json.Unmarshal(raw, p); err != nil {
		return nil, fmt.Errorf("unmarshal predicate: %w", err)
	}
	return p, nil
}

func pickRolloutValue(rp *v1alpha1.RolloutPolicy) (json.RawMessage, error) {
	if len(rp.Spec.Variants) == 0 {
		return nil, fmt.Errorf("rollout policy %q has no variants", rp.Spec.Name)
	}
	var heaviest v1alpha1.RolloutVariant
	for _, v := range rp.Spec.Variants {
		if v.Weight > heaviest.Weight {
			heaviest = v
		}
	}
	if heaviest.Value.Raw == nil {
		heaviest = rp.Spec.Variants[0]
	}
	return json.RawMessage(heaviest.Value.Raw), nil
}
