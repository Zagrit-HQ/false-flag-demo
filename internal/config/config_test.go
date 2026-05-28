package config_test

import (
	"errors"
	"testing"

	"github.com/depot/falseflag/internal/config"
)

func TestStrategyValid(t *testing.T) {
	cases := map[config.Strategy]bool{
		config.StrategyJSON:       true,
		config.StrategyCEL:        true,
		config.StrategyTypeScript: true,
		config.Strategy("ruby"):   false,
		config.Strategy(""):       false,
	}
	for s, want := range cases {
		if got := s.Valid(); got != want {
			t.Errorf("Strategy(%q).Valid() = %v, want %v", s, got, want)
		}
	}
}

func TestJSONCompileHappyPath(t *testing.T) {
	src := []byte(`{
		"value_type": "boolean",
		"default": false,
		"rules": [
			{"id": "r1", "when": {"kind":"eq","attr":"user.plan","value":"\"pro\""}, "value": true}
		]
	}`)
	c, err := config.Compile(config.StrategyJSON, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Strategy != config.StrategyJSON {
		t.Errorf("Strategy = %q, want json", c.Strategy)
	}
	if c.IR.ValueType != config.ValueTypeBoolean {
		t.Errorf("ValueType = %q, want boolean", c.IR.ValueType)
	}
	if len(c.IR.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1", len(c.IR.Rules))
	}
	if c.CELPrograms != nil {
		t.Errorf("JSON strategy must not produce CEL programs; got %d", len(c.CELPrograms))
	}
}

func TestJSONCompileRejectsCEL(t *testing.T) {
	src := []byte(`{
		"value_type": "boolean",
		"default": false,
		"rules": [
			{"id": "r1", "when": {"kind":"cel","source":"ctx.user.age > 18"}, "value": true}
		]
	}`)
	_, err := config.Compile(config.StrategyJSON, src)
	if err == nil {
		t.Fatal("expected error rejecting CEL predicate in JSON strategy")
	}
	if !errors.Is(err, config.ErrInvalidPredicate) {
		t.Errorf("want ErrInvalidPredicate, got %v", err)
	}
}

func TestJSONCompileRejectsBadValueType(t *testing.T) {
	src := []byte(`{"value_type":"date","default":null,"rules":[]}`)
	_, err := config.Compile(config.StrategyJSON, src)
	if !errors.Is(err, config.ErrInvalidValueType) {
		t.Errorf("want ErrInvalidValueType, got %v", err)
	}
}

func TestCELCompileHappyPath(t *testing.T) {
	src := []byte(`{
		"value_type": "string",
		"default": "\"control\"",
		"rules": [
			{"id":"r1","when":{"kind":"cel","source":"ctx.user.country == 'US'"},"value":"\"treatment\""}
		]
	}`)
	c, err := config.Compile(config.StrategyCEL, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Strategy != config.StrategyCEL {
		t.Errorf("Strategy = %q, want cel", c.Strategy)
	}
	if len(c.CELPrograms) != 1 {
		t.Errorf("CELPrograms = %d, want 1", len(c.CELPrograms))
	}
	if _, ok := c.CELPrograms["ctx.user.country == 'US'"]; !ok {
		t.Error("expected program keyed by source string")
	}
}

func TestCELCompileRejectsBadSource(t *testing.T) {
	src := []byte(`{
		"value_type": "boolean",
		"default": false,
		"rules": [
			{"id":"r1","when":{"kind":"cel","source":"this is not cel ((("}, "value": true}
		]
	}`)
	_, err := config.Compile(config.StrategyCEL, src)
	if !errors.Is(err, config.ErrCELCompileFailure) {
		t.Errorf("want ErrCELCompileFailure, got %v", err)
	}
}

func TestCELCompileRejectsNonBoolReturn(t *testing.T) {
	src := []byte(`{
		"value_type": "boolean",
		"default": false,
		"rules": [
			{"id":"r1","when":{"kind":"cel","source":"ctx.user.country"}, "value": true}
		]
	}`)
	_, err := config.Compile(config.StrategyCEL, src)
	if !errors.Is(err, config.ErrCELCompileFailure) {
		t.Errorf("want ErrCELCompileFailure for non-bool CEL, got %v", err)
	}
}

func TestTypeScriptCompileAcceptsCEL(t *testing.T) {
	// TS-authored flags may embed CEL predicates via ff.cel(...).
	// Server-side compile must compile the embedded CEL program.
	src := []byte(`import { FalseFlag as ff } from "@falseflag/config";

export default ff.flag({
  value_type: "boolean",
  default: false,
  rules: [
    ff.rule("ts-1",
      ff.all(
        ff.eq("user.plan", "\"pro\""),
        ff.cel("ctx.user.age > 18"),
      ),
      true),
  ],
});`)
	c, err := config.Compile(config.StrategyTypeScript, src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Strategy != config.StrategyTypeScript {
		t.Errorf("Strategy = %q, want typescript", c.Strategy)
	}
	if len(c.CELPrograms) != 1 {
		t.Errorf("want 1 cel program, got %d", len(c.CELPrograms))
	}
}

func TestUnknownStrategy(t *testing.T) {
	_, err := config.Compile(config.Strategy("ruby"), []byte(`{}`))
	if !errors.Is(err, config.ErrUnknownStrategy) {
		t.Errorf("want ErrUnknownStrategy, got %v", err)
	}
}

func TestRolloutPercentBounds(t *testing.T) {
	bad := []byte(`{
		"value_type": "boolean", "default": false,
		"rules": [{"id":"r1","when":{"kind":"rollout","attr":"user.id","salt":"f","percent":101},"value":true}]
	}`)
	_, err := config.Compile(config.StrategyJSON, bad)
	if !errors.Is(err, config.ErrInvalidPredicate) {
		t.Errorf("want ErrInvalidPredicate for percent>100, got %v", err)
	}
}
