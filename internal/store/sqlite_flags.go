package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	dbsqlite "github.com/depot/falseflag/internal/db/sqlite"
)

// CreateFlag inserts a flag definition.
func (s *sqliteStore) CreateFlag(ctx context.Context, p CreateFlagParams) (Flag, error) {
	row, err := s.queries.CreateFlag(ctx, dbsqlite.CreateFlagParams{
		ID:           sqliteUUIDString(uuid.New()),
		ProjectID:    sqliteUUIDString(p.ProjectID),
		Key:          p.Key,
		Name:         p.Name,
		Description:  p.Description,
		ValueType:    p.ValueType,
		DefaultValue: sqliteRawMessage(p.DefaultValue),
	})
	if err != nil {
		return Flag{}, fmt.Errorf("store: create flag: %w", err)
	}
	return sqliteFlagFromRow(row), nil
}

// GetFlagByKey loads a flag by (project_id, key).
func (s *sqliteStore) GetFlagByKey(ctx context.Context, projectID uuid.UUID, key string) (Flag, error) {
	row, err := s.queries.GetFlagByKey(ctx, dbsqlite.GetFlagByKeyParams{
		ProjectID: sqliteUUIDString(projectID),
		Key:       key,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Flag{}, ErrNotFound
		}
		return Flag{}, fmt.Errorf("store: get flag: %w", err)
	}
	return sqliteFlagFromRow(row), nil
}

// ListFlagsByProject lists every flag in the project.
func (s *sqliteStore) ListFlagsByProject(ctx context.Context, projectID uuid.UUID) ([]Flag, error) {
	rows, err := s.queries.ListFlagsByProject(ctx, sqliteUUIDString(projectID))
	if err != nil {
		return nil, fmt.Errorf("store: list flags: %w", err)
	}
	out := make([]Flag, 0, len(rows))
	for _, r := range rows {
		out = append(out, sqliteFlagFromRow(r))
	}
	return out, nil
}

// publishFlagVersionTx is the shared building block for both the
// standalone and Tx-wrapped publish paths. It reads the next version
// number and inserts inside the caller-owned transaction.
func (s *sqliteStore) publishFlagVersionTx(ctx context.Context, q *dbsqlite.Queries, p PublishFlagVersionParams) (FlagVersion, error) {
	next, err := q.NextFlagVersion(ctx, sqliteUUIDString(p.FlagID))
	if err != nil {
		return FlagVersion{}, fmt.Errorf("store: next version: %w", err)
	}
	row, err := q.CreateFlagVersion(ctx, dbsqlite.CreateFlagVersionParams{
		ID:         sqliteUUIDString(uuid.New()),
		FlagID:     sqliteUUIDString(p.FlagID),
		Version:    next,
		Strategy:   p.Strategy,
		Source:     sqliteRawMessage(p.Source),
		Compiled:   sqliteRawMessage(p.Compiled),
		SourceText: sqliteStringFromString(p.SourceText),
	})
	if err != nil {
		return FlagVersion{}, fmt.Errorf("store: create version: %w", err)
	}
	return sqliteFlagVersionFromRow(row), nil
}

// PublishFlagVersionStandalone opens its own immediate-mode tx and
// delegates. Useful for callers (notably tests) that don't need an
// audit event in the same txn.
func (s *sqliteStore) PublishFlagVersionStandalone(ctx context.Context, p PublishFlagVersionParams) (FlagVersion, error) {
	var v FlagVersion
	err := s.withImmediateTx(ctx, func(tx *sql.Tx) error {
		got, err := s.publishFlagVersionTx(ctx, s.queries.WithTx(tx), p)
		if err != nil {
			return err
		}
		v = got
		return nil
	})
	if err != nil {
		return FlagVersion{}, err
	}
	return v, nil
}

// GetLatestFlagVersion returns the highest-version flag_version row.
func (s *sqliteStore) GetLatestFlagVersion(ctx context.Context, flagID uuid.UUID) (FlagVersion, error) {
	row, err := s.queries.GetLatestFlagVersion(ctx, sqliteUUIDString(flagID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return FlagVersion{}, ErrNotFound
		}
		return FlagVersion{}, fmt.Errorf("store: latest version: %w", err)
	}
	return sqliteFlagVersionFromRow(row), nil
}

// ListFlagVersions returns every flag_version row for the flag,
// newest first.
func (s *sqliteStore) ListFlagVersions(ctx context.Context, flagID uuid.UUID) ([]FlagVersion, error) {
	rows, err := s.queries.ListFlagVersions(ctx, sqliteUUIDString(flagID))
	if err != nil {
		return nil, fmt.Errorf("store: list versions: %w", err)
	}
	out := make([]FlagVersion, 0, len(rows))
	for _, r := range rows {
		out = append(out, sqliteFlagVersionFromRow(r))
	}
	return out, nil
}
