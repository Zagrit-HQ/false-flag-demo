package controllers_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/depot/falseflag/internal/operator/controllers"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

func TestSegmentReconciler_AddsFinalizerThenCreates(t *testing.T) {
	t.Parallel()
	seg := &v1alpha1.Segment{
		ObjectMeta: metav1.ObjectMeta{Name: "internal", Namespace: "default"},
		Spec: v1alpha1.SegmentSpec{
			ProjectSlug: "demo",
			Key:         "internal",
			Name:        "Internal",
			Predicate:   raw(t, map[string]any{"kind": "eq", "attr": "email", "value": "x@y"}),
		},
	}
	api := &fakeAPI{}
	cl := newClient(t, seg)
	r := &controllers.SegmentReconciler{API: api.FakeClient(), Client: cl}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "internal", Namespace: "default"}}

	// First reconcile: add finalizer, requeue.
	res, err := r.Reconcile(ctxBg, req)
	if err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	if res.RequeueAfter == 0 {
		t.Errorf("expected Requeue=true after adding finalizer, got %+v", res)
	}
	// Second reconcile: create upstream.
	if _, err := r.Reconcile(ctxBg, req); err != nil {
		t.Fatalf("second reconcile: %v", err)
	}
	if len(api.CreateSegment) != 1 || api.CreateSegment[0].Key != "internal" {
		t.Errorf("CreateSegment calls = %+v", api.CreateSegment)
	}
}

func TestSegmentReconciler_UpdatesExisting(t *testing.T) {
	t.Parallel()
	seg := &v1alpha1.Segment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "internal",
			Namespace:  "default",
			Finalizers: []string{"segment.falseflag.dev/finalizer"},
		},
		Spec: v1alpha1.SegmentSpec{
			ProjectSlug: "demo", Key: "internal",
			Predicate: raw(t, map[string]any{"kind": "eq", "attr": "email", "value": "x@y"}),
		},
	}
	api := &fakeAPI{SegmentExists: true}
	r := &controllers.SegmentReconciler{API: api.FakeClient(), Client: newClient(t, seg)}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "internal", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(api.UpdateSegment) != 1 {
		t.Errorf("UpdateSegment calls = %+v", api.UpdateSegment)
	}
	if len(api.CreateSegment) != 0 {
		t.Errorf("expected no CreateSegment, got %d", len(api.CreateSegment))
	}
}

func TestSegmentReconciler_DeleteRemovesFinalizer(t *testing.T) {
	t.Parallel()
	now := metav1.Now()
	seg := &v1alpha1.Segment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "internal",
			Namespace:         "default",
			Finalizers:        []string{"segment.falseflag.dev/finalizer"},
			DeletionTimestamp: &now,
		},
		Spec: v1alpha1.SegmentSpec{
			ProjectSlug: "demo", Key: "internal",
			Predicate: raw(t, map[string]any{"kind": "eq", "attr": "email", "value": "x@y"}),
		},
	}
	api := &fakeAPI{SegmentExists: true}
	cl := newClient(t, seg)
	r := &controllers.SegmentReconciler{API: api.FakeClient(), Client: cl}

	if _, err := r.Reconcile(ctxBg, ctrl.Request{NamespacedName: types.NamespacedName{Name: "internal", Namespace: "default"}}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if len(api.UpdateSegment) != 0 {
		t.Errorf("expected no UpdateSegment on delete, got %d", len(api.UpdateSegment))
	}
}
