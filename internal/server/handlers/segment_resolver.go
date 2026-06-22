package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/store"
)

// storeSegmentResolver looks up segments by key against the request
// context, scoped to a single project. It implements
// config.SegmentResolver without leaking *http.Request or *store.Store
// into the config package.
type storeSegmentResolver struct {
	api       *API
	r         *http.Request
	projectID uuid.UUID
}

func (sr *storeSegmentResolver) ResolveSegment(key string) (*config.Predicate, error) {
	seg, err := sr.api.Store.GetSegmentByKey(sr.r.Context(), sr.projectID, key)
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

// resolveSegmentRefs parses the source as a RulesTree, replaces every
// `{kind: "segment", key: "<x>"}` predicate with the inlined segment
// definition, and returns the rewritten source bytes. The returned
// bytes are what gets passed to config.Compile and persisted as the
// flag version source.
func resolveSegmentRefs(r *http.Request, api *API, projectID uuid.UUID, sourceRaw json.RawMessage) (json.RawMessage, error) {
	var tree config.RulesTree
	if err := json.Unmarshal(sourceRaw, &tree); err != nil {
		// Let config.Compile produce the canonical error for unparseable
		// source — but only if we couldn't decode it at all. Return raw.
		return sourceRaw, nil
	}
	resolver := &storeSegmentResolver{api: api, r: r, projectID: projectID}
	if err := config.ResolveSegments(&tree, resolver); err != nil {
		return nil, err
	}
	resolved, err := json.Marshal(&tree)
	if err != nil {
		return nil, fmt.Errorf("re-marshalling resolved source: %w", err)
	}
	return resolved, nil
}
