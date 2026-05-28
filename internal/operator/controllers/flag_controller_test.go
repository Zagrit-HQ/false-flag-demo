package controllers_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/depot/falseflag/internal/operator/controllers"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

func TestFlagReconciler_CreatesAndPublishes(t *testing.T) {
	t.Parallel()
	value := raw(t, true)
	flag := &v1alpha1.Flag{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "banner",
			Namespace:  "default",
			Finalizers: []string{"flag.falseflag.dev/finalizer"},
		},
		Spec: v1alpha1.FlagSpec{
			ProjectSlug: "demo",
			Key:         "banner",
			Name:        "Banner",
			ValueType:   "boolean",
			Default:     raw(t, false),
			Rules: []v1alpha1.FlagRule{
				{ID: "all", When: raw(t, map[string]string{"kind": "always"}), Value: &value},
			},
		},
	}
	api := &fakeAPI{PublishVersion: 5}
	cl := newClient(t, flag)
	r := &controllers.FlagReconciler{API: api.FakeClient(), Client: cl}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "banner", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(api.CreateFlag) != 1 {
		t.Errorf("CreateFlag = %+v", api.CreateFlag)
	}
	if len(api.PublishFlag) != 1 {
		t.Errorf("PublishFlag = %+v", api.PublishFlag)
	}
	got := &v1alpha1.Flag{}
	if err := cl.Get(ctxBg, types.NamespacedName{Name: "banner", Namespace: "default"}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.LastPublishedVersion != 5 {
		t.Errorf("LastPublishedVersion = %d, want 5", got.Status.LastPublishedVersion)
	}
}

func TestFlagReconciler_PublishesOncePerBinding(t *testing.T) {
	t.Parallel()
	value := raw(t, true)
	flag := &v1alpha1.Flag{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "banner",
			Namespace:  "default",
			Finalizers: []string{"flag.falseflag.dev/finalizer"},
		},
		Spec: v1alpha1.FlagSpec{
			ProjectSlug: "demo", Key: "banner", Name: "Banner", ValueType: "boolean",
			Default: raw(t, false),
			Rules: []v1alpha1.FlagRule{
				{ID: "all", When: raw(t, map[string]string{"kind": "always"}), Value: &value},
			},
			Bindings: []v1alpha1.FlagSpecBinding{
				{Environment: "staging"},
				{Environment: "prod"},
			},
		},
	}
	api := &fakeAPI{FlagExists: true}
	cl := newClient(t, flag)
	r := &controllers.FlagReconciler{API: api.FakeClient(), Client: cl}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "banner", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(api.PublishFlag) != 2 {
		t.Errorf("expected 2 publish calls (one per binding), got %d", len(api.PublishFlag))
	}
}

func TestFlagReconciler_AddsFinalizerThenRequeues(t *testing.T) {
	t.Parallel()
	flag := &v1alpha1.Flag{
		ObjectMeta: metav1.ObjectMeta{Name: "banner", Namespace: "default"},
		Spec: v1alpha1.FlagSpec{
			ProjectSlug: "demo", Key: "banner", Name: "Banner", ValueType: "boolean",
			Default: raw(t, false),
		},
	}
	api := &fakeAPI{}
	r := &controllers.FlagReconciler{API: api.FakeClient(), Client: newClient(t, flag)}
	res, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "banner", Namespace: "default"}})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if res.RequeueAfter == 0 {
		t.Errorf("expected Requeue=true after adding finalizer, got %+v", res)
	}
	if len(api.PublishFlag) != 0 {
		t.Errorf("expected no PublishFlag before finalizer settles, got %d", len(api.PublishFlag))
	}
}

func TestFlagReconciler_RolloutRefInlines(t *testing.T) {
	t.Parallel()
	rp := &v1alpha1.RolloutPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "ab", Namespace: "default"},
		Spec: v1alpha1.RolloutPolicySpec{
			ProjectSlug: "demo", Name: "ab",
			Bucketing: v1alpha1.Bucketing{Attribute: "user_id", Strategy: "fnv1a_64"},
			Variants: []v1alpha1.RolloutVariant{
				{ID: "off", Weight: 30, Value: raw(t, false)},
				{ID: "on", Weight: 70, Value: raw(t, true)},
			},
		},
	}
	flag := &v1alpha1.Flag{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "banner",
			Namespace:  "default",
			Finalizers: []string{"flag.falseflag.dev/finalizer"},
		},
		Spec: v1alpha1.FlagSpec{
			ProjectSlug: "demo", Key: "banner", Name: "Banner", ValueType: "boolean",
			Default: raw(t, false),
			Rules: []v1alpha1.FlagRule{
				{ID: "experiment", When: raw(t, map[string]string{"kind": "always"}), RolloutRef: "ab"},
			},
		},
	}
	api := &fakeAPI{}
	r := &controllers.FlagReconciler{API: api.FakeClient(), Client: newClient(t, rp, flag)}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "banner", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(api.PublishFlag) != 1 {
		t.Fatalf("PublishFlag calls = %d, want 1", len(api.PublishFlag))
	}
}
