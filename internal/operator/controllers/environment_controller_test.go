package controllers_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/depot/falseflag/internal/operator/controllers"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

func TestEnvironmentReconciler_CreatesUpstream(t *testing.T) {
	t.Parallel()
	env := &v1alpha1.Environment{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "default"},
		Spec:       v1alpha1.EnvironmentSpec{ProjectSlug: "demo", Slug: "prod", Name: "Production"},
	}
	api := &fakeAPI{}
	cl := newClient(t, env)
	r := &controllers.EnvironmentReconciler{API: api.FakeClient(), Client: cl}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "prod", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(api.CreateEnv) != 1 || api.CreateEnv[0].Slug != "prod" {
		t.Errorf("CreateEnvironment calls = %+v", api.CreateEnv)
	}
	got := &v1alpha1.Environment{}
	if err := cl.Get(ctxBg, types.NamespacedName{Name: "prod", Namespace: "default"}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.ObservedGeneration != 0 && got.Status.ObservedGeneration != got.Generation {
		t.Errorf("ObservedGeneration = %d", got.Status.ObservedGeneration)
	}
}

func TestEnvironmentReconciler_SkipsCreateWhenExists(t *testing.T) {
	t.Parallel()
	env := &v1alpha1.Environment{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "default"},
		Spec:       v1alpha1.EnvironmentSpec{ProjectSlug: "demo", Slug: "prod", Name: "Production"},
	}
	api := &fakeAPI{EnvironmentExists: true}
	r := &controllers.EnvironmentReconciler{API: api.FakeClient(), Client: newClient(t, env)}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "prod", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(api.CreateEnv) != 0 {
		t.Errorf("expected no CreateEnvironment, got %d", len(api.CreateEnv))
	}
}
