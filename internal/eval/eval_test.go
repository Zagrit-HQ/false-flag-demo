package eval_test

import (
	"testing"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/eval"
)

// compile is a tiny helper that mirrors the runtime path (parse +
// validate), keeping tests honest about the IR shape going in.
func compile(t *testing.T, strategy config.Strategy, src string) *config.Compiled {
	t.Helper()
	c, err := config.Compile(strategy, []byte(src))
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return c
}

func TestEvaluateDefaultWhenNoRules(t *testing.T) {
	c := compile(t, config.StrategyJSON, `{
		"value_type":"boolean","default":false,"rules":[]
	}`)
	d, err := eval.Evaluate(c, nil, 1)
	if err != nil {
		t.Fatal(err)
	}
	if d.Reason != eval.ReasonDefault {
		t.Errorf("Reason = %q, want default", d.Reason)
	}
	if d.Value != false {
		t.Errorf("Value = %v, want false", d.Value)
	}
}

func TestEvaluateEqMatch(t *testing.T) {
	c := compile(t, config.StrategyJSON, `{
		"value_type":"string","default":"control","rules":[
			{"id":"r1","when":{"kind":"eq","attr":"user.plan","value":"pro"},"value":"treatment"}
		]
	}`)
	d, err := eval.Evaluate(c, map[string]any{"user": map[string]any{"plan": "pro"}}, 7)
	if err != nil {
		t.Fatal(err)
	}
	if d.Value != "treatment" {
		t.Errorf("Value = %v, want treatment", d.Value)
	}
	if d.Reason != eval.ReasonRuleMatched {
		t.Errorf("Reason = %q", d.Reason)
	}
	if d.RuleID != "r1" {
		t.Errorf("RuleID = %q", d.RuleID)
	}
	if d.Version != 7 {
		t.Errorf("Version = %d", d.Version)
	}
}

func TestEvaluateInMatch(t *testing.T) {
	c := compile(t, config.StrategyJSON, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r1","when":{"kind":"in","attr":"user.plan","values":["pro","enterprise"]},"value":true}
		]
	}`)
	cases := map[string]bool{
		"free":       false,
		"pro":        true,
		"enterprise": true,
		"trial":      false,
	}
	for plan, want := range cases {
		d, err := eval.Evaluate(c, map[string]any{"user": map[string]any{"plan": plan}}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got := d.Value.(bool); got != want {
			t.Errorf("plan=%s value=%v want %v", plan, got, want)
		}
	}
}

func TestEvaluateOrdGt(t *testing.T) {
	c := compile(t, config.StrategyJSON, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r1","when":{"kind":"gt","attr":"user.age","value":18},"value":true}
		]
	}`)
	d, _ := eval.Evaluate(c, map[string]any{"user": map[string]any{"age": float64(21)}}, 1)
	if d.Value != true {
		t.Errorf("age=21 expected true, got %v", d.Value)
	}
	d, _ = eval.Evaluate(c, map[string]any{"user": map[string]any{"age": float64(15)}}, 1)
	if d.Value != false {
		t.Errorf("age=15 expected default false, got %v", d.Value)
	}
}

func TestEvaluateAllAnyNot(t *testing.T) {
	c := compile(t, config.StrategyJSON, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r1","when":{"kind":"all","of":[
				{"kind":"eq","attr":"user.country","value":"US"},
				{"kind":"any","of":[
					{"kind":"eq","attr":"user.plan","value":"pro"},
					{"kind":"not","of_one":{"kind":"eq","attr":"user.trial","value":true}}
				]}
			]},"value":true}
		]
	}`)
	d, _ := eval.Evaluate(c, map[string]any{"user": map[string]any{"country": "US", "plan": "free", "trial": false}}, 1)
	if d.Value != true {
		t.Errorf("expected true (not trial path), got %v", d.Value)
	}
	d, _ = eval.Evaluate(c, map[string]any{"user": map[string]any{"country": "CA", "plan": "pro"}}, 1)
	if d.Value != false {
		t.Errorf("expected false (country=CA fails outer all), got %v", d.Value)
	}
}

func TestEvaluateRollout(t *testing.T) {
	c := compile(t, config.StrategyJSON, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r1","when":{"kind":"rollout","attr":"user.id","salt":"checkout-v2","percent":50},"value":true}
		]
	}`)
	// Determinism: the same id always lands in the same bucket.
	for _, id := range []string{"u-1", "u-42", "u-9999", "u-feature"} {
		d1, _ := eval.Evaluate(c, map[string]any{"user": map[string]any{"id": id}}, 1)
		d2, _ := eval.Evaluate(c, map[string]any{"user": map[string]any{"id": id}}, 1)
		if d1.Value != d2.Value {
			t.Errorf("id=%s produced non-deterministic decision", id)
		}
	}
	// Edge: percent=0 → never in.
	zero := compile(t, config.StrategyJSON, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r0","when":{"kind":"rollout","attr":"user.id","salt":"f","percent":0},"value":true}
		]
	}`)
	d, _ := eval.Evaluate(zero, map[string]any{"user": map[string]any{"id": "any"}}, 1)
	if d.Value != false {
		t.Errorf("percent=0 should never serve true")
	}
	// Edge: percent=100 → always in.
	full := compile(t, config.StrategyJSON, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r100","when":{"kind":"rollout","attr":"user.id","salt":"f","percent":100},"value":true}
		]
	}`)
	d, _ = eval.Evaluate(full, map[string]any{"user": map[string]any{"id": "any"}}, 1)
	if d.Value != true {
		t.Errorf("percent=100 should always serve true")
	}
}

func TestEvaluateCEL(t *testing.T) {
	c := compile(t, config.StrategyCEL, `{
		"value_type":"string","default":"basic","rules":[
			{"id":"r1","when":{"kind":"cel","source":"ctx.user.age >= 21 && ctx.user.country == 'US'"},"value":"premium"}
		]
	}`)
	d, err := eval.Evaluate(c, map[string]any{"user": map[string]any{"age": float64(25), "country": "US"}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if d.Value != "premium" {
		t.Errorf("Value = %v, want premium", d.Value)
	}
	d, _ = eval.Evaluate(c, map[string]any{"user": map[string]any{"age": float64(25), "country": "CA"}}, 1)
	if d.Value != "basic" {
		t.Errorf("CA should fall to default, got %v", d.Value)
	}
}

func TestEvaluateMissingAttrFalls(t *testing.T) {
	c := compile(t, config.StrategyJSON, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r1","when":{"kind":"eq","attr":"user.country","value":"US"},"value":true}
		]
	}`)
	// No user.country in context — should serve default, not error.
	d, err := eval.Evaluate(c, map[string]any{}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Reason != eval.ReasonDefault {
		t.Errorf("Reason = %q, want default", d.Reason)
	}
}

func TestEvaluateMatchesRegex(t *testing.T) {
	c := compile(t, config.StrategyJSON, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r1","when":{"kind":"matches","attr":"user.email","pattern":".+@depot\\.dev$"},"value":true}
		]
	}`)
	d, _ := eval.Evaluate(c, map[string]any{"user": map[string]any{"email": "kyle@depot.dev"}}, 1)
	if d.Value != true {
		t.Errorf("email match should yield true, got %v", d.Value)
	}
	d, _ = eval.Evaluate(c, map[string]any{"user": map[string]any{"email": "kyle@example.com"}}, 1)
	if d.Value != false {
		t.Errorf("non-matching email should yield default false, got %v", d.Value)
	}
}

func TestEvaluateAlwaysFires(t *testing.T) {
	c := compile(t, config.StrategyJSON, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r1","when":{"kind":"always"},"value":true}
		]
	}`)
	d, _ := eval.Evaluate(c, nil, 1)
	if d.Value != true {
		t.Errorf("always should fire, got %v", d.Value)
	}
}
