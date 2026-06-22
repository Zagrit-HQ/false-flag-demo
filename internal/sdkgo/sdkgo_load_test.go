// Many-cases load tests for the Go SDK. Skipped under -short.

package sdkgo_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/depot/falseflag/internal/sdkgo"
)

func slowSkip(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping slow sdk test in -short mode")
	}
}

func snapshotServerBody(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func buildSnapshot(flags int) []byte {
	compiled := map[string]any{"flags": map[string]any{}}
	cf := compiled["flags"].(map[string]any)
	for i := 0; i < flags; i++ {
		cf[fmt.Sprintf("flag-%d", i)] = json.RawMessage(fmt.Sprintf(`{
			"value_type":"boolean","default":false,
			"rules":[{"id":"r","when":{"kind":"eq","attr":"k","value":"v%d"},"value":true}]
		}`, i))
	}
	body, _ := json.Marshal(map[string]any{
		"id":         "11111111-1111-1111-1111-111111111111",
		"project_id": "22222222-2222-2222-2222-222222222222",
		"version":    1,
		"created_at": "2026-05-20T12:00:00Z",
		"compiled":   compiled,
	})
	return body
}

func newStartedClient(t *testing.T, body []byte) *sdkgo.Client {
	t.Helper()
	srv := snapshotServerBody(t, body)
	c, err := sdkgo.NewClient(sdkgo.Options{
		BaseURL:      srv.URL,
		ProjectSlug:  "demo",
		PollInterval: -1, // one-shot
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Stop)
	return c
}

func TestSlow_SDK_ParallelEvaluations(t *testing.T) {
	slowSkip(t)
	t.Parallel()
	c := newStartedClient(t, buildSnapshot(20))
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 5_000; i++ {
				_ = c.Evaluate(fmt.Sprintf("flag-%d", i%20), map[string]any{"k": fmt.Sprintf("v%d", i%20)})
			}
		}(w)
	}
	wg.Wait()
}

func TestSlow_SDK_LargeSnapshot(t *testing.T) {
	slowSkip(t)
	t.Parallel()
	c := newStartedClient(t, buildSnapshot(500))
	for i := 0; i < 2_000; i++ {
		d := c.Evaluate(fmt.Sprintf("flag-%d", i%500), map[string]any{"k": fmt.Sprintf("v%d", i%500)})
		if d.Reason == "" {
			t.Fatalf("empty reason at i=%d", i)
		}
	}
}

func TestSlow_SDK_MisseKey(t *testing.T) {
	slowSkip(t)
	t.Parallel()
	c := newStartedClient(t, buildSnapshot(10))
	for i := 0; i < 2_000; i++ {
		_ = c.Evaluate(fmt.Sprintf("nonexistent-%d", i), nil)
	}
}
