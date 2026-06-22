package tools

import (
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestConnectErrToToolResult_NilPassthrough(t *testing.T) {
	t.Parallel()
	if got := connectErrToToolResult(nil); got != nil {
		t.Fatalf("nil error should produce nil result, got %+v", got)
	}
}

func TestConnectErrToToolResult_CodeMapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		err    error
		wantIn string
	}{
		{"not_found", connect.NewError(connect.CodeNotFound, errors.New("project missing")), "not found: project missing"},
		{"invalid", connect.NewError(connect.CodeInvalidArgument, errors.New("bad slug")), "invalid argument: bad slug"},
		{"perm", connect.NewError(connect.CodePermissionDenied, errors.New("nope")), "permission denied: nope"},
		{"unavail", connect.NewError(connect.CodeUnavailable, errors.New("upstream offline")), "upstream unavailable: upstream offline"},
		{"unauth", connect.NewError(connect.CodeUnauthenticated, errors.New("no token")), "unauthenticated: no token"},
		{"internal", connect.NewError(connect.CodeInternal, errors.New("boom")), "upstream error (internal): boom"},
		{"plain", errors.New("dial tcp: connection refused"), "upstream error: dial tcp: connection refused"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := connectErrToToolResult(tc.err)
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if !got.IsError {
				t.Errorf("expected IsError=true")
			}
			if len(got.Content) != 1 {
				t.Fatalf("want 1 content block, got %d", len(got.Content))
			}
			tc0, ok := got.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("content[0] not TextContent: %T", got.Content[0])
			}
			text := tc0.Text
			if !strings.Contains(text, tc.wantIn) {
				t.Errorf("text %q does not contain %q", text, tc.wantIn)
			}
		})
	}
}
