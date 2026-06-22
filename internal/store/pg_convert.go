package store

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/depot/falseflag/internal/db"
)

// Conversion helpers between sqlc's pgtype-using row structs and our
// uuid.UUID/time.Time/json.RawMessage domain types. The SQLite side
// has its own convert helpers in sqlite_convert.go that bridge
// string/*string into the same domain types.

func toUUID(p pgtype.UUID) uuid.UUID {
	if !p.Valid {
		return uuid.Nil
	}
	return uuid.UUID(p.Bytes)
}

func fromUUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func toNullUUID(p pgtype.UUID) uuid.NullUUID {
	if !p.Valid {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: uuid.UUID(p.Bytes), Valid: true}
}

func fromNullUUID(u uuid.NullUUID) pgtype.UUID {
	if !u.Valid {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: u.UUID, Valid: true}
}

func projectFromRow(r db.Project) Project {
	return Project{
		ID:             toUUID(r.ID),
		Slug:           r.Slug,
		DisplayName:    r.DisplayName,
		ConfigStrategy: r.ConfigStrategy,
		CreatedAt:      r.CreatedAt.Time,
		UpdatedAt:      r.UpdatedAt.Time,
	}
}

func flagFromRow(r db.Flag) Flag {
	return Flag{
		ID:           toUUID(r.ID),
		ProjectID:    toUUID(r.ProjectID),
		Key:          r.Key,
		Name:         r.Name,
		Description:  r.Description,
		ValueType:    r.ValueType,
		DefaultValue: r.DefaultValue,
		CreatedAt:    r.CreatedAt.Time,
		UpdatedAt:    r.UpdatedAt.Time,
	}
}

func flagVersionFromRow(r db.FlagVersion) FlagVersion {
	return FlagVersion{
		ID:          toUUID(r.ID),
		FlagID:      toUUID(r.FlagID),
		Version:     int(r.Version),
		Strategy:    r.Strategy,
		Source:      r.Source,
		Compiled:    r.Compiled,
		SourceText:  textToString(r.SourceText),
		PublishedAt: r.PublishedAt.Time,
	}
}

// auditRow captures the shared shape of the two sqlc-generated row
// types that hold an audit_events row (the INSERT RETURNING row and
// the SELECT row). Generics let us scan both without duplicating the
// conversion body.
type auditRow interface {
	db.AppendAuditEventRow | db.ListAuditEventsByProjectRow
}

func auditFromRow[T auditRow](r T) AuditEvent {
	switch v := any(r).(type) {
	case db.AppendAuditEventRow:
		return AuditEvent{
			ID:        toUUID(v.ID),
			ProjectID: toNullUUID(v.ProjectID),
			FlagID:    toNullUUID(v.FlagID),
			Action:    v.Action,
			Actor:     textToString(v.Actor),
			Payload:   v.Payload,
			CreatedAt: v.CreatedAt.Time,
		}
	case db.ListAuditEventsByProjectRow:
		return AuditEvent{
			ID:        toUUID(v.ID),
			ProjectID: toNullUUID(v.ProjectID),
			FlagID:    toNullUUID(v.FlagID),
			Action:    v.Action,
			Actor:     textToString(v.Actor),
			Payload:   v.Payload,
			CreatedAt: v.CreatedAt.Time,
		}
	}
	return AuditEvent{}
}

func environmentFromRow(r db.Environment) Environment {
	return Environment{
		ID:        toUUID(r.ID),
		ProjectID: toUUID(r.ProjectID),
		Slug:      r.Slug,
		Name:      r.Name,
		CreatedAt: r.CreatedAt.Time,
	}
}

func segmentFromRow(r db.Segment) Segment {
	return Segment{
		ID:          toUUID(r.ID),
		ProjectID:   toUUID(r.ProjectID),
		Key:         r.Key,
		Name:        r.Name,
		Description: r.Description,
		Predicate:   r.Predicate,
		CreatedAt:   r.CreatedAt.Time,
		UpdatedAt:   r.UpdatedAt.Time,
	}
}

func snapshotFromRow(r db.Snapshot) Snapshot {
	return Snapshot{
		ID:            toUUID(r.ID),
		ProjectID:     toUUID(r.ProjectID),
		EnvironmentID: toNullUUID(r.EnvironmentID),
		Version:       int(r.Version),
		Compiled:      r.Compiled,
		CreatedAt:     r.CreatedAt.Time,
	}
}

func latestFlagVersionFromRow(r db.ListLatestFlagVersionsRow) LatestFlagVersion {
	return LatestFlagVersion{
		FlagKey:     r.FlagKey,
		FlagID:      toUUID(r.FlagID),
		Version:     int(r.Version),
		Strategy:    r.Strategy,
		Compiled:    r.Compiled,
		PublishedAt: r.PublishedAt.Time,
	}
}
