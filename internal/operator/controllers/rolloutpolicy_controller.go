package controllers

import (
	"context"
	"fmt"
	"log/slog"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/depot/falseflag/internal/operator/clientapi"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

// RolloutPolicyReconciler validates a RolloutPolicy CR and writes
// status. RolloutPolicy is never published to the upstream API as a
// standalone resource — it inlines into Flag publishes.
type RolloutPolicyReconciler struct {
	Log    *slog.Logger
	API    *clientapi.Client
	Client client.Client
}

// SetupWithManager registers the reconciler with mgr.
func (r *RolloutPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.RolloutPolicy{}).Complete(r)
}

// Reconcile validates that variant weights sum to 100 and that every
// variant has a non-nil value. Demo-quality: doesn't check segment
// references because RolloutPolicy doesn't carry any.
func (r *RolloutPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rp := &v1alpha1.RolloutPolicy{}
	if err := r.Client.Get(ctx, req.NamespacedName, rp); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := validateRolloutPolicy(rp); err != nil {
		setCondition(&rp.Status.Conditions, conditionFalse("Ready", "InvalidSpec", err.Error()))
		setCondition(&rp.Status.Conditions, conditionFalse("Resolved", "InvalidSpec", err.Error()))
		_ = r.Client.Status().Update(ctx, rp)
		return ctrl.Result{}, nil
	}

	setCondition(&rp.Status.Conditions, readyTrue())
	setCondition(&rp.Status.Conditions, metav1.Condition{
		Type:    "Resolved",
		Status:  metav1.ConditionTrue,
		Reason:  "Valid",
		Message: "all variants are valid",
	})
	setCondition(&rp.Status.Conditions, syncedTrue())
	rp.Status.ObservedGeneration = rp.Generation
	now := nowMeta()
	rp.Status.LastSyncTime = &now

	if err := r.Client.Status().Update(ctx, rp); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: requeueAfterSuccess}, nil
}

func validateRolloutPolicy(rp *v1alpha1.RolloutPolicy) error {
	if len(rp.Spec.Variants) == 0 {
		return fmt.Errorf("at least one variant is required")
	}
	var total int32
	for _, v := range rp.Spec.Variants {
		if v.Value.Raw == nil {
			return fmt.Errorf("variant %q is missing a value", v.ID)
		}
		total += v.Weight
	}
	if total != 100 {
		return fmt.Errorf("variant weights must sum to 100, got %d", total)
	}
	return nil
}
