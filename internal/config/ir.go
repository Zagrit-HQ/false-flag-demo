// Package config holds the project-scoped configuration strategy
// implementations (JSON, CEL, TypeScript) that all compile down to the
// same normalized rules-tree IR. The IR is what the evaluator
// (internal/eval) and persistence layer (internal/store) actually
// consume — the runtime doesn't know or care which strategy produced
// it. See docs/plans/2026-05-20-002-feat-configuration-strategies-plan.md.
package config

import (
	"encoding/json"
)

// ValueType enumerates the typed value kinds a flag may return. It
// mirrors OpenFeature's typed flag evaluation.
type ValueType string

const (
	ValueTypeBoolean ValueType = "boolean"
	ValueTypeString  ValueType = "string"
	ValueTypeNumber  ValueType = "number"
	ValueTypeObject  ValueType = "object"
)

// RulesTree is the normalized IR. Every strategy compiles into this.
// The first rule whose `When` predicate matches wins; if none match,
// Default is served.
type RulesTree struct {
	ValueType ValueType       `json:"value_type"`
	Default   json.RawMessage `json:"default"`
	Rules     []Rule          `json:"rules"`
}

// Rule is a single targeting clause.
type Rule struct {
	ID    string          `json:"id"`
	When  *Predicate      `json:"when"`
	Value json.RawMessage `json:"value"`
}

// PredicateKind enumerates the predicate node types. Keep this list in
// sync with internal/eval/predicates.go and js/packages/sdk-js/src/evaluator.ts.
type PredicateKind string

const (
	PredEq      PredicateKind = "eq"
	PredNeq     PredicateKind = "neq"
	PredIn      PredicateKind = "in"
	PredGt      PredicateKind = "gt"
	PredGte     PredicateKind = "gte"
	PredLt      PredicateKind = "lt"
	PredLte     PredicateKind = "lte"
	PredMatches PredicateKind = "matches"
	PredRollout PredicateKind = "rollout"
	PredAll     PredicateKind = "all"
	PredAny     PredicateKind = "any"
	PredNot     PredicateKind = "not"
	PredCEL     PredicateKind = "cel"
	PredAlways  PredicateKind = "always"
	// PredSegment is a reference predicate. The handler resolves it
	// against the project's stored segments before Compile runs; the
	// kind never appears in the persisted IR.
	PredSegment PredicateKind = "segment"
)

// Predicate is a node in the predicate tree. Different kinds use
// different subsets of fields; unset fields stay zero-valued. The
// JSON shape is intentionally flat-with-kind so it matches what the
// TypeScript DSL emits and what Postgres jsonb stores.
type Predicate struct {
	Kind PredicateKind `json:"kind"`

	// Comparison predicates: eq, neq, gt, gte, lt, lte, matches
	Attr  string          `json:"attr,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`

	// Membership: in
	Values []json.RawMessage `json:"values,omitempty"`

	// matches
	Pattern string `json:"pattern,omitempty"`

	// rollout
	Salt    string `json:"salt,omitempty"`
	Percent int    `json:"percent,omitempty"`

	// all, any
	Of []*Predicate `json:"of,omitempty"`

	// not (single child stored in OfOne to disambiguate JSON shape)
	OfOne *Predicate `json:"of_one,omitempty"`

	// cel
	Source string `json:"source,omitempty"`

	// segment (resolved away at publish time by the handler)
	SegmentKey string `json:"key,omitempty"`
}

// Compiled is a strategy-agnostic compiled flag configuration. The IR
// is what the evaluator reads; CELPrograms holds pre-parsed cel.Program
// values for any CEL predicate sources discovered during Compile. The
// programs are not serialized — they're rebuilt when a Compiled is
// loaded from persistence via LoadCompiled.
type Compiled struct {
	Strategy Strategy
	IR       *RulesTree
	// CELPrograms is keyed by the predicate's `source` string. Distinct
	// predicates with identical source strings share a program, which
	// is fine because cel.Program evaluation is stateless given a
	// context.
	CELPrograms map[string]CELProgram
}
