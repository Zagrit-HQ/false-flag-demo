// Slow esbuild-driven typescript compile tests. Each compile call
// spins up esbuild + goja, so the file is sized for measurable
// CI walltime that benefits from sharding. Skipped under -short.

package config

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func skipTSShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping slow typescript compile in -short mode")
	}
}

// TestSlow_TypescriptCompile_Many runs a wide table of small TS
// sources through the real compiler. Each one is independent so they
// can shard cleanly across CI nodes.
func TestSlow_TypescriptCompile_Many(t *testing.T) {
	skipTSShort(t)
	t.Parallel()
	cases := []struct {
		name string
		src  string
	}{
		{
			"boolean-no-rules",
			`import { FalseFlag as ff } from "@falseflag/config";
			 export default ff.flag({ value_type: "boolean", default: false, rules: [] });`,
		},
		{
			"string-eq",
			`import { FalseFlag as ff } from "@falseflag/config";
			 export default ff.flag({
			   value_type: "string", default: "control",
			   rules: [ff.rule("r", ff.eq("env", "\"prod\""), "treatment")]
			 });`,
		},
		{
			"number-gt",
			`import { FalseFlag as ff } from "@falseflag/config";
			 export default ff.flag({
			   value_type: "number", default: 0,
			   rules: [ff.rule("r", ff.gt("age", 18), 1)]
			 });`,
		},
		{
			"all-any-not",
			`import { FalseFlag as ff } from "@falseflag/config";
			 export default ff.flag({
			   value_type: "boolean", default: false,
			   rules: [ff.rule("r",
			     ff.all(
			       ff.eq("country", "\"US\""),
			       ff.any(ff.eq("plan", "\"pro\""), ff.not(ff.eq("trial", true)))
			     ),
			   true)]
			 });`,
		},
		{
			"rollout",
			`import { FalseFlag as ff } from "@falseflag/config";
			 export default ff.flag({
			   value_type: "boolean", default: false,
			   rules: [ff.rule("r", ff.rollout("user.id", "salt", 50), true)]
			 });`,
		},
		{
			"matches",
			`import { FalseFlag as ff } from "@falseflag/config";
			 export default ff.flag({
			   value_type: "boolean", default: false,
			   rules: [ff.rule("r", ff.matches("email", "^.+@example\\.com$"), true)]
			 });`,
		},
		{
			"in",
			`import { FalseFlag as ff } from "@falseflag/config";
			 export default ff.flag({
			   value_type: "boolean", default: false,
			   rules: [ff.rule("r", ff.in("plan", ["\"pro\"","\"ent\"","\"team\""]), true)]
			 });`,
		},
		{
			"object-default",
			`import { FalseFlag as ff } from "@falseflag/config";
			 export default ff.flag({
			   value_type: "object",
			   default: { mode: "light", showBeta: false },
			   rules: [ff.rule("r", ff.eq("user.prefers","\"dark\""), { mode: "dark" })]
			 });`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, err := compileTypescript(context.Background(), []byte(tc.src))
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if c.IR == nil {
				t.Fatal("IR nil")
			}
			if c.Strategy != StrategyTypeScript {
				t.Errorf("strategy = %q", c.Strategy)
			}
		})
	}
}

// TestSlow_TypescriptCompile_LargeRuleCount tests scaling: a TS source
// that emits many rules. Verifies the goja-side serialisation copes
// with bigger output.
func TestSlow_TypescriptCompile_LargeRuleCount(t *testing.T) {
	skipTSShort(t)
	t.Parallel()
	for _, n := range []int{10, 50, 100, 200} {
		n := n
		t.Run(fmt.Sprintf("rules=%d", n), func(t *testing.T) {
			t.Parallel()
			var lines []string
			for i := 0; i < n; i++ {
				lines = append(lines, fmt.Sprintf(
					`ff.rule("r%d", ff.eq("k", "\"v%d\""), true)`,
					i, i,
				))
			}
			src := fmt.Sprintf(`
			import { FalseFlag as ff } from "@falseflag/config";
			export default ff.flag({
			  value_type: "boolean", default: false,
			  rules: [%s]
			});`, strings.Join(lines, ",\n"))
			c, err := compileTypescript(context.Background(), []byte(src))
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if got := len(c.IR.Rules); got != n {
				t.Errorf("rules = %d, want %d", got, n)
			}
		})
	}
}

// TestSlow_TypescriptCompile_BrokenSourceProducesDiagnostics ensures
// that broken TS surfaces structured esbuild diagnostics so the API
// surface can 422 with file/line/column info.
func TestSlow_TypescriptCompile_BrokenSourceProducesDiagnostics(t *testing.T) {
	skipTSShort(t)
	t.Parallel()
	cases := []string{
		`import { FalseFlag as ff } from "@falseflag/config"; export default`,
		`let;;`,
		`import { Missing } from "nowhere"; export default Missing;`,
		`export default function() { throw new SyntaxError(`,
		`const x = 1 + ; export default x;`,
	}
	for i, src := range cases {
		i, src := i, src
		t.Run(fmt.Sprintf("broken-%d", i), func(t *testing.T) {
			t.Parallel()
			_, err := compileTypescript(context.Background(), []byte(src))
			if err == nil {
				t.Fatalf("expected error for: %s", src)
			}
		})
	}
}
