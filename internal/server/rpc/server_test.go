package rpc

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMount_RegistersAllServices(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	out := Mount(mux, Services{Store: nil, Log: slog.Default()})
	if out != mux {
		t.Error("Mount should return the same mux")
	}
	expected := []string{
		"/falseflag.v1.HealthService/",
		"/falseflag.v1.ProjectsService/",
		"/falseflag.v1.EnvironmentsService/",
		"/falseflag.v1.FlagsService/",
		"/falseflag.v1.SegmentsService/",
		"/falseflag.v1.SnapshotsService/",
		"/falseflag.v1.EvaluationService/",
		"/falseflag.v1.AuditService/",
	}
	// Use mux.Handler() to confirm each Connect prefix is registered.
	// We probe with a POST + dummy method path; Connect's handler will
	// own the request even if it ultimately rejects the body.
	for _, path := range expected {
		path := path
		t.Run(strings.Trim(path, "/"), func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("POST", path+"Check", strings.NewReader(""))
			_, pattern := mux.Handler(r)
			if pattern != path {
				t.Errorf("path %q: mux returned pattern %q", path, pattern)
			}
		})
	}
}

func TestMount_AllowsChain(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	if Mount(mux, Services{}) != mux {
		t.Error("Mount should return the same mux for chaining")
	}
}
