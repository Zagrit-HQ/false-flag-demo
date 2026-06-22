package openapi

import "testing"

// TestGeneratedHealthResponseConstructs confirms the oapi-codegen output is
// usable. It exists only to prove `make generate` produces a linkable
// package; the real server tests exercise the OpenAPI surface in slice 3.
func TestGeneratedHealthResponseConstructs(t *testing.T) {
	t.Parallel()

	res := HealthResponse{
		Status:  Ok,
		Service: "falseflag-api",
		Version: "dev",
	}
	if !res.Status.Valid() {
		t.Fatalf("Ok must be a valid enum member")
	}
	if res.Service != "falseflag-api" {
		t.Fatalf("Service round-trip failed")
	}
	if res.Version != "dev" {
		t.Fatalf("Version round-trip failed")
	}
}
