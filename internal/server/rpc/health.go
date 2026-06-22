package rpc

import (
	"context"

	"connectrpc.com/connect"

	"github.com/depot/falseflag/internal/buildinfo"
	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
)

var _ falseflagv1connect.HealthServiceHandler = (*Handlers)(nil)

func (h *Handlers) Check(_ context.Context, _ *connect.Request[pb.CheckRequest]) (*connect.Response[pb.CheckResponse], error) {
	return connect.NewResponse(&pb.CheckResponse{
		Status:  pb.CheckResponse_SERVING_STATUS_SERVING,
		Service: buildinfo.ServiceName("api"),
		Version: buildinfo.Version,
	}), nil
}
