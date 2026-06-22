package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/depot/falseflag/internal/config"
	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
	"github.com/depot/falseflag/internal/store"
)

var _ falseflagv1connect.SegmentsServiceHandler = (*Handlers)(nil)

func (h *Handlers) ListSegments(ctx context.Context, req *connect.Request[pb.ListSegmentsRequest]) (*connect.Response[pb.ListSegmentsResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	rows, err := h.store.ListSegmentsByProject(ctx, proj.ID)
	if err != nil {
		return nil, connectError(err)
	}
	items := make([]*pb.Segment, 0, len(rows))
	for _, s := range rows {
		items = append(items, segmentToProto(s))
	}
	return connect.NewResponse(&pb.ListSegmentsResponse{Items: items}), nil
}

func (h *Handlers) GetSegment(ctx context.Context, req *connect.Request[pb.GetSegmentRequest]) (*connect.Response[pb.GetSegmentResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	seg, err := h.store.GetSegmentByKey(ctx, proj.ID, req.Msg.SegKey)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.GetSegmentResponse{Segment: segmentToProto(seg)}), nil
}

func (h *Handlers) CreateSegment(ctx context.Context, req *connect.Request[pb.CreateSegmentRequest]) (*connect.Response[pb.CreateSegmentResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	predicateRaw, err := normalizePredicateBytes(req.Msg.Predicate)
	if err != nil {
		return nil, badRequest(err)
	}
	var seg store.Segment
	err = h.store.WithAudit(ctx, store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		Action:    "create_segment",
		Actor:     actorFromHeader(req.Header().Get),
		Payload:   jsonOrEmpty(map[string]any{"key": req.Msg.Key}),
	}, func(tx store.Tx) error {
		s, err := tx.CreateSegment(ctx, store.CreateSegmentParams{
			ProjectID:   proj.ID,
			Key:         req.Msg.Key,
			Name:        req.Msg.Name,
			Description: req.Msg.Description,
			Predicate:   predicateRaw,
		})
		if err != nil {
			return err
		}
		seg = s
		return nil
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.CreateSegmentResponse{Segment: segmentToProto(seg)}), nil
}

func (h *Handlers) UpdateSegment(ctx context.Context, req *connect.Request[pb.UpdateSegmentRequest]) (*connect.Response[pb.UpdateSegmentResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	predicateRaw, err := normalizePredicateBytes(req.Msg.Predicate)
	if err != nil {
		return nil, badRequest(err)
	}
	var seg store.Segment
	err = h.store.WithAudit(ctx, store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		Action:    "update_segment",
		Actor:     actorFromHeader(req.Header().Get),
		Payload:   jsonOrEmpty(map[string]any{"key": req.Msg.SegKey}),
	}, func(tx store.Tx) error {
		s, err := tx.UpdateSegment(ctx, store.UpdateSegmentParams{
			ProjectID:   proj.ID,
			Key:         req.Msg.SegKey,
			Name:        req.Msg.Name,
			Description: req.Msg.Description,
			Predicate:   predicateRaw,
		})
		if err != nil {
			return err
		}
		seg = s
		return nil
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.UpdateSegmentResponse{Segment: segmentToProto(seg)}), nil
}

// normalizePredicateBytes converts the Struct request field to bytes,
// validates the result, and returns the canonical JSON for storage.
// Mirrors handlers.normalizePredicate but accepts a *structpb.Struct.
func normalizePredicateBytes(s any) (json.RawMessage, error) {
	if s == nil {
		return nil, errors.New("predicate is required")
	}
	type asMap interface{ AsMap() map[string]any }
	var m map[string]any
	if mapped, ok := s.(asMap); ok && mapped != nil {
		m = mapped.AsMap()
	} else {
		return nil, fmt.Errorf("predicate must be a JSON object")
	}
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encoding predicate: %w", err)
	}
	var p config.Predicate
	if err := json.Unmarshal(bytes, &p); err != nil {
		return nil, fmt.Errorf("decoding predicate: %w", err)
	}
	if p.Kind == config.PredSegment {
		return nil, errors.New("segment predicate must not reference another segment")
	}
	if err := config.ValidatePredicate(&p, true); err != nil {
		return nil, err
	}
	return bytes, nil
}
