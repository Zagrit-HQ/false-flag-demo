package store

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	dbsqlite "github.com/depot/falseflag/internal/db/sqlite"
)

// Conversion helpers between dbsqlite's string/*string/string-time
// row types and our domain types. The SQLite engine stores UUIDs as
// lowercase strings, timestamps as RFC3339-formatted text with
// millisecond precision (matching the schema default), and JSON
// blobs as plain text. The convert layer normalizes all of that to
// uuid.UUID / time.Time / json.RawMessage so the API layer sees
// identical shapes regardless of backend.
//
// sqliteTimeFormat is the layout the SQLite schema's default
// strftime('%Y-%m-%dT%H:%M:%fZ', 'now') emits. We pin one format so
// both directions (storage default and Go-side INSERTs) match
// byte-for-byte.
const sqliteTimeFormat = "2006-01-02T15:04:05.000Z"

// sqliteParseTime is the inverse of sqliteFormatTime. Accepts both
// the schema-default millisecond format and the raw text the
// goose-applied migrations might surface in edge cases.
func sqliteParseTime(s string) time.Time {
	for _, layout := range []string{
		sqliteTimeFormat,
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func sqliteFormatTime(t time.Time) string {
	return t.UTC().Format(sqliteTimeFormat)
}

func sqliteNowString() string {
	return sqliteFormatTime(time.Now())
}

func sqliteUUIDString(u uuid.UUID) string { return u.String() }

func sqliteParseUUID(s string) uuid.UUID {
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return u
}

func sqliteParseNullUUID(s *string) uuid.NullUUID {
	if s == nil || *s == "" {
		return uuid.NullUUID{}
	}
	u, err := uuid.Parse(*s)
	if err != nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: u, Valid: true}
}

func sqliteFromNullUUID(u uuid.NullUUID) *string {
	if !u.Valid {
		return nil
	}
	s := u.UUID.String()
	return &s
}

func sqliteStringFromString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func sqliteStringToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// sqliteRawMessage round-trips a domain json.RawMessage through a
// string column. Empty payloads are persisted as "{}" to match the
// Postgres default-on-insert behavior.
func sqliteRawMessage(b json.RawMessage) string {
	if len(b) == 0 {
		return "{}"
	}
	return string(b)
}

func sqliteRawMessageFrom(s string) json.RawMessage {
	if s == "" {
		return json.RawMessage("{}")
	}
	return json.RawMessage(s)
}

// --- Row → domain converters ---

func sqliteProjectFromRow(r dbsqlite.Project) Project {
	return Project{
		ID:             sqliteParseUUID(r.ID),
		Slug:           r.Slug,
		DisplayName:    r.DisplayName,
		ConfigStrategy: r.ConfigStrategy,
		CreatedAt:      sqliteParseTime(r.CreatedAt),
		UpdatedAt:      sqliteParseTime(r.UpdatedAt),
	}
}

func sqliteFlagFromRow(r dbsqlite.Flag) Flag {
	return Flag{
		ID:           sqliteParseUUID(r.ID),
		ProjectID:    sqliteParseUUID(r.ProjectID),
		Key:          r.Key,
		Name:         r.Name,
		Description:  r.Description,
		ValueType:    r.ValueType,
		DefaultValue: sqliteRawMessageFrom(r.DefaultValue),
		CreatedAt:    sqliteParseTime(r.CreatedAt),
		UpdatedAt:    sqliteParseTime(r.UpdatedAt),
	}
}

func sqliteFlagVersionFromRow(r dbsqlite.FlagVersion) FlagVersion {
	return FlagVersion{
		ID:          sqliteParseUUID(r.ID),
		FlagID:      sqliteParseUUID(r.FlagID),
		Version:     int(r.Version),
		Strategy:    r.Strategy,
		Source:      sqliteRawMessageFrom(r.Source),
		Compiled:    sqliteRawMessageFrom(r.Compiled),
		SourceText:  sqliteStringToString(r.SourceText),
		PublishedAt: sqliteParseTime(r.PublishedAt),
	}
}

func sqliteEnvironmentFromRow(r dbsqlite.Environment) Environment {
	return Environment{
		ID:        sqliteParseUUID(r.ID),
		ProjectID: sqliteParseUUID(r.ProjectID),
		Slug:      r.Slug,
		Name:      r.Name,
		CreatedAt: sqliteParseTime(r.CreatedAt),
	}
}

func sqliteSegmentFromRow(r dbsqlite.Segment) Segment {
	return Segment{
		ID:          sqliteParseUUID(r.ID),
		ProjectID:   sqliteParseUUID(r.ProjectID),
		Key:         r.Key,
		Name:        r.Name,
		Description: r.Description,
		Predicate:   sqliteRawMessageFrom(r.Predicate),
		CreatedAt:   sqliteParseTime(r.CreatedAt),
		UpdatedAt:   sqliteParseTime(r.UpdatedAt),
	}
}

func sqliteSnapshotFromRow(r dbsqlite.Snapshot) Snapshot {
	return Snapshot{
		ID:            sqliteParseUUID(r.ID),
		ProjectID:     sqliteParseUUID(r.ProjectID),
		EnvironmentID: sqliteParseNullUUID(r.EnvironmentID),
		Version:       int(r.Version),
		Compiled:      sqliteRawMessageFrom(r.Compiled),
		CreatedAt:     sqliteParseTime(r.CreatedAt),
	}
}

// sqliteAuditRow captures the two sqlc-generated row shapes that hold
// an audit_events row (INSERT RETURNING and SELECT). Generics handle
// both without duplicating the body.
type sqliteAuditRow interface {
	dbsqlite.AppendAuditEventRow | dbsqlite.ListAuditEventsByProjectRow
}

func sqliteAuditFromRow[T sqliteAuditRow](r T) AuditEvent {
	switch v := any(r).(type) {
	case dbsqlite.AppendAuditEventRow:
		return AuditEvent{
			ID:        sqliteParseUUID(v.ID),
			ProjectID: sqliteParseNullUUID(v.ProjectID),
			FlagID:    sqliteParseNullUUID(v.FlagID),
			Action:    v.Action,
			Actor:     sqliteStringToString(v.Actor),
			Payload:   sqliteRawMessageFrom(v.Payload),
			CreatedAt: sqliteParseTime(v.CreatedAt),
		}
	case dbsqlite.ListAuditEventsByProjectRow:
		return AuditEvent{
			ID:        sqliteParseUUID(v.ID),
			ProjectID: sqliteParseNullUUID(v.ProjectID),
			FlagID:    sqliteParseNullUUID(v.FlagID),
			Action:    v.Action,
			Actor:     sqliteStringToString(v.Actor),
			Payload:   sqliteRawMessageFrom(v.Payload),
			CreatedAt: sqliteParseTime(v.CreatedAt),
		}
	}
	return AuditEvent{}
}

func sqliteLatestFlagVersionFromRow(r dbsqlite.ListLatestFlagVersionsRow) LatestFlagVersion {
	return LatestFlagVersion{
		FlagKey:     r.FlagKey,
		FlagID:      sqliteParseUUID(r.FlagID),
		Version:     int(r.Version),
		Strategy:    r.Strategy,
		Compiled:    sqliteRawMessageFrom(r.Compiled),
		PublishedAt: sqliteParseTime(r.PublishedAt),
	}
}
