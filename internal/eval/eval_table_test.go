package eval_test

import (
	"fmt"
	"testing"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/eval"
)

// Many table-driven eval cases. The size of this file is intentional:
// it exercises every predicate kind across many value shapes so the
// CI demo has a realistic test load. Each case is a sealed compile +
// evaluate pair.

func mustCompile(t *testing.T, src string) *config.Compiled {
	t.Helper()
	c, err := config.Compile(config.StrategyJSON, []byte(src))
	if err != nil {
		t.Fatalf("compile: %v\nsource: %s", err, src)
	}
	return c
}

func TestEvaluate_EqVariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		attr    string
		expect  string // raw json
		ctx     map[string]any
		matches bool
	}{
		{"string-match", "user.plan", `"pro"`, map[string]any{"user": map[string]any{"plan": "pro"}}, true},
		{"string-miss", "user.plan", `"pro"`, map[string]any{"user": map[string]any{"plan": "free"}}, false},
		{"string-case-sensitive", "user.plan", `"Pro"`, map[string]any{"user": map[string]any{"plan": "pro"}}, false},
		{"bool-true", "user.beta", `true`, map[string]any{"user": map[string]any{"beta": true}}, true},
		{"bool-false-match", "user.beta", `false`, map[string]any{"user": map[string]any{"beta": false}}, true},
		{"num-int-match", "user.age", `21`, map[string]any{"user": map[string]any{"age": float64(21)}}, true},
		{"num-float-match", "score", `3.14`, map[string]any{"score": float64(3.14)}, true},
		{"missing-attr", "user.plan", `"pro"`, map[string]any{"user": map[string]any{}}, false},
		{"missing-parent", "user.plan", `"pro"`, map[string]any{}, false},
		{"top-level", "env", `"prod"`, map[string]any{"env": "prod"}, true},
		{"unicode", "name", `"漢字"`, map[string]any{"name": "漢字"}, true},
		{"empty-string", "name", `""`, map[string]any{"name": ""}, true},
		{"empty-string-miss", "name", `""`, map[string]any{"name": "x"}, false},
		{"deep-nested", "a.b.c.d", `"x"`, map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": "x"}}}}, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			src := fmt.Sprintf(`{
				"value_type":"boolean","default":false,"rules":[
					{"id":"r","when":{"kind":"eq","attr":%q,"value":%s},"value":true}
				]
			}`, tc.attr, tc.expect)
			c := mustCompile(t, src)
			d, err := eval.Evaluate(c, tc.ctx, 1)
			if err != nil {
				t.Fatalf("eval: %v", err)
			}
			got := d.Value == true
			if got != tc.matches {
				t.Errorf("matched = %v, want %v", got, tc.matches)
			}
		})
	}
}

func TestEvaluate_NeqVariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		expect  string
		ctx     map[string]any
		matches bool
	}{
		{"different-string", `"pro"`, map[string]any{"plan": "free"}, true},
		{"same-string", `"pro"`, map[string]any{"plan": "pro"}, false},
		{"missing", `"pro"`, map[string]any{}, true},
		{"different-number", `42`, map[string]any{"plan": float64(7)}, true},
		{"same-number", `42`, map[string]any{"plan": float64(42)}, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			src := fmt.Sprintf(`{
				"value_type":"boolean","default":false,"rules":[
					{"id":"r","when":{"kind":"neq","attr":"plan","value":%s},"value":true}
				]
			}`, tc.expect)
			c := mustCompile(t, src)
			d, _ := eval.Evaluate(c, tc.ctx, 1)
			if (d.Value == true) != tc.matches {
				t.Errorf("got %v want match=%v", d.Value, tc.matches)
			}
		})
	}
}

func TestEvaluate_InVariants(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		values  string
		ctx     map[string]any
		matches bool
	}{
		{"hit-first", `["a","b","c"]`, map[string]any{"k": "a"}, true},
		{"hit-mid", `["a","b","c"]`, map[string]any{"k": "b"}, true},
		{"hit-last", `["a","b","c"]`, map[string]any{"k": "c"}, true},
		{"miss", `["a","b","c"]`, map[string]any{"k": "z"}, false},
		{"singleton-hit", `["solo"]`, map[string]any{"k": "solo"}, true},
		{"singleton-miss", `["solo"]`, map[string]any{"k": "other"}, false},
		{"missing-attr", `["a"]`, map[string]any{}, false},
		{"num-hit", `[1,2,3]`, map[string]any{"k": float64(2)}, true},
		{"num-miss", `[1,2,3]`, map[string]any{"k": float64(5)}, false},
		{"bool-hit", `[true]`, map[string]any{"k": true}, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			src := fmt.Sprintf(`{
				"value_type":"boolean","default":false,"rules":[
					{"id":"r","when":{"kind":"in","attr":"k","values":%s},"value":true}
				]
			}`, tc.values)
			c := mustCompile(t, src)
			d, _ := eval.Evaluate(c, tc.ctx, 1)
			if (d.Value == true) != tc.matches {
				t.Errorf("got %v want match=%v", d.Value, tc.matches)
			}
		})
	}
}

func TestEvaluate_Ord(t *testing.T) {
	t.Parallel()
	kinds := []string{"gt", "gte", "lt", "lte"}
	bounds := []float64{0, 1, 17.5, 100, -3}
	values := []float64{-10, -3, 0, 0.5, 1, 17.5, 18, 50, 100, 1000}
	for _, k := range kinds {
		for _, b := range bounds {
			for _, v := range values {
				k, b, v := k, b, v
				t.Run(fmt.Sprintf("%s/bound=%v/val=%v", k, b, v), func(t *testing.T) {
					t.Parallel()
					src := fmt.Sprintf(`{
						"value_type":"boolean","default":false,"rules":[
							{"id":"r","when":{"kind":%q,"attr":"x","value":%v},"value":true}
						]
					}`, k, b)
					c := mustCompile(t, src)
					d, _ := eval.Evaluate(c, map[string]any{"x": v}, 1)
					var want bool
					switch k {
					case "gt":
						want = v > b
					case "gte":
						want = v >= b
					case "lt":
						want = v < b
					case "lte":
						want = v <= b
					}
					if (d.Value == true) != want {
						t.Errorf("%s %v %v: got %v want %v", k, v, b, d.Value, want)
					}
				})
			}
		}
	}
}

func TestEvaluate_Matches(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		pattern string
		val     string
		matches bool
	}{
		{"prefix", "^a", "alpha", true},
		{"prefix-miss", "^a", "beta", false},
		{"suffix", "z$", "buzz", true},
		{"suffix-miss", "z$", "buy", false},
		{"contains", "lph", "alpha", true},
		{"anchored", "^foo$", "foo", true},
		{"anchored-miss", "^foo$", "foobar", false},
		{"escape", `\d+`, "abc123", true},
		{"escape-miss", `\d+`, "abc", false},
		{"unicode", "漢", "漢字", true},
		{"caseless-fail", "FOO", "foo", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			src := fmt.Sprintf(`{
				"value_type":"boolean","default":false,"rules":[
					{"id":"r","when":{"kind":"matches","attr":"k","pattern":%q},"value":true}
				]
			}`, tc.pattern)
			c := mustCompile(t, src)
			d, _ := eval.Evaluate(c, map[string]any{"k": tc.val}, 1)
			if (d.Value == true) != tc.matches {
				t.Errorf("got %v want %v", d.Value, tc.matches)
			}
		})
	}
}

func TestEvaluate_AlwaysAndAllAny(t *testing.T) {
	t.Parallel()
	src := `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r","when":{"kind":"always"},"value":true}
		]
	}`
	c := mustCompile(t, src)
	for i := 0; i < 32; i++ {
		i := i
		t.Run(fmt.Sprintf("always-%d", i), func(t *testing.T) {
			t.Parallel()
			d, _ := eval.Evaluate(c, map[string]any{"i": i}, 1)
			if d.Value != true {
				t.Errorf("always should match")
			}
		})
	}
}

func TestEvaluate_NestedAllAny(t *testing.T) {
	t.Parallel()
	c := mustCompile(t, `{"value_type":"boolean","default":false,"rules":[
		{"id":"r","when":{"kind":"all","of":[
			{"kind":"eq","attr":"a","value":"1"},
			{"kind":"any","of":[
				{"kind":"eq","attr":"b","value":"x"},
				{"kind":"eq","attr":"c","value":"y"}
			]}
		]},"value":true}
	]}`)
	cases := []struct {
		ctx map[string]any
		ok  bool
	}{
		{map[string]any{"a": "1", "b": "x"}, true},
		{map[string]any{"a": "1", "c": "y"}, true},
		{map[string]any{"a": "1", "b": "z", "c": "z"}, false},
		{map[string]any{"a": "2", "b": "x"}, false},
	}
	for i, tc := range cases {
		i, tc := i, tc
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			t.Parallel()
			d, _ := eval.Evaluate(c, tc.ctx, 1)
			if (d.Value == true) != tc.ok {
				t.Errorf("got %v want %v", d.Value, tc.ok)
			}
		})
	}
}

func TestEvaluate_RuleOrderPicksFirst(t *testing.T) {
	t.Parallel()
	src := `{
		"value_type":"string","default":"default","rules":[
			{"id":"r1","when":{"kind":"eq","attr":"k","value":"v"},"value":"first"},
			{"id":"r2","when":{"kind":"eq","attr":"k","value":"v"},"value":"second"}
		]
	}`
	c := mustCompile(t, src)
	d, _ := eval.Evaluate(c, map[string]any{"k": "v"}, 1)
	if d.RuleID != "r1" {
		t.Errorf("RuleID = %q, want r1 (first wins)", d.RuleID)
	}
	if d.Value != "first" {
		t.Errorf("Value = %v", d.Value)
	}
}

func TestEvaluate_VersionPassThrough(t *testing.T) {
	t.Parallel()
	c := mustCompile(t, `{"value_type":"boolean","default":false,"rules":[]}`)
	for i := 0; i < 20; i++ {
		i := i
		t.Run(fmt.Sprintf("v=%d", i), func(t *testing.T) {
			t.Parallel()
			d, _ := eval.Evaluate(c, nil, i)
			if d.Version != i {
				t.Errorf("Version = %d, want %d", d.Version, i)
			}
		})
	}
}

func TestEvaluate_NilCompiledErrors(t *testing.T) {
	t.Parallel()
	if _, err := eval.Evaluate(nil, nil, 1); err == nil {
		t.Errorf("nil Compiled should error")
	}
}

func TestEvaluate_TypeMismatchFallsBackToDefault(t *testing.T) {
	t.Parallel()
	// Wrong value type for the flag's declared value_type: the rule
	// matches, but the value can't be served. Should return default
	// with Reason=type_mismatch.
	c := mustCompile(t, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r","when":{"kind":"always"},"value":"not-a-bool"}
		]
	}`)
	d, _ := eval.Evaluate(c, nil, 1)
	if d.Reason != eval.ReasonTypeMismatch {
		t.Errorf("Reason = %q want %q", d.Reason, eval.ReasonTypeMismatch)
	}
	if d.Value != false {
		t.Errorf("Value = %v want default false", d.Value)
	}
}

func TestEvaluate_WithTraceMirrorsEvaluate(t *testing.T) {
	t.Parallel()
	srcs := []string{
		`{"value_type":"boolean","default":false,"rules":[{"id":"r","when":{"kind":"always"},"value":true}]}`,
		`{"value_type":"boolean","default":false,"rules":[]}`,
		`{"value_type":"string","default":"x","rules":[{"id":"r","when":{"kind":"eq","attr":"k","value":"v"},"value":"y"}]}`,
	}
	for i, src := range srcs {
		i, src := i, src
		t.Run(fmt.Sprintf("src-%d", i), func(t *testing.T) {
			t.Parallel()
			c := mustCompile(t, src)
			ctx := map[string]any{"k": "v"}
			d1, err := eval.Evaluate(c, ctx, 7)
			if err != nil {
				t.Fatal(err)
			}
			d2, _, err := eval.EvaluateWithTrace(c, ctx, 7)
			if err != nil {
				t.Fatal(err)
			}
			if d1.Reason != d2.Reason {
				t.Errorf("reason drift: plain=%q, trace=%q", d1.Reason, d2.Reason)
			}
			if d1.RuleID != d2.RuleID {
				t.Errorf("rule drift: plain=%q, trace=%q", d1.RuleID, d2.RuleID)
			}
		})
	}
}

func TestEvaluate_TraceRecordsEveryRule(t *testing.T) {
	t.Parallel()
	c := mustCompile(t, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r1","when":{"kind":"eq","attr":"k","value":"a"},"value":true},
			{"id":"r2","when":{"kind":"eq","attr":"k","value":"b"},"value":true},
			{"id":"r3","when":{"kind":"eq","attr":"k","value":"c"},"value":true}
		]
	}`)
	_, tr, err := eval.EvaluateWithTrace(c, map[string]any{"k": "b"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.EvaluatedRules) != 3 {
		t.Fatalf("len = %d, want 3 (all rules recorded)", len(tr.EvaluatedRules))
	}
	if tr.MatchedRuleID != "r2" {
		t.Errorf("MatchedRuleID = %q", tr.MatchedRuleID)
	}
	if tr.EvaluatedRules[0].Matched || !tr.EvaluatedRules[1].Matched || tr.EvaluatedRules[2].Matched {
		t.Errorf("matched mask wrong: %+v", tr.EvaluatedRules)
	}
}

func TestEvaluate_TraceNotPredicate(t *testing.T) {
	t.Parallel()
	c := mustCompile(t, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r","when":{"kind":"not","of_one":{"kind":"eq","attr":"k","value":"a"}},"value":true}
		]
	}`)
	for _, tc := range []struct {
		val   string
		match bool
	}{
		{"a", false},
		{"b", true},
		{"", true},
	} {
		tc := tc
		t.Run("val="+tc.val, func(t *testing.T) {
			t.Parallel()
			d, tr, err := eval.EvaluateWithTrace(c, map[string]any{"k": tc.val}, 1)
			if err != nil {
				t.Fatal(err)
			}
			matched := tr.EvaluatedRules[0].Matched
			if matched != tc.match {
				t.Errorf("matched = %v want %v", matched, tc.match)
			}
			if matched && d.Value != true {
				t.Errorf("value = %v", d.Value)
			}
		})
	}
}

func TestEvaluate_DeepLookup(t *testing.T) {
	t.Parallel()
	c := mustCompile(t, `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r","when":{"kind":"eq","attr":"a.b.c.d.e","value":"deep"},"value":true}
		]
	}`)
	ctx := map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": map[string]any{"e": "deep"}}}}}
	d, _ := eval.Evaluate(c, ctx, 1)
	if d.Value != true {
		t.Errorf("deep lookup failed")
	}
}

func TestEvaluate_StringNumberCoercion(t *testing.T) {
	t.Parallel()
	// Numbers may arrive as JSON-decoded float64 or as int/int64 from
	// the SDK. The evaluator tolerates both.
	src := `{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r","when":{"kind":"gt","attr":"age","value":18},"value":true}
		]
	}`
	c := mustCompile(t, src)
	cases := []any{float64(21), 21, int64(21), float32(21)}
	for _, v := range cases {
		v := v
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			t.Parallel()
			d, _ := eval.Evaluate(c, map[string]any{"age": v}, 1)
			if d.Value != true {
				t.Errorf("type=%T did not match gt 18", v)
			}
		})
	}
}
