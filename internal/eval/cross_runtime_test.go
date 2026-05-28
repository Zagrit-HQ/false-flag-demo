package eval_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/eval"
)

// fixture is the JSON shape at tests/eval-corpus/*.json. The same
// shape is consumed by js/packages/shared-eval-corpus/.
type fixture struct {
	ID          string          `json:"id"`
	Description string          `json:"description"`
	Strategy    config.Strategy `json:"strategy"`
	// IR is the pre-compiled IR JSON. For json/cel strategies this is
	// what gets handed to Compile. For typescript flags it serves as
	// the byte-identical reference the server-side compile must
	// reproduce from SourceText.
	IR json.RawMessage `json:"ir"`
	// SourceText is the raw author input (e.g. .ts file contents).
	// When present and Strategy == typescript, fixture drivers compile
	// from SourceText to verify server-side TS compilation.
	SourceText string          `json:"source_text,omitempty"`
	Context    map[string]any  `json:"context"`
	Expected   fixtureExpected `json:"expected"`
	Extra      map[string]any  `json:"-"`
}

type fixtureExpected struct {
	Value  any    `json:"value"`
	Reason string `json:"reason"`
	RuleID string `json:"rule_id,omitempty"`
}

// corpusDir resolves tests/eval-corpus/ relative to this test file,
// which is more robust than relying on `go test`'s working directory.
func corpusDir(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(here), "..", "..", "tests", "eval-corpus")
}

func TestCrossRuntimeCorpus(t *testing.T) {
	dir := corpusDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read corpus dir: %v", err)
	}
	count := 0
	for _, ent := range entries {
		name := ent.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		path := filepath.Join(dir, name)
		t.Run(name, func(t *testing.T) {
			runFixture(t, path)
		})
		count++
	}
	if count < 12 {
		t.Fatalf("corpus too small: %d fixtures (want ≥12). Add more to tests/eval-corpus/.", count)
	}
}

func runFixture(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fx fixture
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if fx.ID == "" {
		t.Fatal("fixture missing id")
	}
	if !fx.Strategy.Valid() {
		t.Fatalf("fixture %q: invalid strategy %q", fx.ID, fx.Strategy)
	}
	if len(fx.IR) == 0 {
		t.Fatalf("fixture %q: missing ir", fx.ID)
	}

	// For typescript fixtures, prefer the raw source_text — that's
	// what the server compiles in production. Fall back to the IR
	// (legacy fixture shape) so older corpus entries still work.
	compileInput := fx.IR
	if fx.Strategy == config.StrategyTypeScript && fx.SourceText != "" {
		compileInput = []byte(fx.SourceText)
	}
	compiled, err := config.Compile(fx.Strategy, compileInput)
	if err != nil {
		t.Fatalf("fixture %q compile: %v", fx.ID, err)
	}
	got, err := eval.Evaluate(compiled, fx.Context, 1)
	if err != nil {
		t.Fatalf("fixture %q eval: %v", fx.ID, err)
	}

	// Compare value via JSON-round-trip to absorb int/float and
	// map-key-ordering noise.
	gotVal, _ := json.Marshal(got.Value)
	wantVal, _ := json.Marshal(fx.Expected.Value)
	if string(gotVal) != string(wantVal) {
		t.Errorf("fixture %q value: got %s, want %s", fx.ID, gotVal, wantVal)
	}
	if got.Reason != fx.Expected.Reason {
		t.Errorf("fixture %q reason: got %q, want %q", fx.ID, got.Reason, fx.Expected.Reason)
	}
	if got.RuleID != fx.Expected.RuleID {
		t.Errorf("fixture %q rule_id: got %q, want %q", fx.ID, got.RuleID, fx.Expected.RuleID)
	}
}
