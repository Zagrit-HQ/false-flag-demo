package controllers_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/depot/falseflag/internal/operator/controllers"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

func TestRolloutPolicyReconciler_ValidatesWeights(t *testing.T) {
	t.Parallel()

	rp := &v1alpha1.RolloutPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "ab", Namespace: "default"},
		Spec: v1alpha1.RolloutPolicySpec{
			ProjectSlug: "demo", Name: "ab",
			Bucketing: v1alpha1.Bucketing{Attribute: "user_id", Strategy: "fnv1a_64"},
			Variants: []v1alpha1.RolloutVariant{
				{ID: "a", Weight: 50, Value: raw(t, false)},
				{ID: "b", Weight: 50, Value: raw(t, true)},
			},
		},
	}
	cl := newClient(t, rp)
	r := &controllers.RolloutPolicyReconciler{API: (&fakeAPI{}).FakeClient(), Client: cl}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "ab", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	got := &v1alpha1.RolloutPolicy{}
	if err := cl.Get(ctxBg, types.NamespacedName{Name: "ab", Namespace: "default"}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if rc := readyCond(got.Status.Conditions); rc == nil || rc.Status != metav1.ConditionTrue {
		t.Errorf("Ready = %+v", rc)
	}
}

func TestRolloutPolicyReconciler_RejectsInvalidWeights(t *testing.T) {
	t.Parallel()
	rp := &v1alpha1.RolloutPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "broken", Namespace: "default"},
		Spec: v1alpha1.RolloutPolicySpec{
			ProjectSlug: "demo", Name: "broken",
			Bucketing: v1alpha1.Bucketing{Attribute: "user_id", Strategy: "fnv1a_64"},
			Variants: []v1alpha1.RolloutVariant{
				{ID: "x", Weight: 40, Value: raw(t, true)},
			},
		},
	}
	cl := newClient(t, rp)
	r := &controllers.RolloutPolicyReconciler{API: (&fakeAPI{}).FakeClient(), Client: cl}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "broken", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	got := &v1alpha1.RolloutPolicy{}
	if err := cl.Get(ctxBg, types.NamespacedName{Name: "broken", Namespace: "default"}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if rc := readyCond(got.Status.Conditions); rc == nil || rc.Status != metav1.ConditionFalse {
		t.Errorf("expected Ready=False, got %+v", rc)
	}
}
