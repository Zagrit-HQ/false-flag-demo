// Package clientapi exposes the operator's Connect API client to its
// reconcilers. It lives as a leaf package so internal/operator can
// import it from the runner and internal/operator/controllers can
// import it from each reconciler without a dependency cycle.
package clientapi

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
)

// Client bundles every Connect service client downstream consumers
// (operator, MCP server) need. One struct keeps caller deps trivial —
// each consumer targets exactly one upstream server.
//
// Not every consumer uses every field. Tests may leave unused fields
// nil; production callers should construct via New.
type Client struct {
	Projects     falseflagv1connect.ProjectsServiceClient
	Environments falseflagv1connect.EnvironmentsServiceClient
	Flags        falseflagv1connect.FlagsServiceClient
	Segments     falseflagv1connect.SegmentsServiceClient
	Snapshots    falseflagv1connect.SnapshotsServiceClient
	Evaluation   falseflagv1connect.EvaluationServiceClient
	Audit        falseflagv1connect.AuditServiceClient
}

// New builds a Client pointing at baseURL with every outbound request
// carrying the X-Actor header. Demo-only attribution.
func New(baseURL, actor string) *Client {
	httpClient := &http.Client{}
	opts := []connect.ClientOption{connect.WithInterceptors(actorInterceptor(actor))}
	return &Client{
		Projects:     falseflagv1connect.NewProjectsServiceClient(httpClient, baseURL, opts...),
		Environments: falseflagv1connect.NewEnvironmentsServiceClient(httpClient, baseURL, opts...),
		Flags:        falseflagv1connect.NewFlagsServiceClient(httpClient, baseURL, opts...),
		Segments:     falseflagv1connect.NewSegmentsServiceClient(httpClient, baseURL, opts...),
		Snapshots:    falseflagv1connect.NewSnapshotsServiceClient(httpClient, baseURL, opts...),
		Evaluation:   falseflagv1connect.NewEvaluationServiceClient(httpClient, baseURL, opts...),
		Audit:        falseflagv1connect.NewAuditServiceClient(httpClient, baseURL, opts...),
	}
}

// actorInterceptor injects the X-Actor header on every outbound
// request. Matches the slice-3 attribution pattern.
func actorInterceptor(actor string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if actor != "" {
				req.Header().Set("X-Actor", actor)
			}
			return next(ctx, req)
		}
	}
}
