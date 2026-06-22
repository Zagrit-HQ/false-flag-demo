package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/depot/falseflag/internal/gen/openapi"
	"github.com/depot/falseflag/internal/store"
)

func TestWriteJSON_SetsContentTypeAndStatus(t *testing.T) {
	t.Parallel()
	cases := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusAccepted,
		http.StatusNoContent,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusConflict,
		http.StatusUnprocessableEntity,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	}
	for _, code := range cases {
		code := code
		t.Run(fmt.Sprintf("status-%d", code), func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			writeJSON(w, code, map[string]string{"k": "v"})
			if w.Code != code {
				t.Errorf("status = %d, want %d", w.Code, code)
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("content-type = %q", got)
			}
			var body map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Errorf("decode: %v", err)
			}
			if body["k"] != "v" {
				t.Errorf("body[k] = %q", body["k"])
			}
		})
	}
}

func TestWriteJSON_NilBody(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	writeJSON(w, 200, nil)
	if got := strings.TrimSpace(w.Body.String()); got != "null" {
		t.Errorf("body = %q, want null", got)
	}
}

func TestWriteError_SimpleAndWrapped(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		err     error
		status  int
		wantDet bool
	}{
		{"simple", errors.New("bad input"), 400, false},
		{"wrapped", fmt.Errorf("outer: %w", errors.New("inner")), 500, true},
		{"sentinel-wrapped", fmt.Errorf("outer: %w", store.ErrNotFound), 404, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			writeError(w, tc.status, tc.err)
			if w.Code != tc.status {
				t.Errorf("status = %d", w.Code)
			}
			var body openapi.Error
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Error == "" {
				t.Errorf("Error empty")
			}
			if (body.Details != nil) != tc.wantDet {
				t.Errorf("Details nilness = %v, want non-nil=%v", body.Details, tc.wantDet)
			}
		})
	}
}

func TestNotFoundOrError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"not-found", store.ErrNotFound, http.StatusNotFound},
		{"wrapped-not-found", fmt.Errorf("get: %w", store.ErrNotFound), http.StatusNotFound},
		{"other", errors.New("oops"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			notFoundOrError(w, tc.err)
			if w.Code != tc.want {
				t.Errorf("status = %d want %d", w.Code, tc.want)
			}
		})
	}
}

func TestWriteStoreErr(t *testing.T) {
	t.Parallel()
	// A conflict from store.IsConflict isn't directly constructible
	// without poking internals; rely on documented sentinels and the
	// fallthrough case.
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"not-found", store.ErrNotFound, http.StatusNotFound},
		{"other", errors.New("disk"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			w := httptest.NewRecorder()
			writeStoreErr(w, tc.err)
			if w.Code != tc.want {
				t.Errorf("status = %d want %d", w.Code, tc.want)
			}
		})
	}
}

func TestBadRequest(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	badRequest(w, errors.New("invalid uuid"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d", w.Code)
	}
}

func TestDerefString(t *testing.T) {
	t.Parallel()
	if derefString(nil) != "" {
		t.Errorf("derefString(nil) != \"\"")
	}
	s := "hi"
	if derefString(&s) != "hi" {
		t.Errorf("derefString got %q", derefString(&s))
	}
}

func TestActorFromRequest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		setHdr bool
		val    string
		want   string
	}{
		{"missing", false, "", ""},
		{"empty", true, "", ""},
		{"alice", true, "alice", "alice"},
		{"email", true, "bob@example.com", "bob@example.com"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("GET", "/", nil)
			if tc.setHdr {
				r.Header.Set("X-Actor", tc.val)
			}
			if got := actorFromRequest(r); got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid", `{"k":"v"}`, false},
		{"empty", ``, true},
		{"malformed", `{`, true},
		{"unknown-field", `{"k":"v","x":1}`, true},
		{"valid-no-fields", `{}`, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("POST", "/", io.NopCloser(bytes.NewBufferString(tc.body)))
			var dst struct {
				K string `json:"k"`
			}
			err := decodeJSON(r, &dst)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestAPI_RequireStore(t *testing.T) {
	t.Parallel()
	a := &API{Store: nil}
	w := httptest.NewRecorder()
	ok := a.requireStore(w)
	if ok {
		t.Errorf("requireStore should return false when Store is nil")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d want 503", w.Code)
	}
}

func TestAPI_GetHealth(t *testing.T) {
	t.Parallel()
	a := &API{}
	r := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	a.GetHealth(w, r)
	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var body openapi.HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Service == "" {
		t.Errorf("Service empty")
	}
	if string(body.Status) != "ok" {
		t.Errorf("Status = %q", body.Status)
	}
	if body.Probe == nil || *body.Probe != "v1.health" {
		t.Errorf("Probe = %v", body.Probe)
	}
	if body.Timestamp == nil {
		t.Errorf("Timestamp nil")
	}
}
