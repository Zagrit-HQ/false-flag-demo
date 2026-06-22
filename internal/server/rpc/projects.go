package rpc

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"

	"github.com/depot/falseflag/internal/config"
	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
	"github.com/depot/falseflag/internal/store"
)

var _ falseflagv1connect.ProjectsServiceHandler = (*Handlers)(nil)

func (h *Handlers) ListProjects(ctx context.Context, _ *connect.Request[pb.ListProjectsRequest]) (*connect.Response[pb.ListProjectsResponse], error) {
	rows, err := h.store.ListProjects(ctx)
	if err != nil {
		return nil, connectError(err)
	}
	items := make([]*pb.Project, 0, len(rows))
	for _, p := range rows {
		items = append(items, projectToProto(p))
	}
	return connect.NewResponse(&pb.ListProjectsResponse{Items: items}), nil
}

func (h *Handlers) GetProject(ctx context.Context, req *connect.Request[pb.GetProjectRequest]) (*connect.Response[pb.GetProjectResponse], error) {
	p, err := h.store.GetProjectBySlug(ctx, req.Msg.Slug)
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.GetProjectResponse{Project: projectToProto(p)}), nil
}

func (h *Handlers) CreateProject(ctx context.Context, req *connect.Request[pb.CreateProjectRequest]) (*connect.Response[pb.CreateProjectResponse], error) {
	strategyStr := strategyFromProto(req.Msg.ConfigStrategy)
	if !config.Strategy(strategyStr).Valid() {
		return nil, badRequest(fmt.Errorf("invalid config_strategy %q", req.Msg.ConfigStrategy))
	}
	var proj store.Project
	err := h.store.WithAudit(ctx, store.AppendAuditParams{
		Action:  "create_project",
		Actor:   actorFromHeader(req.Header().Get),
		Payload: jsonOrEmpty(map[string]any{"slug": req.Msg.Slug}),
	}, func(tx store.Tx) error {
		p, err := tx.CreateProject(ctx, req.Msg.Slug, req.Msg.DisplayName, strategyStr)
		if err != nil {
			return err
		}
		proj = p
		return nil
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.CreateProjectResponse{Project: projectToProto(proj)}), nil
}

// jsonOrEmpty marshals v to bytes or returns `{}` on failure. Audit
// payloads are caller-controlled small maps; failure here is a
// programming error not worth bubbling up.
func jsonOrEmpty(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}
