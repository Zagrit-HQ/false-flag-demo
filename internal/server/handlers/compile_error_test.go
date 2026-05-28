package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/gen/openapi"
)

func TestIsCompileError_Sentinels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("disk"), false},
		{"unknown-strategy", config.ErrUnknownStrategy, false},
		{"ts", config.ErrTypeScriptCompileFailure, true},
		{"ts-wrapped", fmt.Errorf("wrap: %w", config.ErrTypeScriptCompileFailure), true},
		{"cel", config.ErrCELCompileFailure, true},
		{"cel-wrapped", fmt.Errorf("wrap: %w", config.ErrCELCompileFailure), true},
		{"ir", config.ErrInvalidIR, true},
		{"ir-wrapped", fmt.Errorf("wrap: %w", config.ErrInvalidIR), true},
		{"predicate", config.ErrInvalidPredicate, true},
		{"value-type", config.ErrInvalidValueType, true},
		{"esbuild", &config.EsbuildError{Messages: []config.EsbuildMessage{{Text: "x"}}}, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isCompileError(tc.err)
			if got != tc.want {
				t.Errorf("isCompileError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestWriteCompileError_PlainErrorEmits422(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	writeCompileError(w, errors.New("boom"))
	if w.Code != 422 {
		t.Fatalf("status = %d, want 422", w.Code)
	}
	var body openapi.CompileError
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Message != "boom" {
		t.Errorf("Message = %q", body.Message)
	}
	if body.Details != nil {
		t.Errorf("Details should be nil for plain error")
	}
}

func TestWriteCompileError_EsbuildDetails(t *testing.T) {
	t.Parallel()
	esErr := &config.EsbuildError{
		Messages: []config.EsbuildMessage{
			{File: "rules.ts", Line: 12, Column: 3, Text: "Expected ';' but found '}'"},
			{File: "rules.ts", Line: 14, Column: 1, Text: "Cannot find name 'foo'"},
			{Text: "bare diagnostic"},
		},
	}
	w := httptest.NewRecorder()
	writeCompileError(w, esErr)
	if w.Code != 422 {
		t.Fatalf("status = %d", w.Code)
	}
	var body openapi.CompileError
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Details == nil {
		t.Fatal("Details should be populated for EsbuildError")
	}
	if len(*body.Details) != 3 {
		t.Fatalf("details len = %d", len(*body.Details))
	}
	d0 := (*body.Details)[0]
	if d0.File == nil || *d0.File != "rules.ts" {
		t.Errorf("d0.File = %v", d0.File)
	}
	if d0.Line == nil || *d0.Line != 12 {
		t.Errorf("d0.Line = %v", d0.Line)
	}
	if d0.Column == nil || *d0.Column != 3 {
		t.Errorf("d0.Column = %v", d0.Column)
	}
	d2 := (*body.Details)[2]
	if d2.File != nil || d2.Line != nil || d2.Column != nil {
		t.Errorf("d2 should have nil File/Line/Column, got %+v", d2)
	}
}

func TestWriteCompileError_WrappedEsbuild(t *testing.T) {
	t.Parallel()
	esErr := &config.EsbuildError{
		Messages: []config.EsbuildMessage{{File: "a.ts", Line: 1, Column: 1, Text: "x"}},
	}
	wrapped := fmt.Errorf("compile: %w", esErr)
	w := httptest.NewRecorder()
	writeCompileError(w, wrapped)
	var body openapi.CompileError
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body.Details == nil {
		t.Errorf("Details should be set via errors.As on wrapped EsbuildError")
	}
}
