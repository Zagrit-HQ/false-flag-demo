package config

import (
	"errors"
	"fmt"
)

// Strategy enumerates the project-scoped configuration strategies
// FalseFlag supports. The string values match the CHECK constraint on
// projects.config_strategy in db/migrations/0001_init.sql and the
// strategy CHECK on flag_versions in 0002_flags.sql.
type Strategy string

const (
	StrategyJSON       Strategy = "json"
	StrategyCEL        Strategy = "cel"
	StrategyTypeScript Strategy = "typescript"
)

// AllStrategies is the canonical iteration order.
var AllStrategies = []Strategy{StrategyJSON, StrategyCEL, StrategyTypeScript}

// Valid reports whether s is one of the known strategies.
func (s Strategy) Valid() bool {
	switch s {
	case StrategyJSON, StrategyCEL, StrategyTypeScript:
		return true
	default:
		return false
	}
}

// Compiler turns a strategy-specific source blob into a strategy-agnostic
// Compiled. Implementations live in json.go, cel.go, and typescript.go.
type Compiler interface {
	Strategy() Strategy
	Compile(source []byte) (*Compiled, error)
}

// CompilerFor returns the Compiler for a given strategy.
func CompilerFor(s Strategy) (Compiler, error) {
	switch s {
	case StrategyJSON:
		return jsonCompiler{}, nil
	case StrategyCEL:
		return celCompiler{}, nil
	case StrategyTypeScript:
		return typescriptCompiler{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnknownStrategy, s)
	}
}

// Compile is a convenience wrapper that picks the compiler for s and
// runs it against source.
func Compile(s Strategy, source []byte) (*Compiled, error) {
	c, err := CompilerFor(s)
	if err != nil {
		return nil, err
	}
	return c.Compile(source)
}

// Sentinel errors.
var (
	ErrUnknownStrategy          = errors.New("unknown strategy")
	ErrInvalidIR                = errors.New("invalid IR")
	ErrInvalidPredicate         = errors.New("invalid predicate")
	ErrInvalidValueType         = errors.New("invalid value_type")
	ErrCELCompileFailure        = errors.New("cel compile failure")
	ErrTypeScriptCompileFailure = errors.New("typescript compile failure")
)
