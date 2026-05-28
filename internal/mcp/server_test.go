package mcp

import (
	"context"
	"sort"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

// TestServer_AdvertisedToolsAreInCanonicalList asserts that every
// tool the SDK advertises lives in the canonical ToolNames slice.
// Equality (all six tools registered) is enforced once phase 5 lands;
// during phases 3-4 some entries in ToolNames are still placeholders.
// Either way, no orphan tool can appear without ToolNames knowing.
// Uses NewInMemoryTransports so no ports are opened.
func TestServer_AdvertisedToolsAreInCanonicalList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	srv := newServer()
	// Register against a nil client — the test only asserts the
	// surface, not invocation. Tool handlers that dereference
	// client return a friendly "client unavailable" tool error
	// (see tools/projects.go), they do not panic at registration.
	RegisterTools(srv, nil)

	ct, st := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer func() { _ = ss.Close() }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0.0.0"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer func() { _ = cs.Close() }()

	res, err := cs.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	// Phase 5: equality is enforced now that all six tools are registered.
	if len(res.Tools) != len(ToolNames) {
		t.Errorf("tool count mismatch: got %d, want %d (ToolNames=%v)", len(res.Tools), len(ToolNames), ToolNames)
	}
	canonical := make(map[string]bool, len(ToolNames))
	for _, n := range ToolNames {
		canonical[n] = true
	}
	got := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
		if !canonical[tool.Name] {
			t.Errorf("tool %q advertised but missing from ToolNames", tool.Name)
		}
	}
	sort.Strings(got)
	if len(got) == 0 {
		t.Fatal("server advertises no tools")
	}
}

// fakeProjectsForServer implements just enough of the Connect client
// interface for the end-to-end call test.
type fakeProjectsForServer struct {
	falseflagv1connect.ProjectsServiceClient
	items []*pb.Project
}

func (f *fakeProjectsForServer) ListProjects(_ context.Context, _ *connect.Request[pb.ListProjectsRequest]) (*connect.Response[pb.ListProjectsResponse], error) {
	return connect.NewResponse(&pb.ListProjectsResponse{Items: f.items}), nil
}

// TestServer_CallToolEndToEnd drives a tools/call request through
// the in-process MCP transport and asserts the registered handler
// actually fires. Catches wiring regressions between
// RegisterTools, tool name strings, and the SDK's schema layer.
func TestServer_CallToolEndToEnd(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	client := &clientapi.Client{
		Projects: &fakeProjectsForServer{items: []*pb.Project{
			{Id: "p1", Slug: "acme-web", DisplayName: "Acme Web"},
		}},
	}
	srv := newServer()
	RegisterTools(srv, client)

	ct, st := mcp.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer func() { _ = ss.Close() }()
	c := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0.0.0"}, nil)
	cs, err := c.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer func() { _ = cs.Close() }()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_projects",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError: %+v", res)
	}
	if len(res.Content) == 0 {
		t.Fatal("no content blocks returned")
	}
	tc0, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	if !strings.Contains(tc0.Text, "acme-web") {
		t.Errorf("expected acme-web in body, got %s", tc0.Text)
	}
}
