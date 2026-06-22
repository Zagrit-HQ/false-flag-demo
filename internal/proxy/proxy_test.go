package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/depot/falseflag/internal/appconfig"
	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/eval"
	"github.com/depot/falseflag/internal/logging"
	"github.com/depot/falseflag/internal/sdkgo"
)

type fakeEvaluator struct {
	snap     *sdkgo.Snapshot
	lastKey  string
	lastCtx  sdkgo.EvalContext
	decision eval.Decision
}

func (f *fakeEvaluator) Evaluate(key string, ctx sdkgo.EvalContext) eval.Decision {
	f.lastKey = key
	f.lastCtx = ctx
	return f.decision
}

func (f *fakeEvaluator) Snapshot() *sdkgo.Snapshot { return f.snap }

func newProxy(t *testing.T) *Proxy {
	t.Helper()
	p, err := New(context.Background(), appconfig.ProxyConfig{Addr: ":0"}, logging.New("proxy"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return p
}

func TestProxy_Health(t *testing.T) {
	t.Parallel()
	p := newProxy(t)
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	var payload map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if payload["service"] != "falseflag-proxy" {
		t.Errorf("service = %q, want falseflag-proxy", payload["service"])
	}
}

func TestProxy_RequiresLogger(t *testing.T) {
	t.Parallel()
	if _, err := New(context.Background(), appconfig.ProxyConfig{Addr: ":0"}, nil); err == nil {
		t.Fatalf("expected error when logger is nil")
	}
}

func TestProxy_ReadyNoProject(t *testing.T) {
	t.Parallel()
	p := newProxy(t)
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200 (idle ready)", res.StatusCode)
	}
}

func TestProxy_ReadyStarting(t *testing.T) {
	t.Parallel()
	p := newProxy(t)
	p.SetEvaluator(&fakeEvaluator{snap: nil})
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", res.StatusCode)
	}
}

func TestProxy_ReadyMissingRequiredFlag(t *testing.T) {
	t.Parallel()
	p, err := New(context.Background(), appconfig.ProxyConfig{
		Addr:       ":0",
		ReadyFlags: []string{"proxy-readiness-bool", "checkout-redesign"},
	}, logging.New("proxy"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p.SetEvaluator(&fakeEvaluator{snap: &sdkgo.Snapshot{
		Flags: map[string]*config.Compiled{
			"checkout-redesign": {},
		},
	}})
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", res.StatusCode)
	}
	var body struct {
		Status               string   `json:"status"`
		MissingRequiredFlags []string `json:"missing_required_flags"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Status != "missing_required_flags" {
		t.Errorf("status body = %q", body.Status)
	}
	if len(body.MissingRequiredFlags) != 1 || body.MissingRequiredFlags[0] != "proxy-readiness-bool" {
		t.Errorf("missing flags = %#v", body.MissingRequiredFlags)
	}
}

func TestProxy_ReadyRequiredFlagsPresent(t *testing.T) {
	t.Parallel()
	p, err := New(context.Background(), appconfig.ProxyConfig{
		Addr:       ":0",
		ReadyFlags: []string{"proxy-readiness-bool"},
	}, logging.New("proxy"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p.SetEvaluator(&fakeEvaluator{snap: &sdkgo.Snapshot{
		Flags: map[string]*config.Compiled{
			"proxy-readiness-bool": {},
		},
	}})
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
}

func TestProxy_EvaluateHappyPath(t *testing.T) {
	t.Parallel()
	fake := &fakeEvaluator{
		snap: &sdkgo.Snapshot{Version: 4},
		decision: eval.Decision{
			Value:   true,
			Reason:  eval.ReasonRuleMatched,
			RuleID:  "r1",
			Version: 4,
		},
	}
	p := newProxy(t)
	p.SetEvaluator(fake)
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()

	body := bytes.NewBufferString(`{
		"key": "checkout-redesign",
		"default_value": false,
		"context": {"user": {"plan": "pro"}}
	}`)
	res, err := http.Post(ts.URL+"/v1/evaluate", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
	var d eval.Decision
	if err := json.NewDecoder(res.Body).Decode(&d); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if d.Value != true {
		t.Errorf("value = %v, want true", d.Value)
	}
	if d.Reason != eval.ReasonRuleMatched {
		t.Errorf("reason = %s", d.Reason)
	}
	if fake.lastKey != "checkout-redesign" {
		t.Errorf("client called with key=%q", fake.lastKey)
	}
}

func TestProxy_EvaluateNoProject(t *testing.T) {
	t.Parallel()
	p := newProxy(t)
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	res, err := http.Post(ts.URL+"/v1/evaluate", "application/json",
		bytes.NewBufferString(`{"key":"k"}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", res.StatusCode)
	}
}

func TestProxy_EvaluateMissingKey(t *testing.T) {
	t.Parallel()
	p := newProxy(t)
	p.SetEvaluator(&fakeEvaluator{snap: &sdkgo.Snapshot{}})
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	res, err := http.Post(ts.URL+"/v1/evaluate", "application/json",
		bytes.NewBufferString(`{"context":{}}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", res.StatusCode)
	}
}

func TestProxy_EvaluateInvalidBody(t *testing.T) {
	t.Parallel()
	p := newProxy(t)
	p.SetEvaluator(&fakeEvaluator{snap: &sdkgo.Snapshot{}})
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	res, err := http.Post(ts.URL+"/v1/evaluate", "application/json",
		bytes.NewBufferString("not json"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", res.StatusCode)
	}
}

func TestProxy_DefaultSubstitution(t *testing.T) {
	t.Parallel()
	fake := &fakeEvaluator{
		snap:     &sdkgo.Snapshot{},
		decision: eval.Decision{Value: nil, Reason: eval.ReasonDefault},
	}
	p := newProxy(t)
	p.SetEvaluator(fake)
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	res, err := http.Post(ts.URL+"/v1/evaluate", "application/json",
		bytes.NewBufferString(`{"key":"missing", "default_value": 42}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	var d eval.Decision
	_ = json.NewDecoder(res.Body).Decode(&d)
	if d.Value != float64(42) {
		t.Errorf("value = %v, want 42", d.Value)
	}
}

func TestProxy_SnapshotInfo(t *testing.T) {
	t.Parallel()
	fake := &fakeEvaluator{snap: &sdkgo.Snapshot{
		ID:        "abc",
		Version:   9,
		CreatedAt: time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC),
	}}
	p := newProxy(t)
	p.SetEvaluator(fake)
	ts := httptest.NewServer(p.Handler())
	defer ts.Close()
	res, err := http.Get(ts.URL + "/v1/snapshot")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d", res.StatusCode)
	}
	var info SnapshotInfo
	_ = json.NewDecoder(res.Body).Decode(&info)
	if info.ID != "abc" || info.Version != 9 {
		t.Errorf("info = %+v", info)
	}
}
