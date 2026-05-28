package controllers

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/operator/clientapi"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

// FlagSnapshotReconciler polls the upstream API for the latest
// compiled snapshot and writes the version into status. Read-only
// from the user side.
type FlagSnapshotReconciler struct {
	Log    *slog.Logger
	API    *clientapi.Client
	Client client.Client
}

// SetupWithManager registers the reconciler with mgr.
func (r *FlagSnapshotReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.FlagSnapshot{}).Complete(r)
}

// Reconcile fetches the latest snapshot and writes status.
func (r *FlagSnapshotReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	snap := &v1alpha1.FlagSnapshot{}
	if err := r.Client.Get(ctx, req.NamespacedName, snap); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	resp, err := r.API.Snapshots.GetLatestSnapshot(ctx, connect.NewRequest(&pb.GetLatestSnapshotRequest{
		ProjectSlug: snap.Spec.ProjectSlug,
	}))
	result, syncCond, retErr := translateError(err)
	setCondition(&snap.Status.Conditions, syncCond)

	if err == nil && resp != nil && resp.Msg != nil && resp.Msg.Snapshot != nil {
		setCondition(&snap.Status.Conditions, metav1.Condition{
			Type:    "Compiled",
			Status:  metav1.ConditionTrue,
			Reason:  "SnapshotExists",
			Message: "latest snapshot fetched from API",
		})
		setCondition(&snap.Status.Conditions, readyTrue())
		snap.Status.CompiledVersion = resp.Msg.Snapshot.Version
		if resp.Msg.Snapshot.Compiled != nil {
			if flags, ok := resp.Msg.Snapshot.Compiled.Fields["flags"]; ok && flags.GetStructValue() != nil {
				snap.Status.FlagCount = int32(len(flags.GetStructValue().Fields))
			}
		}
		snap.Status.ObservedGeneration = snap.Generation
		now := nowMeta()
		snap.Status.LastSyncTime = &now
	} else if err != nil {
		setCondition(&snap.Status.Conditions, conditionFalse("Compiled", "Unavailable", "latest snapshot unavailable"))
	}

	if uerr := r.Client.Status().Update(ctx, snap); uerr != nil && retErr == nil {
		retErr = uerr
	}
	return result, retErr
}
