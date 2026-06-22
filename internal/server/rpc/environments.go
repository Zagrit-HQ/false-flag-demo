package rpc

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
	"github.com/depot/falseflag/internal/store"
)

var _ falseflagv1connect.EnvironmentsServiceHandler = (*Handlers)(nil)

func (h *Handlers) ListEnvironments(ctx context.Context, req *connect.Request[pb.ListEnvironmentsRequest]) (*connect.Response[pb.ListEnvironmentsResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	rows, err := h.store.ListEnvironmentsByProject(ctx, proj.ID)
	if err != nil {
		return nil, connectError(err)
	}
	items := make([]*pb.Environment, 0, len(rows))
	for _, e := range rows {
		items = append(items, environmentToProto(e))
	}
	return connect.NewResponse(&pb.ListEnvironmentsResponse{Items: items}), nil
}

func (h *Handlers) GetEnvironment(ctx context.Context, req *connect.Request[pb.GetEnvironmentRequest]) (*connect.Response[pb.GetEnvironmentResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	env, err := h.store.GetEnvironmentBySlug(ctx, proj.ID, req.Msg.EnvSlug)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.GetEnvironmentResponse{Environment: environmentToProto(env)}), nil
}

func (h *Handlers) CreateEnvironment(ctx context.Context, req *connect.Request[pb.CreateEnvironmentRequest]) (*connect.Response[pb.CreateEnvironmentResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	var env store.Environment
	err = h.store.WithAudit(ctx, store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		Action:    "create_environment",
		Actor:     actorFromHeader(req.Header().Get),
		Payload:   jsonOrEmpty(map[string]any{"slug": req.Msg.Slug}),
	}, func(tx store.Tx) error {
		e, err := tx.CreateEnvironment(ctx, proj.ID, req.Msg.Slug, req.Msg.Name)
		if err != nil {
			return err
		}
		env = e
		return nil
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.CreateEnvironmentResponse{Environment: environmentToProto(env)}), nil
}
