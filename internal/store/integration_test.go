package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/depot/falseflag/internal/store"
)

// Integration tests against live storage. Every test runs against both
// backends via forEachBackend: the SQLite subtest runs unconditionally
// using t.TempDir(); the Postgres subtest runs only when
// FALSEFLAG_TEST_DATABASE_URL is set, otherwise it skips.
//
// To run the Postgres half locally:
//   make up
//   FALSEFLAG_TEST_DATABASE_URL=postgres://falseflag:falseflag@localhost:5432/falseflag?sslmode=disable \
//     go test ./internal/store/...
//
// The SQLite half needs nothing — `go test ./internal/store/...` runs
// it out of the box.

// forEachBackend runs fn against both storage backends. The fresh
// store created per subtest is migrated and truncated before fn is
// called, so each subtest sees a clean slate.
func forEachBackend(t *testing.T, fn func(t *testing.T, s store.Store)) {
	t.Helper()
	t.Run("sqlite", func(t *testing.T) {
		dir := t.TempDir()
		s := mustOpen(t, "sqlite://"+filepath.Join(dir, "test.db"))
		fn(t, s)
	})
	t.Run("postgres", func(t *testing.T) {
		dsn := os.Getenv("FALSEFLAG_TEST_DATABASE_URL")
		if dsn == "" {
			t.Skip("set FALSEFLAG_TEST_DATABASE_URL to enable the postgres integration subtest")
		}
		s := mustOpen(t, dsn)
		fn(t, s)
	})
}

func mustOpen(t *testing.T, dsn string) store.Store {
	t.Helper()
	ctx := context.Background()
	s, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(s.Close)
	if err := s.Migrate(ctx, nil); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := s.TruncateForTest(ctx); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return s
}

func TestProjectAndFlagLifecycle(t *testing.T) {
	forEachBackend(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		proj, err := s.CreateProject(ctx, "demo", "Demo Project", "json")
		if err != nil {
			t.Fatalf("create project: %v", err)
		}
		if proj.Slug != "demo" {
			t.Errorf("Slug = %q", proj.Slug)
		}

		gotProj, err := s.GetProjectBySlug(ctx, "demo")
		if err != nil {
			t.Fatalf("get project: %v", err)
		}
		if gotProj.ID != proj.ID {
			t.Errorf("project id mismatch")
		}

		flag, err := s.CreateFlag(ctx, store.CreateFlagParams{
			ProjectID:    proj.ID,
			Key:          "checkout-v2",
			Name:         "Checkout V2",
			Description:  "rollout for new checkout",
			ValueType:    "boolean",
			DefaultValue: json.RawMessage(`false`),
		})
		if err != nil {
			t.Fatalf("create flag: %v", err)
		}

		v1, err := s.PublishFlagVersionStandalone(ctx, store.PublishFlagVersionParams{
			FlagID:   flag.ID,
			Strategy: "json",
			Source:   json.RawMessage(`{"value_type":"boolean","default":false,"rules":[]}`),
			Compiled: json.RawMessage(`{"value_type":"boolean","default":false,"rules":[]}`),
		})
		if err != nil {
			t.Fatalf("publish v1: %v", err)
		}
		if v1.Version != 1 {
			t.Errorf("v1.Version = %d, want 1", v1.Version)
		}

		v2, err := s.PublishFlagVersionStandalone(ctx, store.PublishFlagVersionParams{
			FlagID:   flag.ID,
			Strategy: "cel",
			Source:   json.RawMessage(`{"value_type":"boolean","default":false,"rules":[]}`),
			Compiled: json.RawMessage(`{"value_type":"boolean","default":false,"rules":[]}`),
		})
		if err != nil {
			t.Fatalf("publish v2: %v", err)
		}
		if v2.Version != 2 {
			t.Errorf("v2.Version = %d, want 2", v2.Version)
		}

		latest, err := s.GetLatestFlagVersion(ctx, flag.ID)
		if err != nil {
			t.Fatalf("latest: %v", err)
		}
		if latest.Version != 2 {
			t.Errorf("latest.Version = %d, want 2", latest.Version)
		}
		if latest.Strategy != "cel" {
			t.Errorf("latest.Strategy = %q", latest.Strategy)
		}

		versions, err := s.ListFlagVersions(ctx, flag.ID)
		if err != nil {
			t.Fatalf("list versions: %v", err)
		}
		if len(versions) != 2 {
			t.Errorf("len(versions) = %d, want 2", len(versions))
		}

		if _, err := s.AppendAudit(ctx, store.AppendAuditParams{
			ProjectID: nullUUID(proj.ID.String()),
			FlagID:    nullUUID(flag.ID.String()),
			Action:    "publish_version",
			Actor:     "alice@example.com",
			Payload:   json.RawMessage(`{"version":2}`),
		}); err != nil {
			t.Fatalf("append audit: %v", err)
		}

		events, err := s.ListAuditEvents(ctx, store.ListAuditEventsParams{
			ProjectID: proj.ID,
			Limit:     100,
		})
		if err != nil {
			t.Fatalf("list audit: %v", err)
		}
		if len(events) != 1 {
			t.Fatalf("audit count = %d, want 1", len(events))
		}
		if events[0].Actor != "alice@example.com" {
			t.Errorf("actor = %q, want alice@example.com", events[0].Actor)
		}
	})
}

func TestProjectNotFound(t *testing.T) {
	forEachBackend(t, func(t *testing.T, s store.Store) {
		_, err := s.GetProjectBySlug(context.Background(), "does-not-exist")
		if err != store.ErrNotFound {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

func TestEnvironmentsLifecycle(t *testing.T) {
	forEachBackend(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		proj, err := s.CreateProject(ctx, "envtest", "Env Test", "json")
		if err != nil {
			t.Fatalf("create project: %v", err)
		}
		prod, err := s.CreateEnvironment(ctx, proj.ID, "prod", "Production")
		if err != nil {
			t.Fatalf("create env: %v", err)
		}
		if prod.Slug != "prod" {
			t.Errorf("Slug = %q", prod.Slug)
		}
		if _, err := s.CreateEnvironment(ctx, proj.ID, "prod", "Duplicate"); err == nil {
			t.Errorf("expected conflict on duplicate (project, slug)")
		} else if !store.IsConflict(err) {
			t.Errorf("expected IsConflict, got %v", err)
		}
		envs, err := s.ListEnvironmentsByProject(ctx, proj.ID)
		if err != nil || len(envs) != 1 {
			t.Errorf("list envs len=%d err=%v", len(envs), err)
		}
		if _, err := s.GetEnvironmentBySlug(ctx, proj.ID, "missing"); err != store.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestSegmentsLifecycle(t *testing.T) {
	forEachBackend(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		proj, err := s.CreateProject(ctx, "segtest", "Seg Test", "json")
		if err != nil {
			t.Fatalf("create project: %v", err)
		}
		predicate := json.RawMessage(`{"kind":"eq","attr":"user.plan","value":"pro"}`)
		seg, err := s.CreateSegment(ctx, store.CreateSegmentParams{
			ProjectID: proj.ID,
			Key:       "pro-users",
			Name:      "Pro Users",
			Predicate: predicate,
		})
		if err != nil {
			t.Fatalf("create segment: %v", err)
		}
		if seg.Key != "pro-users" {
			t.Errorf("Key = %q", seg.Key)
		}

		updated, err := s.UpdateSegment(ctx, store.UpdateSegmentParams{
			ProjectID:   proj.ID,
			Key:         "pro-users",
			Name:        "Pro & Enterprise Users",
			Description: "anyone on a paid plan",
			Predicate:   json.RawMessage(`{"kind":"in","attr":"user.plan","values":["pro","enterprise"]}`),
		})
		if err != nil {
			t.Fatalf("update segment: %v", err)
		}
		if updated.Name != "Pro & Enterprise Users" {
			t.Errorf("Name = %q", updated.Name)
		}
	})
}

func TestSnapshotMonotonic(t *testing.T) {
	forEachBackend(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		proj, err := s.CreateProject(ctx, "snaptest", "Snap Test", "json")
		if err != nil {
			t.Fatalf("create project: %v", err)
		}
		for i := 1; i <= 3; i++ {
			snap, err := s.CompileSnapshot(ctx, store.CompileSnapshotParams{
				ProjectID: proj.ID,
				Compiled:  json.RawMessage(`{"flags":{}}`),
			})
			if err != nil {
				t.Fatalf("compile snapshot %d: %v", i, err)
			}
			if snap.Version != i {
				t.Errorf("snap[%d].Version = %d, want %d", i, snap.Version, i)
			}
		}
		latest, err := s.GetLatestSnapshot(ctx, proj.ID)
		if err != nil {
			t.Fatalf("latest: %v", err)
		}
		if latest.Version != 3 {
			t.Errorf("latest.Version = %d, want 3", latest.Version)
		}
	})
}

func TestWithAuditRollsBackOnMutationFailure(t *testing.T) {
	forEachBackend(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		proj, err := s.CreateProject(ctx, "audittest", "Audit Test", "json")
		if err != nil {
			t.Fatalf("create project: %v", err)
		}
		wantErr := errors.New("forced rollback")
		err = s.WithAudit(ctx, store.AppendAuditParams{
			ProjectID: nullUUID(proj.ID.String()),
			Action:    "test",
		}, func(_ store.Tx) error {
			return wantErr
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("WithAudit returned %v, want %v", err, wantErr)
		}
		events, err := s.ListAuditEvents(ctx, store.ListAuditEventsParams{
			ProjectID: proj.ID,
			Limit:     100,
		})
		if err != nil {
			t.Fatalf("list audit: %v", err)
		}
		if len(events) != 0 {
			t.Errorf("audit count = %d, want 0 (transaction should have rolled back)", len(events))
		}
	})
}

// TestWithAuditRollsBackOnPanic asserts that flag-version writes
// performed inside a WithAudit closure are rolled back if the closure
// panics. Before slice 8 Phase 4, PublishFlagVersion opened its own
// independent serializable transaction, so its insert would commit
// before WithAudit's deferred rollback ran — orphaning the row.
func TestWithAuditRollsBackOnPanic(t *testing.T) {
	forEachBackend(t, func(t *testing.T, s store.Store) {
		ctx := context.Background()

		proj, err := s.CreateProject(ctx, "panictest", "Panic Test", "json")
		if err != nil {
			t.Fatalf("create project: %v", err)
		}
		flag, err := s.CreateFlag(ctx, store.CreateFlagParams{
			ProjectID:    proj.ID,
			Key:          "boom",
			Name:         "Boom",
			ValueType:    "boolean",
			DefaultValue: json.RawMessage(`false`),
		})
		if err != nil {
			t.Fatalf("create flag: %v", err)
		}

		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic to propagate, got none")
				}
			}()
			_ = s.WithAudit(ctx, store.AppendAuditParams{
				ProjectID: nullUUID(proj.ID.String()),
				FlagID:    nullUUID(flag.ID.String()),
				Action:    "publish_version",
			}, func(tx store.Tx) error {
				if _, err := tx.PublishFlagVersion(ctx, store.PublishFlagVersionParams{
					FlagID:   flag.ID,
					Strategy: "json",
					Source:   json.RawMessage(`{"value_type":"boolean","default":false,"rules":[]}`),
					Compiled: json.RawMessage(`{"value_type":"boolean","default":false,"rules":[]}`),
				}); err != nil {
					return err
				}
				panic("forced post-insert panic")
			})
		}()

		versions, err := s.ListFlagVersions(ctx, flag.ID)
		if err != nil {
			t.Fatalf("list versions: %v", err)
		}
		if len(versions) != 0 {
			t.Errorf("flag_versions count = %d, want 0 (panic should have rolled back)", len(versions))
		}
		events, err := s.ListAuditEvents(ctx, store.ListAuditEventsParams{
			ProjectID: proj.ID,
			Limit:     100,
		})
		if err != nil {
			t.Fatalf("list audit: %v", err)
		}
		if len(events) != 0 {
			t.Errorf("audit count = %d, want 0 (panic should have rolled back)", len(events))
		}
	})
}
