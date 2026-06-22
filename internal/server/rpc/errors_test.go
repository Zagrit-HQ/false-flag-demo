package rpc

import (
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/store"
)

func TestConnectError_NilReturnsNil(t *testing.T) {
	t.Parallel()
	if got := connectError(nil); got != nil {
		t.Errorf("connectError(nil) = %v", got)
	}
}

func TestConnectError_Mapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want connect.Code
	}{
		{"not-found", store.ErrNotFound, connect.CodeNotFound},
		{"not-found-wrapped", fmt.Errorf("get: %w", store.ErrNotFound), connect.CodeNotFound},
		{"conflict", store.ErrConflict, connect.CodeAlreadyExists},
		{"conflict-wrapped", fmt.Errorf("write: %w", store.ErrConflict), connect.CodeAlreadyExists},
		{"random", errors.New("disk"), connect.CodeInternal},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := connectError(tc.err)
			if err == nil {
				t.Fatalf("connectError(%v) = nil", tc.err)
			}
			if err.Code() != tc.want {
				t.Errorf("code = %v, want %v", err.Code(), tc.want)
			}
		})
	}
}

func TestBadRequest(t *testing.T) {
	t.Parallel()
	err := badRequest(errors.New("invalid uuid"))
	if err == nil || err.Code() != connect.CodeInvalidArgument {
		t.Errorf("badRequest code = %v", err)
	}
}

func TestIsCompileError_Sentinels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("oops"), false},
		{"ts", config.ErrTypeScriptCompileFailure, true},
		{"ts-wrapped", fmt.Errorf("x: %w", config.ErrTypeScriptCompileFailure), true},
		{"cel", config.ErrCELCompileFailure, true},
		{"ir", config.ErrInvalidIR, true},
		{"pred", config.ErrInvalidPredicate, true},
		{"value-type", config.ErrInvalidValueType, true},
		{"esbuild", &config.EsbuildError{Messages: []config.EsbuildMessage{{Text: "x"}}}, true},
		{"unknown-strategy", config.ErrUnknownStrategy, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isCompileError(tc.err); got != tc.want {
				t.Errorf("isCompileError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestCompileErrorToConnect(t *testing.T) {
	t.Parallel()
	plain := errors.New("plain")
	es := &config.EsbuildError{Messages: []config.EsbuildMessage{{File: "a.ts", Line: 1, Column: 2, Text: "x"}}}
	for _, err := range []error{plain, es} {
		err := err
		t.Run(fmt.Sprintf("%T", err), func(t *testing.T) {
			t.Parallel()
			out := compileErrorToConnect(err)
			if out == nil {
				t.Fatal("nil error")
			}
			if out.Code() != connect.CodeInvalidArgument {
				t.Errorf("code = %v", out.Code())
			}
		})
	}
}

func TestActorFromHeader(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"X-Actor": "alice",
		"x-actor": "alice", // function delegates to header.Get
		"x-Actor": "alice",
	}
	for k, v := range cases {
		k, v := k, v
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			get := func(name string) string {
				if name == k {
					return v
				}
				return ""
			}
			// The function only calls get("X-Actor"); only the
			// canonical key actually matches.
			got := actorFromHeader(get)
			want := ""
			if k == "X-Actor" {
				want = v
			}
			if got != want {
				t.Errorf("got %q want %q", got, want)
			}
		})
	}
}

func TestActorFromHeader_Empty(t *testing.T) {
	t.Parallel()
	got := actorFromHeader(func(string) string { return "" })
	if got != "" {
		t.Errorf("got %q want empty", got)
	}
}
