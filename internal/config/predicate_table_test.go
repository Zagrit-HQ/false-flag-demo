package config_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/depot/falseflag/internal/config"
)

func TestValidatePredicate_Valid(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"always":      `{"kind":"always"}`,
		"eq":          `{"kind":"eq","attr":"k","value":"v"}`,
		"neq":         `{"kind":"neq","attr":"k","value":"v"}`,
		"in":          `{"kind":"in","attr":"k","values":["a","b"]}`,
		"gt":          `{"kind":"gt","attr":"x","value":10}`,
		"gte":         `{"kind":"gte","attr":"x","value":10}`,
		"lt":          `{"kind":"lt","attr":"x","value":10}`,
		"lte":         `{"kind":"lte","attr":"x","value":10}`,
		"matches":     `{"kind":"matches","attr":"k","pattern":"^x"}`,
		"rollout":     `{"kind":"rollout","attr":"user.id","salt":"flag","percent":50}`,
		"all":         `{"kind":"all","of":[{"kind":"always"}]}`,
		"any":         `{"kind":"any","of":[{"kind":"always"}]}`,
		"not":         `{"kind":"not","of_one":{"kind":"always"}}`,
		"nested-all":  `{"kind":"all","of":[{"kind":"eq","attr":"a","value":1},{"kind":"any","of":[{"kind":"always"}]}]}`,
		"rollout-0":   `{"kind":"rollout","attr":"k","salt":"f","percent":0}`,
		"rollout-100": `{"kind":"rollout","attr":"k","salt":"f","percent":100}`,
	}
	for name, raw := range cases {
		name, raw := name, raw
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var p config.Predicate
			if err := json.Unmarshal([]byte(raw), &p); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if err := config.ValidatePredicate(&p, false); err != nil {
				t.Errorf("ValidatePredicate(%s) = %v", raw, err)
			}
		})
	}
}

func TestValidatePredicate_Invalid(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"unknown-kind":       `{"kind":"xyz"}`,
		"eq-no-attr":         `{"kind":"eq","value":"v"}`,
		"eq-no-value":        `{"kind":"eq","attr":"k"}`,
		"in-no-values":       `{"kind":"in","attr":"k","values":[]}`,
		"matches-no-pattern": `{"kind":"matches","attr":"k"}`,
		"rollout-negative":   `{"kind":"rollout","attr":"k","salt":"f","percent":-5}`,
		"rollout-too-big":    `{"kind":"rollout","attr":"k","salt":"f","percent":150}`,
		"all-empty":          `{"kind":"all","of":[]}`,
		"any-empty":          `{"kind":"any","of":[]}`,
		"not-missing":        `{"kind":"not"}`,
	}
	for name, raw := range cases {
		name, raw := name, raw
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var p config.Predicate
			if err := json.Unmarshal([]byte(raw), &p); err != nil {
				// Some inputs may even fail to unmarshal; if so, the
				// test is still valid (no predicate can be built).
				return
			}
			if err := config.ValidatePredicate(&p, false); err == nil {
				t.Errorf("ValidatePredicate(%s) should fail", raw)
			}
		})
	}
}

func TestValidatePredicate_CELRequiresFlag(t *testing.T) {
	t.Parallel()
	raw := `{"kind":"cel","source":"user.id == 'a'"}`
	var p config.Predicate
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := config.ValidatePredicate(&p, false); err == nil {
		t.Errorf("CEL predicate should fail without allowCEL flag")
	}
	if err := config.ValidatePredicate(&p, true); err != nil {
		t.Errorf("CEL predicate should validate with allowCEL: %v", err)
	}
}

func TestCompileJSON_BoundaryRolloutPercents(t *testing.T) {
	t.Parallel()
	for p := 0; p <= 100; p += 10 {
		p := p
		t.Run(fmt.Sprintf("percent=%d", p), func(t *testing.T) {
			t.Parallel()
			src := fmt.Sprintf(`{
				"value_type":"boolean","default":false,"rules":[
					{"id":"r","when":{"kind":"rollout","attr":"user.id","salt":"s","percent":%d},"value":true}
				]
			}`, p)
			if _, err := config.Compile(config.StrategyJSON, []byte(src)); err != nil {
				t.Errorf("percent %d should compile: %v", p, err)
			}
		})
	}
}

func TestCompileJSON_LargeRules(t *testing.T) {
	t.Parallel()
	// A flag with many rules must compile and validate without
	// quadratic-blowup or stack overflow.
	for _, n := range []int{1, 10, 50, 200, 500} {
		n := n
		t.Run(fmt.Sprintf("n=%d", n), func(t *testing.T) {
			t.Parallel()
			var rules []string
			for i := 0; i < n; i++ {
				rules = append(rules, fmt.Sprintf(`{"id":"r%d","when":{"kind":"eq","attr":"k","value":"v%d"},"value":true}`, i, i))
			}
			src := fmt.Sprintf(`{"value_type":"boolean","default":false,"rules":[%s]}`, strings.Join(rules, ","))
			if _, err := config.Compile(config.StrategyJSON, []byte(src)); err != nil {
				t.Errorf("compile n=%d: %v", n, err)
			}
		})
	}
}

func TestCompileJSON_MalformedSources(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"empty":           ``,
		"not-json":        `not json`,
		"missing-default": `{"value_type":"boolean","rules":[]}`,
		"missing-type":    `{"default":false,"rules":[]}`,
		"bad-value-type":  `{"value_type":"date","default":"x","rules":[]}`,
		"rule-no-id":      `{"value_type":"boolean","default":false,"rules":[{"when":{"kind":"always"},"value":true}]}`,
		"rule-no-when":    `{"value_type":"boolean","default":false,"rules":[{"id":"r","value":true}]}`,
		"rule-no-value":   `{"value_type":"boolean","default":false,"rules":[{"id":"r","when":{"kind":"always"}}]}`,
	}
	for name, raw := range cases {
		name, raw := name, raw
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := config.Compile(config.StrategyJSON, []byte(raw)); err == nil {
				t.Errorf("compile(%q) should error", raw)
			}
		})
	}
}
