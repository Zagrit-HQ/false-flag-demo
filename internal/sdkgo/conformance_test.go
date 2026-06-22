package sdkgo_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/depot/falseflag/internal/sdkgo"
)

// TestConformance drives every tests/eval-corpus/*.json fixture through
// the SDK's snapshot-polling client and asserts that the resulting
// Decision matches the fixture's expected output byte-for-byte. This
// is the SDK-level analogue of internal/eval/cross_runtime_test.go —
// it covers the polling, parsing, and Evaluate code paths the
// evaluator-level test never touches.
func TestConformance(t *testing.T) {
	dir := corpusDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read corpus dir: %v", err)
	}

	type corpusFixture struct {
		ID       string          `json:"id"`
		Strategy string          `json:"strategy"`
		IR       json.RawMessage `json:"ir"`
		Context  map[string]any  `json:"context"`
		Expected struct {
			Value  any    `json:"value"`
			Reason string `json:"reason"`
			RuleID string `json:"rule_id,omitempty"`
		} `json:"expected"`
	}

	var fixtures []corpusFixture
	for _, ent := range entries {
		name := ent.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		var fx corpusFixture
		if err := json.Unmarshal(raw, &fx); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		fixtures = append(fixtures, fx)
	}

	if len(fixtures) < 25 {
		t.Fatalf("conformance corpus too small: %d fixtures (want >=25)", len(fixtures))
	}

	// Build a synthetic snapshot containing every fixture as its own
	// flag, keyed by fixture id. Then point a Client at an httptest
	// server returning that snapshot and run Evaluate per fixture.
	flagsMap := make(map[string]json.RawMessage, len(fixtures))
	for _, fx := range fixtures {
		flagsMap[fx.ID] = fx.IR
	}
	snapshotPayload, err := json.Marshal(map[string]any{
		"id":         "conformance-snapshot",
		"project_id": "conformance-project",
		"version":    1,
		"created_at": "2026-05-20T00:00:00Z",
		"compiled":   map[string]any{"flags": flagsMap},
	})
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/snapshots/latest") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write(snapshotPayload)
	}))
	defer srv.Close()

	c, err := sdkgo.NewClient(sdkgo.Options{
		BaseURL:      srv.URL,
		ProjectSlug:  "conformance",
		PollInterval: -1,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Stop()

	if c.Snapshot() == nil {
		t.Fatal("snapshot nil after Start")
	}

	matched := 0
	for _, fx := range fixtures {
		t.Run(fx.ID, func(t *testing.T) {
			got := c.Evaluate(fx.ID, fx.Context)

			gotVal, _ := json.Marshal(got.Value)
			wantVal, _ := json.Marshal(fx.Expected.Value)
			if string(gotVal) != string(wantVal) {
				t.Errorf("value: got %s, want %s", gotVal, wantVal)
			}
			if got.Reason != fx.Expected.Reason {
				t.Errorf("reason: got %q, want %q", got.Reason, fx.Expected.Reason)
			}
			if fx.Expected.RuleID != "" && got.RuleID != fx.Expected.RuleID {
				t.Errorf("rule_id: got %q, want %q", got.RuleID, fx.Expected.RuleID)
			}
		})
		matched++
	}
	fmt.Printf("conformance: %d/%d fixtures match\n", matched, len(fixtures))
}

func corpusDir(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(here), "..", "..", "tests", "eval-corpus")
}
