package controllers_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/depot/falseflag/internal/operator/controllers"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

func TestFlagSnapshotReconciler_RecordsLatestVersion(t *testing.T) {
	t.Parallel()
	snap := &v1alpha1.FlagSnapshot{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-snap", Namespace: "default"},
		Spec:       v1alpha1.FlagSnapshotSpec{ProjectSlug: "demo"},
	}
	api := &fakeAPI{LatestSnapshotVersion: 7}
	cl := newClient(t, snap)
	r := &controllers.FlagSnapshotReconciler{API: api.FakeClient(), Client: cl}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "demo-snap", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	got := &v1alpha1.FlagSnapshot{}
	if err := cl.Get(ctxBg, types.NamespacedName{Name: "demo-snap", Namespace: "default"}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.CompiledVersion != 7 {
		t.Errorf("CompiledVersion = %d, want 7", got.Status.CompiledVersion)
	}
	if got.Status.FlagCount != 2 {
		t.Errorf("FlagCount = %d, want 2", got.Status.FlagCount)
	}
	if rc := readyCond(got.Status.Conditions); rc == nil || rc.Status != metav1.ConditionTrue {
		t.Errorf("Ready = %+v", rc)
	}
}

func TestFlagSnapshotReconciler_HandlesMissingSnapshot(t *testing.T) {
	t.Parallel()
	snap := &v1alpha1.FlagSnapshot{
		ObjectMeta: metav1.ObjectMeta{Name: "empty", Namespace: "default"},
		Spec:       v1alpha1.FlagSnapshotSpec{ProjectSlug: "demo"},
	}
	api := &fakeAPI{} // LatestSnapshotVersion=0 → notFound
	cl := newClient(t, snap)
	r := &controllers.FlagSnapshotReconciler{API: api.FakeClient(), Client: cl}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "empty", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	got := &v1alpha1.FlagSnapshot{}
	if err := cl.Get(ctxBg, types.NamespacedName{Name: "empty", Namespace: "default"}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.CompiledVersion != 0 {
		t.Errorf("CompiledVersion = %d, want 0", got.Status.CompiledVersion)
	}
}
