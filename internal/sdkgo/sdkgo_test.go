package sdkgo_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/depot/falseflag/internal/eval"
	"github.com/depot/falseflag/internal/sdkgo"
)

// fakeSnapshot is the JSON the API returns from
// /v1/projects/{slug}/snapshots/latest.
const fakeSnapshot = `{
  "id": "11111111-1111-1111-1111-111111111111",
  "project_id": "22222222-2222-2222-2222-222222222222",
  "version": 7,
  "created_at": "2026-05-20T12:00:00Z",
  "compiled": {
    "flags": {
      "checkout-redesign": {
        "value_type": "boolean",
        "default": false,
        "rules": [
          {
            "id": "r1",
            "when": { "kind": "eq", "attr": "user.plan", "value": "pro" },
            "value": true
          }
        ]
      },
      "max-items": {
        "value_type": "number",
        "default": 10,
        "rules": []
      },
      "greeting": {
        "value_type": "string",
        "default": "hello",
        "rules": []
      },
      "badges": {
        "value_type": "object",
        "default": { "enabled": false },
        "rules": []
      }
    }
  }
}`

func snapshotServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/snapshots/latest") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("content-type", "application/json")
		_, _ = io.WriteString(w, fakeSnapshot)
	}))
}

func mustClient(t *testing.T, baseURL string) *sdkgo.Client {
	t.Helper()
	c, err := sdkgo.NewClient(sdkgo.Options{
		BaseURL:      baseURL,
		ProjectSlug:  "demo",
		PollInterval: -1,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestNewClient_Validation(t *testing.T) {
	t.Parallel()
	if _, err := sdkgo.NewClient(sdkgo.Options{ProjectSlug: "demo"}); err == nil {
		t.Fatalf("expected error for empty BaseURL")
	}
	if _, err := sdkgo.NewClient(sdkgo.Options{BaseURL: "http://x"}); err == nil {
		t.Fatalf("expected error for empty ProjectSlug")
	}
}

func TestStart_LoadsSnapshot(t *testing.T) {
	t.Parallel()
	srv := snapshotServer(t)
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Stop()

	snap := c.Snapshot()
	if snap == nil {
		t.Fatalf("snapshot nil after Start")
	}
	if snap.Version != 7 {
		t.Errorf("version = %d, want 7", snap.Version)
	}
	if _, ok := snap.Flags["checkout-redesign"]; !ok {
		t.Errorf("missing flag in snapshot")
	}
}

func TestEvaluate_Snapshot(t *testing.T) {
	t.Parallel()
	srv := snapshotServer(t)
	defer srv.Close()

	c := mustClient(t, srv.URL)
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Stop()

	d := c.Evaluate("checkout-redesign", sdkgo.EvalContext{
		"user": map[string]any{"plan": "pro"},
	})
	if d.Value != true {
		t.Errorf("value = %v, want true", d.Value)
	}
	if d.Reason != eval.ReasonRuleMatched {
		t.Errorf("reason = %s, want rule_matched", d.Reason)
	}

	d2 := c.Evaluate("max-items", sdkgo.EvalContext{})
	v, ok := d2.Value.(float64)
	if !ok || v != 10 {
		t.Errorf("default number = %v, want 10", d2.Value)
	}

	d3 := c.Evaluate("unknown-flag", sdkgo.EvalContext{})
	if d3.Reason != eval.ReasonDefault {
		t.Errorf("missing flag reason = %s, want default", d3.Reason)
	}
}

func TestEvaluate_NoSnapshot(t *testing.T) {
	t.Parallel()
	c := mustClient(t, "http://invalid.example")
	d := c.Evaluate("any", sdkgo.EvalContext{})
	if d.Reason != eval.ReasonError {
		t.Errorf("reason = %s, want error", d.Reason)
	}
}

func TestProvider_BooleanEvaluation(t *testing.T) {
	t.Parallel()
	srv := snapshotServer(t)
	defer srv.Close()
	c := mustClient(t, srv.URL)
	_ = c.Start(context.Background())
	defer c.Stop()
	p := sdkgo.NewProvider(c, "")
	if p.Metadata().Name != "falseflag" {
		t.Errorf("default name = %q, want falseflag", p.Metadata().Name)
	}

	d := p.BooleanEvaluation(context.Background(), "checkout-redesign", false, sdkgo.EvalContext{
		"user": map[string]any{"plan": "pro"},
	})
	if d.Value != true {
		t.Errorf("value = %v, want true", d.Value)
	}

	// type mismatch (asking for bool from a number flag) → default
	d2 := p.BooleanEvaluation(context.Background(), "max-items", true, sdkgo.EvalContext{})
	if d2.Value != true {
		t.Errorf("type mismatch value = %v, want true (default)", d2.Value)
	}
	if d2.Reason != eval.ReasonTypeMismatch {
		t.Errorf("type mismatch reason = %s, want type_mismatch", d2.Reason)
	}
}

func TestProvider_StringNumberObject(t *testing.T) {
	t.Parallel()
	srv := snapshotServer(t)
	defer srv.Close()
	c := mustClient(t, srv.URL)
	_ = c.Start(context.Background())
	defer c.Stop()
	p := sdkgo.NewProvider(c, "demo-provider")

	if got := p.StringEvaluation(context.Background(), "greeting", "x", sdkgo.EvalContext{}); got.Value != "hello" {
		t.Errorf("string = %v, want hello", got.Value)
	}
	if got := p.NumberEvaluation(context.Background(), "max-items", -1, sdkgo.EvalContext{}); got.Value != 10.0 {
		t.Errorf("number = %v, want 10", got.Value)
	}
	obj := p.ObjectEvaluation(context.Background(), "badges", map[string]any{}, sdkgo.EvalContext{})
	m, ok := obj.Value.(map[string]any)
	if !ok {
		t.Fatalf("object = %T, want map", obj.Value)
	}
	if m["enabled"] != false {
		t.Errorf("object.enabled = %v, want false", m["enabled"])
	}
}

func TestPolling_LastGoodOnError(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.Header().Set("content-type", "application/json")
			_, _ = io.WriteString(w, fakeSnapshot)
			return
		}
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, err := sdkgo.NewClient(sdkgo.Options{
		BaseURL:      srv.URL,
		ProjectSlug:  "demo",
		PollInterval: 25 * time.Millisecond,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Stop()

	// Wait for at least one polling attempt past initial Start.
	deadline := time.Now().Add(2 * time.Second)
	for calls.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	snap := c.Snapshot()
	if snap == nil || snap.Version != 7 {
		t.Errorf("expected last-good snapshot v7, got %+v", snap)
	}
	if calls.Load() < 2 {
		t.Errorf("expected >=2 poll calls, got %d", calls.Load())
	}
}

func TestPoll_404IsNotError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := mustClient(t, srv.URL)
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer c.Stop()
	if c.Snapshot() != nil {
		t.Errorf("expected nil snapshot when API has none")
	}
}
