package controllers

import (
	"errors"
	"time"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// requeueAfterSuccess is the cadence at which every reconciler
// re-runs after a successful pass. Demo-quality.
const requeueAfterSuccess = 30 * time.Second

// requeueAfterConflict is the short backoff used after the API
// returns a recoverable conflict (e.g. concurrent flag version
// publish).
const requeueAfterConflict = 1 * time.Second

// translateError maps an upstream Connect error into a controller
// result + condition for the CR's Synced/Ready conditions. Returning
// (Result, cond, nil) tells controller-runtime "I handled this, just
// requeue when you said." Returning (Result, cond, err) lets
// controller-runtime apply its default backoff and surface the error
// in metrics.
func translateError(err error) (ctrl.Result, metav1.Condition, error) {
	if err == nil {
		return ctrl.Result{RequeueAfter: requeueAfterSuccess}, syncedTrue(), nil
	}

	var ce *connect.Error
	if errors.As(err, &ce) {
		switch ce.Code() {
		case connect.CodeAlreadyExists:
			// Treated as success by callers — they fall through to
			// update. This branch is mostly defensive.
			return ctrl.Result{RequeueAfter: requeueAfterSuccess}, syncedTrue(), nil
		case connect.CodeFailedPrecondition, connect.CodeAborted:
			return ctrl.Result{RequeueAfter: requeueAfterConflict}, conditionFalse("Synced", "Conflict", ce.Message()), nil
		case connect.CodeInvalidArgument:
			return ctrl.Result{}, conditionFalse("Ready", "InvalidSpec", ce.Message()), nil
		case connect.CodeNotFound:
			return ctrl.Result{RequeueAfter: requeueAfterConflict}, conditionFalse("Synced", "UpstreamMissing", ce.Message()), nil
		}
	}

	// Unknown error — let controller-runtime back off.
	return ctrl.Result{}, conditionFalse("Synced", "Error", err.Error()), err
}

func syncedTrue() metav1.Condition {
	return metav1.Condition{
		Type:    "Synced",
		Status:  metav1.ConditionTrue,
		Reason:  "UpstreamApplied",
		Message: "operator applied the desired state to the upstream API",
	}
}

func readyTrue() metav1.Condition {
	return metav1.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "AsExpected",
		Message: "resource matches the desired state",
	}
}

func conditionFalse(typ, reason, msg string) metav1.Condition {
	return metav1.Condition{
		Type:    typ,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: msg,
	}
}

// setCondition upserts a condition into the slice, mutating in place.
// Mirrors meta.SetStatusCondition from apimachinery but kept local to
// avoid an extra import surface for one helper.
func setCondition(conds *[]metav1.Condition, cond metav1.Condition) {
	cond.LastTransitionTime = metav1.Now()
	for i := range *conds {
		if (*conds)[i].Type == cond.Type {
			if (*conds)[i].Status == cond.Status {
				cond.LastTransitionTime = (*conds)[i].LastTransitionTime
			}
			(*conds)[i] = cond
			return
		}
	}
	*conds = append(*conds, cond)
}

// nowMeta is metav1.Now wrapped so tests can override it. Unused for
// production; envtest-style tests set a fixed time.
var nowMeta = func() metav1.Time { return metav1.Now() }
