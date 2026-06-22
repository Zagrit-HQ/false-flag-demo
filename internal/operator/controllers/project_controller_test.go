package controllers_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/depot/falseflag/internal/operator/controllers"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

func TestProjectReconciler_CreatesUpstream(t *testing.T) {
	t.Parallel()

	proj := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: v1alpha1.ProjectSpec{
			ProjectSlug:    "demo",
			DisplayName:    "Demo",
			ConfigStrategy: "json",
		},
	}
	api := &fakeAPI{ProjectExists: false}
	cl := newClient(t, proj)
	r := &controllers.ProjectReconciler{API: api.FakeClient(), Client: cl}

	res, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "demo", Namespace: "default"}})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if res.RequeueAfter == 0 {
		t.Errorf("expected requeue, got %+v", res)
	}
	if len(api.CreateProject) != 1 || api.CreateProject[0].Slug != "demo" {
		t.Errorf("CreateProject calls = %+v", api.CreateProject)
	}

	got := &v1alpha1.Project{}
	if err := cl.Get(ctxBg, types.NamespacedName{Name: "demo", Namespace: "default"}, got); err != nil {
		t.Fatalf("post-reconcile get: %v", err)
	}
	if rc := readyCond(got.Status.Conditions); rc == nil || rc.Status != metav1.ConditionTrue {
		t.Errorf("Ready condition = %+v", rc)
	}
	if got.Status.LastSyncTime == nil {
		t.Errorf("expected LastSyncTime set")
	}
}

func TestProjectReconciler_SkipsCreateWhenExists(t *testing.T) {
	t.Parallel()
	proj := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec:       v1alpha1.ProjectSpec{ProjectSlug: "demo", DisplayName: "Demo"},
	}
	api := &fakeAPI{ProjectExists: true}
	r := &controllers.ProjectReconciler{API: api.FakeClient(), Client: newClient(t, proj)}

	_, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "demo", Namespace: "default"}})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(api.CreateProject) != 0 {
		t.Errorf("expected no CreateProject, got %d", len(api.CreateProject))
	}
	if len(api.GetProjectCalls) != 1 {
		t.Errorf("expected one GetProject, got %d", len(api.GetProjectCalls))
	}
}

func TestProjectReconciler_NotFoundExits(t *testing.T) {
	t.Parallel()
	api := &fakeAPI{}
	r := &controllers.ProjectReconciler{API: api.FakeClient(), Client: newClient(t)}
	res, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "missing", Namespace: "default"}})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Errorf("expected zero requeue for missing CR, got %+v", res)
	}
}
