// Package store is the persistence layer for the FalseFlag control
// plane. The exported Store is an interface satisfied by two concrete
// backends — Postgres (pg_*.go files in this package) and SQLite
// (sqlite_*.go files). Open inspects the DSN scheme and returns the
// matching implementation.
package store

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// Store is the persistence handle that handlers and RPCs depend on.
// Both Postgres and SQLite implementations satisfy it; callers select
// at boot time via Open.
type Store interface {
	// Lifecycle
	Migrate(ctx context.Context, log *slog.Logger) error
	Close()
	Backend() Backend
	TruncateForTest(ctx context.Context) error

	// Projects
	CreateProject(ctx context.Context, slug, displayName, strategy string) (Project, error)
	GetProjectBySlug(ctx context.Context, slug string) (Project, error)
	ListProjects(ctx context.Context) ([]Project, error)

	// Flags
	CreateFlag(ctx context.Context, p CreateFlagParams) (Flag, error)
	GetFlagByKey(ctx context.Context, projectID uuid.UUID, key string) (Flag, error)
	ListFlagsByProject(ctx context.Context, projectID uuid.UUID) ([]Flag, error)

	// Flag versions
	PublishFlagVersionStandalone(ctx context.Context, p PublishFlagVersionParams) (FlagVersion, error)
	GetLatestFlagVersion(ctx context.Context, flagID uuid.UUID) (FlagVersion, error)
	ListFlagVersions(ctx context.Context, flagID uuid.UUID) ([]FlagVersion, error)

	// Environments
	CreateEnvironment(ctx context.Context, projectID uuid.UUID, slug, name string) (Environment, error)
	GetEnvironmentBySlug(ctx context.Context, projectID uuid.UUID, slug string) (Environment, error)
	ListEnvironmentsByProject(ctx context.Context, projectID uuid.UUID) ([]Environment, error)

	// Segments
	CreateSegment(ctx context.Context, p CreateSegmentParams) (Segment, error)
	GetSegmentByKey(ctx context.Context, projectID uuid.UUID, key string) (Segment, error)
	ListSegmentsByProject(ctx context.Context, projectID uuid.UUID) ([]Segment, error)
	UpdateSegment(ctx context.Context, p UpdateSegmentParams) (Segment, error)

	// Snapshots
	CompileSnapshot(ctx context.Context, p CompileSnapshotParams) (Snapshot, error)
	GetSnapshotByID(ctx context.Context, projectID, id uuid.UUID) (Snapshot, error)
	GetLatestSnapshot(ctx context.Context, projectID uuid.UUID) (Snapshot, error)
	ListSnapshotsByProject(ctx context.Context, projectID uuid.UUID, limit int32) ([]Snapshot, error)
	ListLatestFlagVersions(ctx context.Context, projectID uuid.UUID) ([]LatestFlagVersion, error)

	// Audit
	AppendAudit(ctx context.Context, p AppendAuditParams) (AuditEvent, error)
	ListAuditEvents(ctx context.Context, p ListAuditEventsParams) ([]AuditEvent, error)
	WithAudit(ctx context.Context, ev AppendAuditParams, fn func(Tx) error) error
}

// Tx is the tx-scoped surface that WithAudit callbacks see. Every
// write a callback issues must go through this interface so the
// audit row and the mutation share one transaction — that guarantees
// atomicity (audit + mutation commit together or roll back together)
// and avoids second-connection deadlocks on backends with a single
// writer (notably SQLite).
type Tx interface {
	CreateProject(ctx context.Context, slug, displayName, strategy string) (Project, error)
	CreateFlag(ctx context.Context, p CreateFlagParams) (Flag, error)
	CreateEnvironment(ctx context.Context, projectID uuid.UUID, slug, name string) (Environment, error)
	CreateSegment(ctx context.Context, p CreateSegmentParams) (Segment, error)
	UpdateSegment(ctx context.Context, p UpdateSegmentParams) (Segment, error)
	PublishFlagVersion(ctx context.Context, p PublishFlagVersionParams) (FlagVersion, error)
	CompileSnapshot(ctx context.Context, p CompileSnapshotParams) (Snapshot, error)
}

// Open dials the database at the supplied DSN and returns a ready
// Store. The DSN scheme picks the backend: postgres:// (or
// postgresql://) for Postgres; sqlite:// (or file:) for SQLite.
// Callers are responsible for closing the returned Store via Close.
func Open(ctx context.Context, dsn string) (Store, error) {
	backend, driverDSN, err := parseBackend(dsn)
	if err != nil {
		return nil, err
	}
	switch backend {
	case BackendPostgres:
		return openPostgres(ctx, driverDSN)
	case BackendSQLite:
		return openSQLite(ctx, driverDSN)
	}
	return nil, fmt.Errorf("store: unsupported backend %q", backend)
}
