package config

import (
	"encoding/json"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
)

// CELProgram is the runtime-evaluable form of a CEL source string. We
// alias the cel-go type so callers in internal/eval don't import cel-go
// directly.
type CELProgram = cel.Program

type celCompiler struct{}

func (celCompiler) Strategy() Strategy { return StrategyCEL }

// Compile validates the IR structurally and pre-compiles every CEL
// predicate's source string into a cel.Program. CEL programs are
// stored on Compiled.CELPrograms keyed by source string so the
// evaluator can look them up by predicate.Source at eval time.
//
// Distinct predicates with identical sources share a program, which is
// fine: cel.Program evaluation is stateless given an Activation.
func (celCompiler) Compile(source []byte) (*Compiled, error) {
	var tree RulesTree
	if err := json.Unmarshal(source, &tree); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidIR, err)
	}
	// CEL predicates ARE allowed in this strategy.
	if err := validateTreeWith(&tree, true); err != nil {
		return nil, err
	}

	programs := map[string]cel.Program{}
	env, err := celEnv()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCELCompileFailure, err)
	}
	if err := compilePredicates(tree.Rules, env, programs); err != nil {
		return nil, err
	}
	return &Compiled{
		Strategy:    StrategyCEL,
		IR:          &tree,
		CELPrograms: programs,
	}, nil
}

// validateTreeWith is the CEL-aware counterpart to validateTree.
func validateTreeWith(tree *RulesTree, allowCEL bool) error {
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
		if err := validatePredicate(r.When, allowCEL); err != nil {
			return fmt.Errorf("rule %q: %w", r.ID, err)
		}
	}
	return nil
}

// celEnv constructs the CEL environment used for compilation and
// evaluation. The environment declares a single dynamic variable
// `ctx` of type map(string, dyn) which is what the evaluator passes
// in. Predicate authors write expressions like `ctx.user.country == "US"`.
func celEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Variable("ctx", cel.MapType(cel.StringType, cel.DynType)),
	)
}

func compilePredicates(rules []Rule, env *cel.Env, out map[string]cel.Program) error {
	for i := range rules {
		if err := walkAndCompile(rules[i].When, env, out); err != nil {
			return fmt.Errorf("rule %q: %w", rules[i].ID, err)
		}
	}
	return nil
}

func walkAndCompile(p *Predicate, env *cel.Env, out map[string]cel.Program) error {
	if p == nil {
		return nil
	}
	switch p.Kind {
	case PredCEL:
		if _, ok := out[p.Source]; ok {
			return nil
		}
		ast, iss := env.Compile(p.Source)
		if iss != nil && iss.Err() != nil {
			return fmt.Errorf("%w: %s", ErrCELCompileFailure, iss.Err())
		}
		if !ast.OutputType().IsExactType(cel.BoolType) {
			return fmt.Errorf("%w: cel expression must return bool, got %s", ErrCELCompileFailure, ast.OutputType())
		}
		prog, err := env.Program(ast)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrCELCompileFailure, err)
		}
		out[p.Source] = prog
	case PredAll, PredAny:
		for _, c := range p.Of {
			if err := walkAndCompile(c, env, out); err != nil {
				return err
			}
		}
	case PredNot:
		if err := walkAndCompile(p.OfOne, env, out); err != nil {
			return err
		}
	}
	return nil
}

// EvalCEL runs a previously compiled program against ctx. It returns
// the boolean output; non-bool outputs are treated as false.
//
// This lives here (rather than in internal/eval) so callers can swap
// the implementation without leaking cel-go into the evaluator's
// imports. The evaluator goes through Compiled.CELPrograms.
func EvalCEL(prog cel.Program, ctx map[string]any) (bool, error) {
	out, _, err := prog.Eval(map[string]any{"ctx": ctx})
	if err != nil {
		return false, err
	}
	return celRefToBool(out), nil
}

func celRefToBool(v ref.Val) bool {
	if b, ok := v.(types.Bool); ok {
		return bool(b)
	}
	return false
}
