package rpc

import (
	"context"

	"connectrpc.com/connect"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
	"github.com/depot/falseflag/internal/store"
)

var _ falseflagv1connect.AuditServiceHandler = (*Handlers)(nil)

func (h *Handlers) ListAuditEvents(ctx context.Context, req *connect.Request[pb.ListAuditEventsRequest]) (*connect.Response[pb.ListAuditEventsResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	limit := req.Msg.Limit
	if limit <= 0 {
		limit = 100
	}
	params := store.ListAuditEventsParams{
		ProjectID: proj.ID,
		Limit:     limit + 1,
	}
	if req.Msg.Action != nil {
		params.Action = *req.Msg.Action
	}
	if req.Msg.Actor != nil {
		params.Actor = *req.Msg.Actor
	}
	if req.Msg.From != nil {
		params.From = req.Msg.From.AsTime()
	}
	if req.Msg.To != nil {
		params.To = req.Msg.To.AsTime()
	}
	// Connect surface does not interpret the REST cursor format; demo
	// quality allows cursorless pagination across Connect (callers can
	// page via REST if they need it). Skipping cursor decode here.

	rows, err := h.store.ListAuditEvents(ctx, params)
	if err != nil {
		return nil, connectError(err)
	}

	items := make([]*pb.AuditEvent, 0, len(rows))
	for i, ev := range rows {
		if int32(i) >= limit {
			break
		}
		items = append(items, auditEventToProto(ev))
	}
	resp := &pb.ListAuditEventsResponse{Items: items}
	return connect.NewResponse(resp), nil
}
