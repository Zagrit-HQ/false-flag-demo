package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/depot/falseflag/internal/db"
)

// CreateFlag inserts a flag definition.
func (s *pgStore) CreateFlag(ctx context.Context, p CreateFlagParams) (Flag, error) {
	row, err := s.queries.CreateFlag(ctx, db.CreateFlagParams{
		ID:           fromUUID(uuid.New()),
		ProjectID:    fromUUID(p.ProjectID),
		Key:          p.Key,
		Name:         p.Name,
		Description:  p.Description,
		ValueType:    p.ValueType,
		DefaultValue: p.DefaultValue,
	})
	if err != nil {
		return Flag{}, fmt.Errorf("store: create flag: %w", err)
	}
	return flagFromRow(row), nil
}

// GetFlagByKey loads a flag by (project_id, key).
func (s *pgStore) GetFlagByKey(ctx context.Context, projectID uuid.UUID, key string) (Flag, error) {
	row, err := s.queries.GetFlagByKey(ctx, db.GetFlagByKeyParams{
		ProjectID: fromUUID(projectID),
		Key:       key,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Flag{}, ErrNotFound
		}
		return Flag{}, fmt.Errorf("store: get flag: %w", err)
	}
	return flagFromRow(row), nil
}

// ListFlagsByProject lists every flag in the project.
func (s *pgStore) ListFlagsByProject(ctx context.Context, projectID uuid.UUID) ([]Flag, error) {
	rows, err := s.queries.ListFlagsByProject(ctx, fromUUID(projectID))
	if err != nil {
		return nil, fmt.Errorf("store: list flags: %w", err)
	}
	out := make([]Flag, 0, len(rows))
	for _, r := range rows {
		out = append(out, flagFromRow(r))
	}
	return out, nil
}

// publishFlagVersionTx picks the next version number and inserts a
// flag_versions row using the caller-supplied txn-scoped *db.Queries.
// It is the building block for both PublishFlagVersionStandalone and
// the Tx-side implementation that WithAudit hands to callbacks.
func (s *pgStore) publishFlagVersionTx(ctx context.Context, q *db.Queries, p PublishFlagVersionParams) (FlagVersion, error) {
	next, err := q.NextFlagVersion(ctx, fromUUID(p.FlagID))
	if err != nil {
		return FlagVersion{}, fmt.Errorf("store: next version: %w", err)
	}
	row, err := q.CreateFlagVersion(ctx, db.CreateFlagVersionParams{
		ID:         fromUUID(uuid.New()),
		FlagID:     fromUUID(p.FlagID),
		Version:    next,
		Strategy:   p.Strategy,
		Source:     p.Source,
		Compiled:   p.Compiled,
		SourceText: textFromString(p.SourceText),
	})
	if err != nil {
		return FlagVersion{}, fmt.Errorf("store: create version: %w", err)
	}
	return flagVersionFromRow(row), nil
}

// PublishFlagVersionStandalone opens its own serializable transaction
// and delegates to publishFlagVersionTx. Useful for callers (notably
// tests) that don't need an audit event in the same txn.
func (s *pgStore) PublishFlagVersionStandalone(ctx context.Context, p PublishFlagVersionParams) (FlagVersion, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return FlagVersion{}, fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	v, err := s.publishFlagVersionTx(ctx, s.queries.WithTx(tx), p)
	if err != nil {
		return FlagVersion{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return FlagVersion{}, fmt.Errorf("store: commit: %w", err)
	}
	return v, nil
}

// GetLatestFlagVersion returns the highest-version flag_version row
// for the given flag.
func (s *pgStore) GetLatestFlagVersion(ctx context.Context, flagID uuid.UUID) (FlagVersion, error) {
	row, err := s.queries.GetLatestFlagVersion(ctx, fromUUID(flagID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FlagVersion{}, ErrNotFound
		}
		return FlagVersion{}, fmt.Errorf("store: latest version: %w", err)
	}
	return flagVersionFromRow(row), nil
}

// ListFlagVersions returns every flag_version row for the flag,
// newest first.
func (s *pgStore) ListFlagVersions(ctx context.Context, flagID uuid.UUID) ([]FlagVersion, error) {
	rows, err := s.queries.ListFlagVersions(ctx, fromUUID(flagID))
	if err != nil {
		return nil, fmt.Errorf("store: list versions: %w", err)
	}
	out := make([]FlagVersion, 0, len(rows))
	for _, r := range rows {
		out = append(out, flagVersionFromRow(r))
	}
	return out, nil
}
