package tools

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
)

// fakeProjects implements falseflagv1connect.ProjectsServiceClient
// for the MCP tools tests. Only the methods read_only tools call are
// non-stub; the rest return Unimplemented so a slip in a tool that
// shouldn't touch them is loud.
type fakeProjects struct {
	falseflagv1connect.ProjectsServiceClient
	Items []*pb.Project
	Err   error
}

func (f *fakeProjects) ListProjects(_ context.Context, _ *connect.Request[pb.ListProjectsRequest]) (*connect.Response[pb.ListProjectsResponse], error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return connect.NewResponse(&pb.ListProjectsResponse{Items: f.Items}), nil
}

type fakeFlags struct {
	falseflagv1connect.FlagsServiceClient
	List          []*pb.Flag
	Get           *pb.Flag
	LatestVersion *pb.FlagVersion
	ListErr       error
	GetErr        error

	// LastListReq / LastGetReq are recorded for assertions.
	LastListReq *pb.ListFlagsRequest
	LastGetReq  *pb.GetFlagRequest
}

func (f *fakeFlags) ListFlags(_ context.Context, req *connect.Request[pb.ListFlagsRequest]) (*connect.Response[pb.ListFlagsResponse], error) {
	f.LastListReq = req.Msg
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	return connect.NewResponse(&pb.ListFlagsResponse{Items: f.List}), nil
}

func (f *fakeFlags) GetFlag(_ context.Context, req *connect.Request[pb.GetFlagRequest]) (*connect.Response[pb.GetFlagResponse], error) {
	f.LastGetReq = req.Msg
	if f.GetErr != nil {
		return nil, f.GetErr
	}
	if f.Get == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("flag not found"))
	}
	return connect.NewResponse(&pb.GetFlagResponse{Flag: f.Get, LatestVersion: f.LatestVersion}), nil
}

// notFoundErr is a helper for tests that want to inject a Connect
// not_found from upstream.
func notFoundErr(msg string) error {
	return connect.NewError(connect.CodeNotFound, errors.New(msg))
}
