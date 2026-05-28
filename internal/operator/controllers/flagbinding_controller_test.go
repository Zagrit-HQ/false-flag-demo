package controllers_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/depot/falseflag/internal/operator/controllers"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

func TestFlagBindingReconciler_PublishesPerEnvironment(t *testing.T) {
	t.Parallel()
	flag := &v1alpha1.Flag{
		ObjectMeta: metav1.ObjectMeta{Name: "banner", Namespace: "default"},
		Spec: v1alpha1.FlagSpec{
			ProjectSlug: "demo", Key: "banner", Name: "Banner", ValueType: "boolean",
			Default: raw(t, false),
		},
	}
	binding := &v1alpha1.FlagBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "banner-binding", Namespace: "default"},
		Spec: v1alpha1.FlagBindingSpec{
			ProjectSlug:  "demo",
			FlagKey:      "banner",
			Environments: []string{"staging", "prod"},
		},
	}
	api := &fakeAPI{}
	cl := newClient(t, flag, binding)
	r := &controllers.FlagBindingReconciler{API: api.FakeClient(), Client: cl}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "banner-binding", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(api.PublishFlag) != 2 {
		t.Errorf("expected 2 publish calls, got %d", len(api.PublishFlag))
	}
	got := &v1alpha1.FlagBinding{}
	if err := cl.Get(ctxBg, types.NamespacedName{Name: "banner-binding", Namespace: "default"}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Status.PublishedVersions) != 2 {
		t.Errorf("PublishedVersions = %+v", got.Status.PublishedVersions)
	}
}

func TestFlagBindingReconciler_MissingFlagSetsCondition(t *testing.T) {
	t.Parallel()
	binding := &v1alpha1.FlagBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "orphan", Namespace: "default"},
		Spec: v1alpha1.FlagBindingSpec{
			ProjectSlug:  "demo",
			FlagKey:      "missing",
			Environments: []string{"prod"},
		},
	}
	api := &fakeAPI{}
	cl := newClient(t, binding)
	r := &controllers.FlagBindingReconciler{API: api.FakeClient(), Client: cl}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "orphan", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	got := &v1alpha1.FlagBinding{}
	if err := cl.Get(ctxBg, types.NamespacedName{Name: "orphan", Namespace: "default"}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	rc := readyCond(got.Status.Conditions)
	if rc == nil || rc.Status != metav1.ConditionFalse || rc.Reason != "FlagNotFound" {
		t.Errorf("Ready cond = %+v", rc)
	}
}
