package controllers_test

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
)

// --- ProjectsService ---

type fakeProjects struct{ f *fakeAPI }

func (p *fakeProjects) ListProjects(context.Context, *connect.Request[pb.ListProjectsRequest]) (*connect.Response[pb.ListProjectsResponse], error) {
	return connect.NewResponse(&pb.ListProjectsResponse{}), nil
}
func (p *fakeProjects) GetProject(_ context.Context, req *connect.Request[pb.GetProjectRequest]) (*connect.Response[pb.GetProjectResponse], error) {
	p.f.GetProjectCalls = append(p.f.GetProjectCalls, req.Msg)
	if !p.f.ProjectExists {
		return nil, notFound()
	}
	return connect.NewResponse(&pb.GetProjectResponse{Project: &pb.Project{Slug: req.Msg.Slug}}), nil
}
func (p *fakeProjects) CreateProject(_ context.Context, req *connect.Request[pb.CreateProjectRequest]) (*connect.Response[pb.CreateProjectResponse], error) {
	p.f.CreateProject = append(p.f.CreateProject, req.Msg)
	return connect.NewResponse(&pb.CreateProjectResponse{Project: &pb.Project{Slug: req.Msg.Slug, DisplayName: req.Msg.DisplayName}}), nil
}

// --- EnvironmentsService ---

type fakeEnvs struct{ f *fakeAPI }

func (e *fakeEnvs) ListEnvironments(context.Context, *connect.Request[pb.ListEnvironmentsRequest]) (*connect.Response[pb.ListEnvironmentsResponse], error) {
	return connect.NewResponse(&pb.ListEnvironmentsResponse{}), nil
}
func (e *fakeEnvs) GetEnvironment(_ context.Context, req *connect.Request[pb.GetEnvironmentRequest]) (*connect.Response[pb.GetEnvironmentResponse], error) {
	e.f.GetEnvironment = append(e.f.GetEnvironment, req.Msg)
	if !e.f.EnvironmentExists {
		return nil, notFound()
	}
	return connect.NewResponse(&pb.GetEnvironmentResponse{Environment: &pb.Environment{Slug: req.Msg.EnvSlug}}), nil
}
func (e *fakeEnvs) CreateEnvironment(_ context.Context, req *connect.Request[pb.CreateEnvironmentRequest]) (*connect.Response[pb.CreateEnvironmentResponse], error) {
	e.f.CreateEnv = append(e.f.CreateEnv, req.Msg)
	return connect.NewResponse(&pb.CreateEnvironmentResponse{Environment: &pb.Environment{Slug: req.Msg.Slug, Name: req.Msg.Name}}), nil
}

// --- SegmentsService ---

type fakeSegments struct{ f *fakeAPI }

func (s *fakeSegments) ListSegments(context.Context, *connect.Request[pb.ListSegmentsRequest]) (*connect.Response[pb.ListSegmentsResponse], error) {
	return connect.NewResponse(&pb.ListSegmentsResponse{}), nil
}
func (s *fakeSegments) GetSegment(_ context.Context, req *connect.Request[pb.GetSegmentRequest]) (*connect.Response[pb.GetSegmentResponse], error) {
	s.f.GetSegment = append(s.f.GetSegment, req.Msg)
	if !s.f.SegmentExists {
		return nil, notFound()
	}
	return connect.NewResponse(&pb.GetSegmentResponse{Segment: &pb.Segment{Key: req.Msg.SegKey}}), nil
}
func (s *fakeSegments) CreateSegment(_ context.Context, req *connect.Request[pb.CreateSegmentRequest]) (*connect.Response[pb.CreateSegmentResponse], error) {
	s.f.CreateSegment = append(s.f.CreateSegment, req.Msg)
	return connect.NewResponse(&pb.CreateSegmentResponse{Segment: &pb.Segment{Key: req.Msg.Key}}), nil
}
func (s *fakeSegments) UpdateSegment(_ context.Context, req *connect.Request[pb.UpdateSegmentRequest]) (*connect.Response[pb.UpdateSegmentResponse], error) {
	s.f.UpdateSegment = append(s.f.UpdateSegment, req.Msg)
	return connect.NewResponse(&pb.UpdateSegmentResponse{Segment: &pb.Segment{Key: req.Msg.SegKey}}), nil
}

// --- FlagsService ---

type fakeFlags struct{ f *fakeAPI }

func (fs *fakeFlags) ListFlags(context.Context, *connect.Request[pb.ListFlagsRequest]) (*connect.Response[pb.ListFlagsResponse], error) {
	return connect.NewResponse(&pb.ListFlagsResponse{}), nil
}
func (fs *fakeFlags) GetFlag(_ context.Context, req *connect.Request[pb.GetFlagRequest]) (*connect.Response[pb.GetFlagResponse], error) {
	fs.f.GetFlag = append(fs.f.GetFlag, req.Msg)
	if !fs.f.FlagExists {
		return nil, notFound()
	}
	return connect.NewResponse(&pb.GetFlagResponse{Flag: &pb.Flag{Key: req.Msg.Key}}), nil
}
func (fs *fakeFlags) CreateFlag(_ context.Context, req *connect.Request[pb.CreateFlagRequest]) (*connect.Response[pb.CreateFlagResponse], error) {
	fs.f.CreateFlag = append(fs.f.CreateFlag, req.Msg)
	return connect.NewResponse(&pb.CreateFlagResponse{Flag: &pb.Flag{Key: req.Msg.Key, Name: req.Msg.Name}}), nil
}
func (fs *fakeFlags) PublishFlagVersion(_ context.Context, req *connect.Request[pb.PublishFlagVersionRequest]) (*connect.Response[pb.PublishFlagVersionResponse], error) {
	fs.f.PublishFlag = append(fs.f.PublishFlag, req.Msg)
	version := fs.f.PublishVersion
	if version == 0 {
		version = int32(len(fs.f.PublishFlag))
	}
	return connect.NewResponse(&pb.PublishFlagVersionResponse{Version: &pb.FlagVersion{Version: version}}), nil
}
func (fs *fakeFlags) ListFlagVersions(context.Context, *connect.Request[pb.ListFlagVersionsRequest]) (*connect.Response[pb.ListFlagVersionsResponse], error) {
	return connect.NewResponse(&pb.ListFlagVersionsResponse{}), nil
}

// --- SnapshotsService ---

type fakeSnaps struct{ f *fakeAPI }

func (s *fakeSnaps) ListSnapshots(context.Context, *connect.Request[pb.ListSnapshotsRequest]) (*connect.Response[pb.ListSnapshotsResponse], error) {
	return connect.NewResponse(&pb.ListSnapshotsResponse{}), nil
}
func (s *fakeSnaps) GetSnapshot(context.Context, *connect.Request[pb.GetSnapshotRequest]) (*connect.Response[pb.GetSnapshotResponse], error) {
	return connect.NewResponse(&pb.GetSnapshotResponse{}), nil
}
func (s *fakeSnaps) GetLatestSnapshot(_ context.Context, req *connect.Request[pb.GetLatestSnapshotRequest]) (*connect.Response[pb.GetLatestSnapshotResponse], error) {
	s.f.GetLatestSnap = append(s.f.GetLatestSnap, req.Msg)
	if s.f.LatestSnapshotVersion == 0 {
		return nil, notFound()
	}
	compiled, _ := structpb.NewStruct(map[string]any{"flags": map[string]any{"a": map[string]any{}, "b": map[string]any{}}})
	return connect.NewResponse(&pb.GetLatestSnapshotResponse{
		Snapshot: &pb.Snapshot{Version: s.f.LatestSnapshotVersion, Compiled: compiled},
	}), nil
}
func (s *fakeSnaps) CompileSnapshot(context.Context, *connect.Request[pb.CompileSnapshotRequest]) (*connect.Response[pb.CompileSnapshotResponse], error) {
	return connect.NewResponse(&pb.CompileSnapshotResponse{Snapshot: &pb.Snapshot{Version: 1}}), nil
}
