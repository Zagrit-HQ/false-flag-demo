package rpc

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/eval"
	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
)

var _ falseflagv1connect.EvaluationServiceHandler = (*Handlers)(nil)

func (h *Handlers) Evaluate(ctx context.Context, req *connect.Request[pb.EvaluateRequest]) (*connect.Response[pb.EvaluateResponse], error) {
	compiled, version, ctxMap, err := h.loadEvaluation(ctx, req.Msg.ProjectSlug, req.Msg.Key, req.Msg.Context)
	if err != nil {
		return nil, err
	}
	d, err := eval.Evaluate(compiled, ctxMap, version)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.EvaluateResponse{Decision: decisionToProto(d)}), nil
}

func (h *Handlers) EvaluateWithTrace(ctx context.Context, req *connect.Request[pb.EvaluateWithTraceRequest]) (*connect.Response[pb.EvaluateWithTraceResponse], error) {
	compiled, version, ctxMap, err := h.loadEvaluation(ctx, req.Msg.ProjectSlug, req.Msg.Key, req.Msg.Context)
	if err != nil {
		return nil, err
	}
	d, trace, err := eval.EvaluateWithTrace(compiled, ctxMap, version)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&pb.EvaluateWithTraceResponse{
		Decision: decisionToProto(d),
		Trace:    traceToProto(trace),
	}), nil
}

func (h *Handlers) loadEvaluation(ctx context.Context, slug, key string, ctxStruct any) (*config.Compiled, int, map[string]any, error) {
	proj, err := h.store.GetProjectBySlug(ctx, slug)
	if err != nil {
		return nil, 0, nil, connectError(err)
	}
	flag, err := h.store.GetFlagByKey(ctx, proj.ID, key)
	if err != nil {
		return nil, 0, nil, connectError(err)
	}
	version, err := h.store.GetLatestFlagVersion(ctx, flag.ID)
	if err != nil {
		return nil, 0, nil, connectError(err)
	}
	compiled, err := config.Compile(config.Strategy(version.Strategy), version.Compiled)
	if err != nil {
		return nil, 0, nil, connect.NewError(connect.CodeInternal, fmt.Errorf("rehydrating IR: %w", err))
	}
	type asMap interface{ AsMap() map[string]any }
	var ctxMap map[string]any
	if m, ok := ctxStruct.(asMap); ok && m != nil {
		ctxMap = m.AsMap()
	}
	return compiled, version.Version, ctxMap, nil
}
