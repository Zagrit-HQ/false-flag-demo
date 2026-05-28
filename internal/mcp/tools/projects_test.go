package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

func TestListProjects_Happy(t *testing.T) {
	t.Parallel()
	fake := &fakeProjects{Items: []*pb.Project{
		{Id: "p1", Slug: "acme-web", DisplayName: "Acme Web"},
		{Id: "p2", Slug: "acme-mobile", DisplayName: "Acme Mobile"},
	}}
	client := &clientapi.Client{Projects: fake}
	h := listProjects(client)

	res, _, err := h(context.Background(), nil, ListProjectsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected non-error result, got %+v", res)
	}
	text := firstText(t, res)
	if !strings.Contains(text, "acme-web") || !strings.Contains(text, "acme-mobile") {
		t.Errorf("expected both project slugs in body, got %s", text)
	}
}

func TestListProjects_NilClient(t *testing.T) {
	t.Parallel()
	h := listProjects(nil)
	res, _, err := h(context.Background(), nil, ListProjectsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true for nil client")
	}
	if !strings.Contains(firstText(t, res), "projects client unavailable") {
		t.Errorf("unexpected error text: %s", firstText(t, res))
	}
}

func TestListProjects_UpstreamError(t *testing.T) {
	t.Parallel()
	fake := &fakeProjects{Err: notFoundErr("nothing here")}
	client := &clientapi.Client{Projects: fake}
	h := listProjects(client)

	res, _, err := h(context.Background(), nil, ListProjectsInput{})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true on upstream not_found")
	}
	if !strings.Contains(firstText(t, res), "not found") {
		t.Errorf("expected 'not found' in body, got %s", firstText(t, res))
	}
}

// firstText extracts the first TextContent body for assertions.
// Tools always emit exactly one text block today.
func firstText(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	if r == nil || len(r.Content) == 0 {
		t.Fatal("nil or empty content")
	}
	tc, ok := r.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", r.Content[0])
	}
	return tc.Text
}
