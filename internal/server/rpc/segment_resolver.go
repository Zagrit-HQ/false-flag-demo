package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/store"
)

// storeSegmentResolver mirrors the REST handler's resolver — kept as a
// duplicate so the RPC package doesn't depend on internal/server/handlers.
type storeSegmentResolver struct {
	ctx       context.Context
	store     store.Store
	projectID uuid.UUID
}

func (sr *storeSegmentResolver) ResolveSegment(key string) (*config.Predicate, error) {
	seg, err := sr.store.GetSegmentByKey(sr.ctx, sr.projectID, key)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, fmt.Errorf("segment %q does not exist", key)
		}
		return nil, fmt.Errorf("looking up segment %q: %w", key, err)
	}
	var pred config.Predicate
	if err := json.Unmarshal(seg.Predicate, &pred); err != nil {
		return nil, fmt.Errorf("decoding segment %q predicate: %w", key, err)
	}
	return &pred, nil
}

func resolveSegmentRefs(ctx context.Context, s store.Store, projectID uuid.UUID, sourceRaw json.RawMessage) (json.RawMessage, error) {
	var tree config.RulesTree
	if err := json.Unmarshal(sourceRaw, &tree); err != nil {
		return sourceRaw, nil
	}
	resolver := &storeSegmentResolver{ctx: ctx, store: s, projectID: projectID}
	if err := config.ResolveSegments(&tree, resolver); err != nil {
		return nil, err
	}
	return json.Marshal(&tree)
}
