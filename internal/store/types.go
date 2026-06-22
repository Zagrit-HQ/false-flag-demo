package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Domain types used at the HTTP boundary. The sqlc-generated row
// types use pgtype wrappers; we convert at the store boundary so
// handlers see Go-native types.

type Project struct {
	ID             uuid.UUID `json:"id"`
	Slug           string    `json:"slug"`
	DisplayName    string    `json:"display_name"`
	ConfigStrategy string    `json:"config_strategy"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Flag struct {
	ID           uuid.UUID       `json:"id"`
	ProjectID    uuid.UUID       `json:"project_id"`
	Key          string          `json:"key"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	ValueType    string          `json:"value_type"`
	DefaultValue json.RawMessage `json:"default_value"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type FlagVersion struct {
	ID       uuid.UUID       `json:"id"`
	FlagID   uuid.UUID       `json:"flag_id"`
	Version  int             `json:"version"`
	Strategy string          `json:"strategy"`
	Source   json.RawMessage `json:"source"`
	Compiled json.RawMessage `json:"compiled"`
	// SourceText is the raw author input (e.g. .ts/.cel/.json file
	// contents). Empty when the row predates the source_text column
	// or the publisher didn't supply it.
	SourceText  string    `json:"source_text,omitempty"`
	PublishedAt time.Time `json:"published_at"`
}

type AuditEvent struct {
	ID        uuid.UUID       `json:"id"`
	ProjectID uuid.NullUUID   `json:"project_id"`
	FlagID    uuid.NullUUID   `json:"flag_id"`
	Action    string          `json:"action"`
	Actor     string          `json:"actor,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type Environment struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Segment struct {
	ID          uuid.UUID       `json:"id"`
	ProjectID   uuid.UUID       `json:"project_id"`
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Predicate   json.RawMessage `json:"predicate"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type Snapshot struct {
	ID            uuid.UUID       `json:"id"`
	ProjectID     uuid.UUID       `json:"project_id"`
	EnvironmentID uuid.NullUUID   `json:"environment_id"`
	Version       int             `json:"version"`
	Compiled      json.RawMessage `json:"compiled"`
	CreatedAt     time.Time       `json:"created_at"`
}

// LatestFlagVersion is the slim view returned by
// Store.ListLatestFlagVersions: enough to compile a snapshot, no more.
type LatestFlagVersion struct {
	FlagKey     string
	FlagID      uuid.UUID
	Version     int
	Strategy    string
	Compiled    json.RawMessage
	PublishedAt time.Time
}
