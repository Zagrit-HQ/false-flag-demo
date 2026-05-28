package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Parameter types accepted by Store methods. They live at the package
// boundary so callers in internal/server can import them without
// pulling in any backend-specific code.

// AppendAuditParams is the input shape for AppendAudit and WithAudit.
type AppendAuditParams struct {
	ProjectID uuid.NullUUID
	FlagID    uuid.NullUUID
	Action    string
	Actor     string
	Payload   json.RawMessage
}

// ListAuditEventsParams scopes an audit search. Zero values mean
// "no filter" for every field except ProjectID and Limit.
type ListAuditEventsParams struct {
	ProjectID uuid.UUID
	Action    string
	Actor     string
	From      time.Time
	To        time.Time
	CursorTs  time.Time
	CursorID  uuid.UUID
	Limit     int32
}

// CreateFlagParams is the input shape for CreateFlag.
type CreateFlagParams struct {
	ProjectID    uuid.UUID
	Key          string
	Name         string
	Description  string
	ValueType    string
	DefaultValue json.RawMessage
}

// PublishFlagVersionParams is the input shape for PublishFlagVersion.
type PublishFlagVersionParams struct {
	FlagID     uuid.UUID
	Strategy   string
	Source     json.RawMessage
	Compiled   json.RawMessage
	SourceText string
}

// CreateSegmentParams is the input shape for CreateSegment.
type CreateSegmentParams struct {
	ProjectID   uuid.UUID
	Key         string
	Name        string
	Description string
	Predicate   json.RawMessage
}

// UpdateSegmentParams is the input shape for UpdateSegment.
type UpdateSegmentParams struct {
	ProjectID   uuid.UUID
	Key         string
	Name        string
	Description string
	Predicate   json.RawMessage
}

// CompileSnapshotParams is the input shape for CompileSnapshot.
type CompileSnapshotParams struct {
	ProjectID     uuid.UUID
	EnvironmentID uuid.NullUUID
	Compiled      json.RawMessage
}
