package config

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestTypescriptCompileHappyPath(t *testing.T) {
	src := []byte(`
import { FalseFlag as ff } from "@falseflag/config";

export default ff.flag({
  value_type: "object",
  default: { mode: "light" },
  rules: [
    ff.rule("r-dark", ff.eq("user.prefers", "\"dark\""), { mode: "dark" }),
  ],
});
`)
	c, err := compileTypescript(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Strategy != StrategyTypeScript {
		t.Errorf("Strategy = %q, want typescript", c.Strategy)
	}
	if c.IR.ValueType != ValueTypeObject {
		t.Errorf("ValueType = %q, want object", c.IR.ValueType)
	}
	if len(c.IR.Rules) != 1 {
		t.Fatalf("Rules len = %d, want 1", len(c.IR.Rules))
	}
	if c.IR.Rules[0].ID != "r-dark" {
		t.Errorf("Rules[0].ID = %q, want r-dark", c.IR.Rules[0].ID)
	}
	if c.IR.Rules[0].When.Kind != PredEq {
		t.Errorf("Rules[0].When.Kind = %q, want eq", c.IR.Rules[0].When.Kind)
	}
}

func TestTypescriptCompileAllBuilders(t *testing.T) {
	// Exercises every shim builder so the JS shim and the TS package
	// can't drift silently. Keep this in sync with
	// js/packages/config-ts/src/index.ts.
	src := []byte(`
import { FalseFlag as ff } from "@falseflag/config";

export default ff.flag({
  value_type: "boolean",
  default: false,
  rules: [
    ff.rule("r-all",
      ff.all(
        ff.eq("a", "1"),
        ff.neq("b", "2"),
        ff.in("c", ["x", "y"]),
        ff.gt("d", 3),
        ff.gte("e", 4),
        ff.lt("f", 5),
        ff.lte("g", 6),
        ff.matches("h", "^x"),
        ff.rollout("i", "salt", 50),
        ff.any(ff.always(), ff.not(ff.always())),
        ff.cel("ctx.x == 'y'"),
      ),
      true),
  ],
});
`)
	c, err := compileTypescript(context.Background(), src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.IR.Rules[0].When.Kind != PredAll {
		t.Fatalf("outer kind = %q, want all", c.IR.Rules[0].When.Kind)
	}
	kinds := make([]string, 0, len(c.IR.Rules[0].When.Of))
	for _, p := range c.IR.Rules[0].When.Of {
		kinds = append(kinds, string(p.Kind))
	}
	wantKinds := []string{"eq", "neq", "in", "gt", "gte", "lt", "lte",
		"matches", "rollout", "any", "cel"}
	if len(kinds) != len(wantKinds) {
		t.Fatalf("kinds = %v, want %v", kinds, wantKinds)
	}
	for i, k := range kinds {
		if k != wantKinds[i] {
			t.Errorf("kinds[%d] = %q, want %q", i, k, wantKinds[i])
		}
	}
}

func TestTypescriptCompileSyntaxError(t *testing.T) {
	// Intentional TS parse error.
	src := []byte(`import { FalseFlag } from "@falseflag/config"; export default flag({`)
	_, err := compileTypescript(context.Background(), src)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrTypeScriptCompileFailure) {
		t.Errorf("err = %v, want ErrTypeScriptCompileFailure", err)
	}
	var ee *EsbuildError
	if !errors.As(err, &ee) {
		t.Fatalf("want *EsbuildError, got %T", err)
	}
	if len(ee.Messages) == 0 {
		t.Fatal("EsbuildError has no messages")
	}
	if ee.Messages[0].Line == 0 {
		t.Errorf("expected line > 0, got %d", ee.Messages[0].Line)
	}
}

func TestTypescriptCompileSandboxBlocksRequire(t *testing.T) {
	// Bypass esbuild's tree-shaking by calling require() directly.
	// `fs` is a Node builtin, so esbuild leaves the call intact under
	// platform=node. The goja require() stub must throw.
	src := []byte(`
import { FalseFlag } from "@falseflag/config";
declare const require: any;
const stolen = require("fs");
export default FalseFlag.flag({
  value_type: "boolean",
  default: false,
  rules: [FalseFlag.rule("r", FalseFlag.always(), stolen.readFileSync ? true : false)],
});
`)
	_, err := compileTypescript(context.Background(), src)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrTypeScriptCompileFailure) {
		t.Errorf("err = %v, want ErrTypeScriptCompileFailure", err)
	}
	if !strings.Contains(err.Error(), "require() not allowed") &&
		!strings.Contains(err.Error(), "fs") {
		t.Errorf("err message should mention the denied module: %v", err)
	}
}

func TestTypescriptCompileInterruptsInfiniteLoop(t *testing.T) {
	src := []byte(`
import { FalseFlag } from "@falseflag/config";
while (true) { /* spin */ }
export default FalseFlag.flag({ value_type: "boolean", default: false, rules: [] });
`)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := compileTypescript(ctx, src)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrTypeScriptCompileFailure) {
		t.Errorf("err = %v, want ErrTypeScriptCompileFailure", err)
	}
	// 1s wall clock is the slow fallback; the 50ms context should
	// fire much sooner. Allow generous slack for CI.
	if elapsed > 500*time.Millisecond {
		t.Errorf("interrupt took %v; expected < 500ms via ctx", elapsed)
	}
}

func TestTypescriptCompileMissingDefault(t *testing.T) {
	src := []byte(`
import { FalseFlag } from "@falseflag/config";
const _ = FalseFlag.flag({ value_type: "boolean", default: false, rules: [] });
`)
	_, err := compileTypescript(context.Background(), src)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrTypeScriptCompileFailure) {
		t.Errorf("err = %v, want ErrTypeScriptCompileFailure", err)
	}
}

func TestTypescriptCompileInvalidIR(t *testing.T) {
	// User authored a flag with an unknown value_type — passes through
	// goja fine, fails IR validation.
	src := []byte(`
import { FalseFlag } from "@falseflag/config";
export default { value_type: "rocket", default: false, rules: [] };
`)
	_, err := compileTypescript(context.Background(), src)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidValueType) {
		t.Errorf("err = %v, want ErrInvalidValueType", err)
	}
}

func TestTypescriptCompileEmptySource(t *testing.T) {
	_, err := compileTypescript(context.Background(), []byte(""))
	if !errors.Is(err, ErrTypeScriptCompileFailure) {
		t.Errorf("err = %v, want ErrTypeScriptCompileFailure", err)
	}
}

func TestTypescriptCompileOversizedSource(t *testing.T) {
	big := make([]byte, maxSourceTextBytes+1)
	for i := range big {
		big[i] = ' '
	}
	_, err := compileTypescript(context.Background(), big)
	if !errors.Is(err, ErrTypeScriptCompileFailure) {
		t.Errorf("err = %v, want ErrTypeScriptCompileFailure", err)
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("want size-limit message, got %v", err)
	}
}

func TestTypescriptCompileViaCompilerFor(t *testing.T) {
	// The public entry point — CompilerFor / Compile — must keep working.
	src := []byte(`
import { FalseFlag } from "@falseflag/config";
export default FalseFlag.flag({ value_type: "boolean", default: false, rules: [] });
`)
	c, err := Compile(StrategyTypeScript, src)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if c.Strategy != StrategyTypeScript {
		t.Errorf("Strategy = %q, want typescript", c.Strategy)
	}
}

func TestTypescriptCompileIRBytesShape(t *testing.T) {
	// Sanity-check that the JSON marshal of the goja-extracted IR
	// matches the JSON the CLI would have produced. We don't assert
	// byte-identical here (that's the conformance test's job); we
	// assert that round-tripping through json.Marshal/Unmarshal lands
	// in the same RulesTree shape with the same predicate kinds.
	src := []byte(`
import { FalseFlag as ff } from "@falseflag/config";
export default ff.flag({
  value_type: "string",
  default: "off",
  rules: [
    ff.rule("r-segment", ff.eq("user.country", "\"US\""), "us"),
  ],
});
`)
	c, err := compileTypescript(context.Background(), src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	roundTrip, err := json.Marshal(c.IR)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got RulesTree
	if err := json.Unmarshal(roundTrip, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ValueType != ValueTypeString {
		t.Errorf("ValueType = %q, want string", got.ValueType)
	}
	if len(got.Rules) != 1 || got.Rules[0].ID != "r-segment" {
		t.Errorf("Rules = %+v", got.Rules)
	}
}
