// Slow load-style eval tests intended to make CI walltime visible.
// They cover credible "real" scenarios (large rule lists, deep
// predicates, mixed rollout sampling) — the slowness comes from
// running each case at N×, not from artificial sleeps.
//
// These tests skip under `go test -short`.

package eval_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/eval"
)

func skipShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping slow eval test in -short mode")
	}
}

// TestSlow_RolloutDistribution_Many asserts that across 50k samples
// the in-bucket count is within tolerance for a few percent gates.
// Intentionally large sample to make a single test ~hundreds of ms.
func TestSlow_RolloutDistribution_Many(t *testing.T) {
	skipShort(t)
	t.Parallel()
	for _, percent := range []int{5, 25, 50, 75, 95} {
		percent := percent
		t.Run(fmt.Sprintf("p=%d", percent), func(t *testing.T) {
			t.Parallel()
			src := fmt.Sprintf(`{
				"value_type":"boolean","default":false,"rules":[
					{"id":"r","when":{"kind":"rollout","attr":"id","salt":"f","percent":%d},"value":true}
				]
			}`, percent)
			c, err := config.Compile(config.StrategyJSON, []byte(src))
			if err != nil {
				t.Fatal(err)
			}
			n := 50_000
			in := 0
			for i := 0; i < n; i++ {
				d, _ := eval.Evaluate(c, map[string]any{"id": fmt.Sprintf("u-%d", i)}, 1)
				if d.Value == true {
					in++
				}
			}
			got := float64(in) / float64(n) * 100
			if diff := abs(got - float64(percent)); diff > 3 {
				t.Errorf("percent=%d observed %.1f%% (diff %.1f) over %d samples", percent, got, diff, n)
			}
		})
	}
}

// TestSlow_LargeRulesList_Evaluate compiles and evaluates a flag
// with many rules so the linear scan + JSON decode cost shows up
// in the test runtime.
func TestSlow_LargeRulesList_Evaluate(t *testing.T) {
	skipShort(t)
	t.Parallel()
	for _, n := range []int{100, 500, 1000, 2000} {
		n := n
		t.Run(fmt.Sprintf("rules=%d", n), func(t *testing.T) {
			t.Parallel()
			var rules []string
			for i := 0; i < n; i++ {
				rules = append(rules, fmt.Sprintf(
					`{"id":"r%d","when":{"kind":"eq","attr":"k","value":"v%d"},"value":true}`,
					i, i,
				))
			}
			src := fmt.Sprintf(`{"value_type":"boolean","default":false,"rules":[%s]}`, strings.Join(rules, ","))
			c, err := config.Compile(config.StrategyJSON, []byte(src))
			if err != nil {
				t.Fatal(err)
			}
			// 1000 evaluations through n rules each.
			for i := 0; i < 1000; i++ {
				_, _ = eval.Evaluate(c, map[string]any{"k": fmt.Sprintf("v%d", i%n)}, 1)
			}
		})
	}
}

// TestSlow_DeeplyNestedPredicate exercises the recursive matcher.
// Several depths run in parallel so the suite hits real walltime in
// one shard but parallelises across them.
func TestSlow_DeeplyNestedPredicate(t *testing.T) {
	skipShort(t)
	t.Parallel()
	for _, depth := range []int{10, 25, 50, 75, 100} {
		depth := depth
		t.Run(fmt.Sprintf("depth=%d", depth), func(t *testing.T) {
			t.Parallel()
			pred := `{"kind":"always"}`
			for i := 0; i < depth; i++ {
				pred = fmt.Sprintf(`{"kind":"all","of":[%s]}`, pred)
			}
			src := fmt.Sprintf(`{"value_type":"boolean","default":false,"rules":[
				{"id":"r","when":%s,"value":true}
			]}`, pred)
			c, err := config.Compile(config.StrategyJSON, []byte(src))
			if err != nil {
				t.Fatalf("depth=%d compile: %v", depth, err)
			}
			for i := 0; i < 200; i++ {
				d, _ := eval.Evaluate(c, nil, 1)
				if d.Value != true {
					t.Errorf("nested always should match")
				}
			}
		})
	}
}

// TestSlow_TraceLargeRules builds traces of many-rule evaluations.
// EvaluateWithTrace must visit every rule regardless of an early
// match, so this stresses the trace allocator and node builders.
func TestSlow_TraceLargeRules(t *testing.T) {
	skipShort(t)
	t.Parallel()
	for _, n := range []int{50, 200, 500} {
		n := n
		t.Run(fmt.Sprintf("rules=%d", n), func(t *testing.T) {
			t.Parallel()
			var rules []string
			for i := 0; i < n; i++ {
				rules = append(rules, fmt.Sprintf(
					`{"id":"r%d","when":{"kind":"in","attr":"k","values":["a","b","c","d","e"]},"value":true}`,
					i,
				))
			}
			src := fmt.Sprintf(`{"value_type":"boolean","default":false,"rules":[%s]}`, strings.Join(rules, ","))
			c, _ := config.Compile(config.StrategyJSON, []byte(src))
			for i := 0; i < 100; i++ {
				_, tr, err := eval.EvaluateWithTrace(c, map[string]any{"k": "z"}, 1)
				if err != nil {
					t.Fatal(err)
				}
				if len(tr.EvaluatedRules) != n {
					t.Errorf("got %d rules want %d", len(tr.EvaluatedRules), n)
				}
			}
		})
	}
}

// TestSlow_ParallelEvaluators is a long-running concurrency check
// that runs many goroutines hammering a single Compiled. There is
// nothing to assert other than absence of races (which `go test -race`
// catches) and deterministic results.
func TestSlow_ParallelEvaluators(t *testing.T) {
	skipShort(t)
	t.Parallel()
	c, _ := config.Compile(config.StrategyJSON, []byte(`{
		"value_type":"boolean","default":false,"rules":[
			{"id":"r","when":{"kind":"rollout","attr":"id","salt":"f","percent":50},"value":true}
		]
	}`))
	for w := 0; w < 8; w++ {
		w := w
		t.Run(fmt.Sprintf("worker-%d", w), func(t *testing.T) {
			t.Parallel()
			start := time.Now()
			for i := 0; i < 20_000; i++ {
				_, _ = eval.Evaluate(c, map[string]any{"id": fmt.Sprintf("u-%d-%d", w, i)}, 1)
			}
			_ = time.Since(start) // signal to the human reading verbose output
		})
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
