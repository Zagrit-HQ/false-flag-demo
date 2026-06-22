package config

import (
	"errors"
	"fmt"
	"testing"
)

func TestStrategy_Valid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		s    Strategy
		want bool
	}{
		{StrategyJSON, true},
		{StrategyCEL, true},
		{StrategyTypeScript, true},
		{Strategy(""), false},
		{Strategy("yaml"), false},
		{Strategy("JSON"), false},
		{Strategy("Json"), false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.s), func(t *testing.T) {
			t.Parallel()
			if got := tc.s.Valid(); got != tc.want {
				t.Errorf("Valid(%q) = %v, want %v", tc.s, got, tc.want)
			}
		})
	}
}

func TestCompilerFor(t *testing.T) {
	t.Parallel()
	for _, s := range AllStrategies {
		s := s
		t.Run(string(s), func(t *testing.T) {
			t.Parallel()
			c, err := CompilerFor(s)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if c == nil {
				t.Fatal("compiler nil")
			}
			if c.Strategy() != s {
				t.Errorf("Strategy() = %v want %v", c.Strategy(), s)
			}
		})
	}
}

func TestCompilerFor_Unknown(t *testing.T) {
	t.Parallel()
	cases := []Strategy{"", "yaml", "Json", "TypeScript", "cel-v2"}
	for _, s := range cases {
		s := s
		t.Run(fmt.Sprintf("%q", s), func(t *testing.T) {
			t.Parallel()
			_, err := CompilerFor(s)
			if err == nil {
				t.Errorf("err should be non-nil for %q", s)
			}
			if !errors.Is(err, ErrUnknownStrategy) {
				t.Errorf("err = %v, want wraps ErrUnknownStrategy", err)
			}
		})
	}
}

func TestCompile_DispatchesPerStrategy(t *testing.T) {
	t.Parallel()
	src := []byte(`{"value_type":"boolean","default":false,"rules":[]}`)
	for _, s := range []Strategy{StrategyJSON, StrategyCEL} {
		s := s
		t.Run(string(s), func(t *testing.T) {
			t.Parallel()
			c, err := Compile(s, src)
			if err != nil {
				t.Fatalf("compile %s: %v", s, err)
			}
			if c.Strategy != s {
				t.Errorf("Strategy = %v want %v", c.Strategy, s)
			}
			if c.IR == nil {
				t.Fatal("IR nil")
			}
			if c.IR.ValueType != "boolean" {
				t.Errorf("ValueType = %q", c.IR.ValueType)
			}
		})
	}
}

func TestCompile_UnknownStrategyErrors(t *testing.T) {
	t.Parallel()
	_, err := Compile(Strategy("yaml"), []byte(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnknownStrategy) {
		t.Errorf("err = %v, want wraps ErrUnknownStrategy", err)
	}
}

func TestSentinelErrors_Distinct(t *testing.T) {
	t.Parallel()
	all := []error{
		ErrUnknownStrategy,
		ErrInvalidIR,
		ErrInvalidPredicate,
		ErrInvalidValueType,
		ErrCELCompileFailure,
		ErrTypeScriptCompileFailure,
	}
	for i, a := range all {
		for j, b := range all {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("sentinel %v should not match %v", a, b)
			}
		}
	}
}
