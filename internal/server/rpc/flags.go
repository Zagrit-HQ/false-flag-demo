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

var _ falseflagv1connect.FlagsServiceHandler = (*Handlers)(nil)

func (h *Handlers) ListFlags(ctx context.Context, req *connect.Request[pb.ListFlagsRequest]) (*connect.Response[pb.ListFlagsResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	rows, err := h.store.ListFlagsByProject(ctx, proj.ID)
	if err != nil {
		return nil, connectError(err)
	}
	items := make([]*pb.Flag, 0, len(rows))
	for _, f := range rows {
		items = append(items, flagToProto(f))
	}
	return connect.NewResponse(&pb.ListFlagsResponse{Items: items}), nil
}

func (h *Handlers) GetFlag(ctx context.Context, req *connect.Request[pb.GetFlagRequest]) (*connect.Response[pb.GetFlagResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	flag, err := h.store.GetFlagByKey(ctx, proj.ID, req.Msg.Key)
	if err != nil {
		return nil, connectError(err)
	}
	resp := &pb.GetFlagResponse{Flag: flagToProto(flag)}
	if v, err := h.store.GetLatestFlagVersion(ctx, flag.ID); err == nil {
		resp.LatestVersion = flagVersionToProto(v)
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, connectError(err)
	}
	return connect.NewResponse(resp), nil
}

func (h *Handlers) CreateFlag(ctx context.Context, req *connect.Request[pb.CreateFlagRequest]) (*connect.Response[pb.CreateFlagResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	defaultRaw, err := json.Marshal(valueToAny(req.Msg.DefaultValue))
	if err != nil {
		return nil, badRequest(fmt.Errorf("encoding default_value: %w", err))
	}

	var flag store.Flag
	err = h.store.WithAudit(ctx, store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		Action:    "create_flag",
		Actor:     actorFromHeader(req.Header().Get),
		Payload:   jsonOrEmpty(map[string]any{"key": req.Msg.Key}),
	}, func(tx store.Tx) error {
		f, err := tx.CreateFlag(ctx, store.CreateFlagParams{
			ProjectID:    proj.ID,
			Key:          req.Msg.Key,
			Name:         req.Msg.Name,
			Description:  req.Msg.Description,
			ValueType:    valueTypeFromProto(req.Msg.ValueType),
			DefaultValue: defaultRaw,
		})
		if err != nil {
			return err
		}
		flag = f
		return nil
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.CreateFlagResponse{Flag: flagToProto(flag)}), nil
}

func (h *Handlers) PublishFlagVersion(ctx context.Context, req *connect.Request[pb.PublishFlagVersionRequest]) (*connect.Response[pb.PublishFlagVersionResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	flag, err := h.store.GetFlagByKey(ctx, proj.ID, req.Msg.Key)
	if err != nil {
		return nil, connectError(err)
	}
	strategyStr := strategyFromProto(req.Msg.Strategy)
	if !config.Strategy(strategyStr).Valid() {
		return nil, badRequest(fmt.Errorf("invalid strategy %q", req.Msg.Strategy))
	}
	sourceRaw, err := structToJSONRaw(req.Msg.Source)
	if err != nil {
		return nil, badRequest(fmt.Errorf("encoding source: %w", err))
	}

	sourceText := req.Msg.SourceText
	strategy := config.Strategy(strategyStr)
	compileInput := sourceRaw
	if strategy == config.StrategyTypeScript && sourceText != "" {
		compileInput = []byte(sourceText)
	}

	resolved, err := resolveSegmentRefs(ctx, h.store, proj.ID, compileInput)
	if err != nil {
		return nil, badRequest(err)
	}

	compiled, err := config.Compile(strategy, resolved)
	if err != nil {
		if isCompileError(err) {
			return nil, compileErrorToConnect(err)
		}
		return nil, badRequest(err)
	}
	compiledRaw, err := json.Marshal(compiled.IR)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encoding compiled: %w", err))
	}
	persistedSource := sourceRaw
	if strategy == config.StrategyTypeScript && sourceText != "" {
		persistedSource = compiledRaw
	}

	var published store.FlagVersion
	err = h.store.WithAudit(ctx, store.AppendAuditParams{
		ProjectID: uuid.NullUUID{UUID: proj.ID, Valid: true},
		FlagID:    uuid.NullUUID{UUID: flag.ID, Valid: true},
		Action:    "publish_version",
		Actor:     actorFromHeader(req.Header().Get),
		Payload:   jsonOrEmpty(map[string]any{"strategy": strategyStr, "flag_key": flag.Key}),
	}, func(tx store.Tx) error {
		v, err := tx.PublishFlagVersion(ctx, store.PublishFlagVersionParams{
			FlagID:     flag.ID,
			Strategy:   strategyStr,
			Source:     persistedSource,
			Compiled:   compiledRaw,
			SourceText: sourceText,
		})
		if err != nil {
			return err
		}
		published = v
		return nil
	})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&pb.PublishFlagVersionResponse{Version: flagVersionToProto(published)}), nil
}

func (h *Handlers) ListFlagVersions(ctx context.Context, req *connect.Request[pb.ListFlagVersionsRequest]) (*connect.Response[pb.ListFlagVersionsResponse], error) {
	proj, err := h.store.GetProjectBySlug(ctx, req.Msg.ProjectSlug)
	if err != nil {
		return nil, connectError(err)
	}
	flag, err := h.store.GetFlagByKey(ctx, proj.ID, req.Msg.Key)
	if err != nil {
		return nil, connectError(err)
	}
	rows, err := h.store.ListFlagVersions(ctx, flag.ID)
	if err != nil {
		return nil, connectError(err)
	}
	items := make([]*pb.FlagVersion, 0, len(rows))
	for _, v := range rows {
		items = append(items, flagVersionToProto(v))
	}
	return connect.NewResponse(&pb.ListFlagVersionsResponse{Items: items}), nil
}
