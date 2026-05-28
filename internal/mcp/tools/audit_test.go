package tools

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

type fakeAudit struct {
	falseflagv1connect.AuditServiceClient
	LastReq *pb.ListAuditEventsRequest
	Resp    *pb.ListAuditEventsResponse
	Err     error
}

func (f *fakeAudit) ListAuditEvents(_ context.Context, req *connect.Request[pb.ListAuditEventsRequest]) (*connect.Response[pb.ListAuditEventsResponse], error) {
	f.LastReq = req.Msg
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Resp == nil {
		return connect.NewResponse(&pb.ListAuditEventsResponse{}), nil
	}
	return connect.NewResponse(f.Resp), nil
}

func TestSearchAuditLog_DefaultLimit(t *testing.T) {
	t.Parallel()
	fake := &fakeAudit{}
	h := searchAuditLog(&clientapi.Client{Audit: fake})

	_, _, err := h(context.Background(), nil, SearchAuditLogInput{ProjectSlug: "acme"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if got := fake.LastReq.GetLimit(); got != 50 {
		t.Errorf("default limit: got %d want 50", got)
	}
}

func TestSearchAuditLog_LimitClamp(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		in   int32
		want int32
	}{
		"zero":     {0, 50},
		"negative": {-3, 50},
		"normal":   {25, 25},
		"max":      {200, 200},
		"over":     {9999, 200},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fake := &fakeAudit{}
			h := searchAuditLog(&clientapi.Client{Audit: fake})
			_, _, err := h(context.Background(), nil, SearchAuditLogInput{ProjectSlug: "p", Limit: tc.in})
			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}
			if got := fake.LastReq.GetLimit(); got != tc.want {
				t.Errorf("limit: got %d want %d", got, tc.want)
			}
		})
	}
}

func TestSearchAuditLog_FiltersForwarded(t *testing.T) {
	t.Parallel()
	fake := &fakeAudit{}
	h := searchAuditLog(&clientapi.Client{Audit: fake})

	_, _, err := h(context.Background(), nil, SearchAuditLogInput{
		ProjectSlug: "acme",
		Action:      "publish_version",
		Actor:       "mcp/falseflag-mcp",
		From:        "2026-01-01T00:00:00Z",
		To:          "2026-02-01T00:00:00Z",
		Cursor:      "opaque-cursor",
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if fake.LastReq.GetAction() != "publish_version" {
		t.Errorf("action not forwarded: %q", fake.LastReq.GetAction())
	}
	if fake.LastReq.GetActor() != "mcp/falseflag-mcp" {
		t.Errorf("actor not forwarded: %q", fake.LastReq.GetActor())
	}
	if fake.LastReq.GetFrom() == nil || fake.LastReq.GetTo() == nil {
		t.Errorf("from/to timestamps not forwarded: from=%v to=%v", fake.LastReq.GetFrom(), fake.LastReq.GetTo())
	}
	if fake.LastReq.GetCursor() != "opaque-cursor" {
		t.Errorf("cursor not forwarded: %q", fake.LastReq.GetCursor())
	}
}

func TestSearchAuditLog_BadTimestamp(t *testing.T) {
	t.Parallel()
	fake := &fakeAudit{}
	h := searchAuditLog(&clientapi.Client{Audit: fake})

	res, _, err := h(context.Background(), nil, SearchAuditLogInput{
		ProjectSlug: "acme",
		From:        "not a date",
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true for invalid timestamp")
	}
	if !strings.Contains(firstText(t, res), "invalid from timestamp") {
		t.Errorf("expected 'invalid from timestamp', got %s", firstText(t, res))
	}
	if fake.LastReq != nil {
		t.Error("upstream should not have been called on bad input")
	}
}

func TestSearchAuditLog_ProjectNotFound(t *testing.T) {
	t.Parallel()
	fake := &fakeAudit{Err: notFoundErr("project missing")}
	h := searchAuditLog(&clientapi.Client{Audit: fake})

	res, _, err := h(context.Background(), nil, SearchAuditLogInput{ProjectSlug: "nope"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true")
	}
	if !strings.Contains(firstText(t, res), "not found") {
		t.Errorf("expected 'not found' in body, got %s", firstText(t, res))
	}
}
