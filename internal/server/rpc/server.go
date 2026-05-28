package rpc

import (
	"log/slog"
	"net/http"

	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
	"github.com/depot/falseflag/internal/store"
)

// Services bundles the RPC handler implementations. Each field is the
// `*Svc` for one of the generated service interfaces. The bundle is
// constructed once at server start and passed to Mount, which mounts
// every Connect handler onto an http.ServeMux.
type Services struct {
	Store store.Store
	Log   *slog.Logger
}

// Mount registers every Connect handler on the supplied mux. Returns
// the same mux so callers can chain.
func Mount(mux *http.ServeMux, deps Services) *http.ServeMux {
	h := &Handlers{store: deps.Store, log: deps.Log}

	mux.Handle(falseflagv1connect.NewHealthServiceHandler(h))
	mux.Handle(falseflagv1connect.NewProjectsServiceHandler(h))
	mux.Handle(falseflagv1connect.NewEnvironmentsServiceHandler(h))
	mux.Handle(falseflagv1connect.NewFlagsServiceHandler(h))
	mux.Handle(falseflagv1connect.NewSegmentsServiceHandler(h))
	mux.Handle(falseflagv1connect.NewSnapshotsServiceHandler(h))
	mux.Handle(falseflagv1connect.NewEvaluationServiceHandler(h))
	mux.Handle(falseflagv1connect.NewAuditServiceHandler(h))
	return mux
}

// Handlers implements every generated *ServiceHandler interface. One
// struct lets us share the Store and Logger across services without
// duplicating constructor boilerplate. The per-resource handler
// methods live in their respective files (projects.go, flags.go, ...).
type Handlers struct {
	store store.Store
	log   *slog.Logger
}

// Compile-time interface assertions live next to the methods they
// cover, in each per-resource file.
