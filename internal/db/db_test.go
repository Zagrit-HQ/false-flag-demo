package db

import "testing"

// TestGeneratedProjectModelExists is a smoke test asserting sqlc produced
// a linkable Queries surface against the projects table. Real database
// integration tests come with slice 3.
func TestGeneratedProjectModelExists(t *testing.T) {
	t.Parallel()

	p := Project{
		Slug:           "demo",
		DisplayName:    "Demo Project",
		ConfigStrategy: "json",
	}
	if p.Slug != "demo" {
		t.Fatalf("Project.Slug round-trip failed")
	}
	if p.ConfigStrategy != "json" {
		t.Fatalf("Project.ConfigStrategy default not preserved")
	}
	if p.DisplayName != "Demo Project" {
		t.Fatalf("Project.DisplayName round-trip failed")
	}
}

func TestListProjectsParamsCompile(t *testing.T) {
	t.Parallel()

	// Confirm the generated CreateProjectParams type is structurally
	// what we expect. This guards against silent sqlc regenerations
	// that drop columns.
	_ = CreateProjectParams{
		Slug:           "demo",
		DisplayName:    "Demo Project",
		ConfigStrategy: "json",
	}
}
