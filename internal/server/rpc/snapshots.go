package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/depot/falseflag/internal/config"
	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
	"github.com/depot/falseflag/internal/store"
)

var _ falseflagv1connect.SnapshotsServiceHandler = (*Handlers)(nil)

func (h *Handlers) ListSnapshots(ctx context.Context, req *connect.Request[pb.ListSnapshotsRequest]) (*connect.Response[pb.ListSnapshotsResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 50
	}
	rows, err := h.store.ListSnapshotsByProject(ctx, proj.ID, limit)
	if err != nil {
		return nil, connectError(err)
	}
	items := make([]*pb.Snapshot, 0, len(rows))
	for _, s := range rows {
		items = append(items, snapshotToProto(s))
	}
	return connect.NewResponse(&pb.ListSnapshotsResponse{Items: items}), nil
}

func (h *Handlers) GetSnapshot(ctx context.Context, req *connect.Request[pb.GetSnapshotRequest]) (*connect.Response[pb.GetSnapshotResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, badRequest(fmt.Errorf("invalid snapshot id: %w", err))
	}
	snap, err := h.store.GetSnapshotByID(ctx, proj.ID, id)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.GetSnapshotResponse{Snapshot: snapshotToProto(snap)}), nil
}

func (h *Handlers) GetLatestSnapshot(ctx context.Context, req *connect.Request[pb.GetLatestSnapshotRequest]) (*connect.Response[pb.GetLatestSnapshotResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	snap, err := h.store.GetLatestSnapshot(ctx, proj.ID)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.GetLatestSnapshotResponse{Snapshot: snapshotToProto(snap)}), nil
}

func (h *Handlers) CompileSnapshot(ctx context.Context, req *connect.Request[pb.CompileSnapshotRequest]) (*connect.Response[pb.CompileSnapshotResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}

	var envID uuid.NullUUID
	if req.Msg.EnvironmentSlug != nil && *req.Msg.EnvironmentSlug != "" {
		env, err := h.store.GetEnvironmentBySlug(ctx, proj.ID, *req.Msg.EnvironmentSlug)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, badRequest(fmt.Errorf("environment %q does not exist", *req.Msg.EnvironmentSlug))
			}
			return nil, connectError(err)
		}
		envID = uuid.NullUUID{UUID: env.ID, Valid: true}
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	bundle, err := compileBundle(ctx, h.store, proj.ID)
	if err != nil {
		return nil, connectError(err)
	}
	compiledRaw, err := json.Marshal(bundle)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encoding bundle: %w", err))
	}

	var snap store.Snapshot
	err = h.store.WithAudit(ctx, store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		Action:    "compile_snapshot",
		Actor:     actorFromHeader(req.Header().Get),
		Payload:   jsonOrEmpty(map[string]any{"flag_count": len(bundle.Flags)}),
	}, func(tx store.Tx) error {
		s, err := tx.CompileSnapshot(ctx, store.CompileSnapshotParams{
			ProjectID:     proj.ID,
			EnvironmentID: envID,
			Compiled:      compiledRaw,
		})
		if err != nil {
			return err
		}
		snap = s
		return nil
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.CompileSnapshotResponse{Snapshot: snapshotToProto(snap)}), nil
}

// snapshotBundle mirrors the REST handler's bundle shape. Both
// surfaces must produce identical bytes for the same input.
type snapshotBundle struct {
	Flags map[string]*config.RulesTree `json:"flags"`
}

func compileBundle(ctx context.Context, s store.Store, projectID uuid.UUID) (snapshotBundle, error) {
	bundle := snapshotBundle{Flags: map[string]*config.RulesTree{}}
	latest, err := s.ListLatestFlagVersions(ctx, projectID)
	if err != nil {
		return bundle, fmt.Errorf("listing latest versions: %w", err)
	}
	for _, lv := range latest {
		var tree config.RulesTree
		if err := json.Unmarshal(lv.Compiled, &tree); err != nil {
			return bundle, fmt.Errorf("decoding compiled IR for %s: %w", lv.FlagKey, err)
		}
		bundle.Flags[lv.FlagKey] = &tree
	}
	return bundle, nil
}
