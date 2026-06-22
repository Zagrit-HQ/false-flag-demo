package clientapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
)

func TestNew_WiresAllServices(t *testing.T) {
	t.Parallel()
	c := New("http://localhost:9999", "operator")
	if c.Projects == nil {
		t.Errorf("Projects nil")
	}
	if c.Environments == nil {
		t.Errorf("Environments nil")
	}
	if c.Flags == nil {
		t.Errorf("Flags nil")
	}
	if c.Segments == nil {
		t.Errorf("Segments nil")
	}
	if c.Snapshots == nil {
		t.Errorf("Snapshots nil")
	}
	if c.Evaluation == nil {
		t.Errorf("Evaluation nil")
	}
	if c.Audit == nil {
		t.Errorf("Audit nil")
	}
}

func TestNew_AcceptsEmptyActor(t *testing.T) {
	t.Parallel()
	c := New("http://localhost:9999", "")
	if c.Projects == nil {
		t.Errorf("Projects nil")
	}
}

// fakeHealthService is the minimal HealthService implementation used to
// verify the actor interceptor injects X-Actor on outgoing requests.
type fakeHealthService struct {
	lastActor atomic.Value // string
}

func (f *fakeHealthService) Check(_ context.Context, req *connect.Request[pb.CheckRequest]) (*connect.Response[pb.CheckResponse], error) {
	f.lastActor.Store(req.Header().Get("X-Actor"))
	return connect.NewResponse(&pb.CheckResponse{}), nil
}

func startTestHealth(t *testing.T) (svc *fakeHealthService, url string) {
	t.Helper()
	svc = &fakeHealthService{}
	mux := http.NewServeMux()
	mux.Handle(falseflagv1connect.NewHealthServiceHandler(svc))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return svc, srv.URL
}

func TestActorInterceptor_SetsHeader(t *testing.T) {
	t.Parallel()
	svc, url := startTestHealth(t)
	c := New(url, "alice")
	_, err := c.Projects.ListProjects(context.Background(), connect.NewRequest(&pb.ListProjectsRequest{}))
	// Listing projects against a Health-only server returns an error,
	// but it still passes through the interceptor. We exercise the
	// interceptor via the Health endpoint to read the captured header.
	_ = err

	_, err = c.Audit.ListAuditEvents(context.Background(), connect.NewRequest(&pb.ListAuditEventsRequest{}))
	_ = err

	// Use the health client directly: it does have a real handler.
	hc := falseflagv1connect.NewHealthServiceClient(&http.Client{}, url, connect.WithInterceptors(actorInterceptor("alice")))
	_, err = hc.Check(context.Background(), connect.NewRequest(&pb.CheckRequest{}))
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if got, _ := svc.lastActor.Load().(string); got != "alice" {
		t.Errorf("X-Actor = %q want alice", got)
	}
}

func TestActorInterceptor_EmptyActorOmitsHeader(t *testing.T) {
	t.Parallel()
	svc, url := startTestHealth(t)
	hc := falseflagv1connect.NewHealthServiceClient(&http.Client{}, url, connect.WithInterceptors(actorInterceptor("")))
	if _, err := hc.Check(context.Background(), connect.NewRequest(&pb.CheckRequest{})); err != nil {
		t.Fatalf("health: %v", err)
	}
	if got, _ := svc.lastActor.Load().(string); got != "" {
		t.Errorf("X-Actor = %q want empty", got)
	}
}

func TestActorInterceptor_VariousActorValues(t *testing.T) {
	t.Parallel()
	cases := []string{"alice", "bot", "alice@example.com", "ci-runner-42", "operator"}
	for _, actor := range cases {
		actor := actor
		t.Run(actor, func(t *testing.T) {
			t.Parallel()
			svc, url := startTestHealth(t)
			hc := falseflagv1connect.NewHealthServiceClient(&http.Client{}, url, connect.WithInterceptors(actorInterceptor(actor)))
			if _, err := hc.Check(context.Background(), connect.NewRequest(&pb.CheckRequest{})); err != nil {
				t.Fatalf("health: %v", err)
			}
			if got, _ := svc.lastActor.Load().(string); got != actor {
				t.Errorf("X-Actor = %q want %q", got, actor)
			}
		})
	}
}

func TestNew_BaseURLEcho(t *testing.T) {
	t.Parallel()
	// New only validates the URL lazily (Connect clients build paths
	// on first call). Construction with various base URLs must
	// succeed without panic.
	urls := []string{
		"http://localhost:8080",
		"http://localhost:8080/",
		"https://api.example.com",
		"https://api.example.com/api",
		"http://10.0.0.1:9090",
	}
	for _, u := range urls {
		u := u
		t.Run(strings.ReplaceAll(u, "/", "_"), func(t *testing.T) {
			t.Parallel()
			c := New(u, "actor")
			if c == nil {
				t.Errorf("nil client for %q", u)
			}
		})
	}
}
