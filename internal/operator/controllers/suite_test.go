package controllers_test

import (
	"context"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/operator/clientapi"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

// newScheme returns a runtime.Scheme with v1alpha1 registered. Each
// test builds a fresh fake client off the scheme so subtests stay
// isolated.
func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}
	return s
}

// newClient builds a fake controller-runtime client seeded with objs.
// The Status subresource is enabled for every CR so reconciler Status
// updates land where the assertions look.
func newClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()
	return fake.NewClientBuilder().
		WithScheme(newScheme(t)).
		WithObjects(objs...).
		WithStatusSubresource(
			&v1alpha1.Project{},
			&v1alpha1.Environment{},
			&v1alpha1.Segment{},
			&v1alpha1.RolloutPolicy{},
			&v1alpha1.Flag{},
			&v1alpha1.FlagBinding{},
			&v1alpha1.FlagSnapshot{},
		).
		Build()
}

// raw returns a runtime.RawExtension wrapping v's JSON encoding.
func raw(t *testing.T, v any) runtime.RawExtension {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return runtime.RawExtension{Raw: data}
}

// fakeAPI implements the subset of clientapi.Client behaviour the
// reconcilers exercise. Test code wires it into a *clientapi.Client
// via FakeClient() (below).
type fakeAPI struct {
	// Recorded calls per service. Tests assert against these.
	GetProjectCalls []*pb.GetProjectRequest
	CreateProject   []*pb.CreateProjectRequest
	GetEnvironment  []*pb.GetEnvironmentRequest
	CreateEnv       []*pb.CreateEnvironmentRequest
	GetSegment      []*pb.GetSegmentRequest
	CreateSegment   []*pb.CreateSegmentRequest
	UpdateSegment   []*pb.UpdateSegmentRequest
	GetFlag         []*pb.GetFlagRequest
	CreateFlag      []*pb.CreateFlagRequest
	PublishFlag     []*pb.PublishFlagVersionRequest
	GetLatestSnap   []*pb.GetLatestSnapshotRequest

	// Behaviour knobs. When set, override the default not-found-then-
	// create flow for the corresponding GET.
	ProjectExists     bool
	EnvironmentExists bool
	SegmentExists     bool
	FlagExists        bool

	// PublishVersion controls the version returned by PublishFlag.
	PublishVersion int32

	// LatestSnapshotVersion controls GetLatestSnapshot's response.
	LatestSnapshotVersion int32
}

// FakeClient bundles f into a *clientapi.Client by stubbing every
// service interface.
func (f *fakeAPI) FakeClient() *clientapi.Client {
	return &clientapi.Client{
		Projects:     &fakeProjects{f: f},
		Environments: &fakeEnvs{f: f},
		Segments:     &fakeSegments{f: f},
		Flags:        &fakeFlags{f: f},
		Snapshots:    &fakeSnaps{f: f},
		Audit:        nil, // operator never calls audit
	}
}

// notFound mints a connect.NotFound error for the "GET → 404 → CREATE"
// path used by every upsert reconciler.
func notFound() error {
	return connect.NewError(connect.CodeNotFound, nil)
}

// allowReconcilesToWriteStatus updates conds tracking on obj after
// reconcile. Tests use it to inspect mutated status.
func readyCond(conds []metav1.Condition) *metav1.Condition {
	for i := range conds {
		if conds[i].Type == "Ready" {
			return &conds[i]
		}
	}
	return nil
}

// ctxBg is shared across subtests.
var ctxBg = context.Background()
