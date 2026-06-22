package controllers

import (
	"context"
	"errors"
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

// ProjectReconciler reconciles Project CRs into the FalseFlag API.
type ProjectReconciler struct {
	Log    *slog.Logger
	API    *clientapi.Client
	Client client.Client
}

// SetupWithManager registers the reconciler with mgr.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Project{}).Complete(r)
}

// Reconcile creates or updates the upstream project addressed by
// spec.projectSlug, then writes status.
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	proj := &v1alpha1.Project{}
	if err := r.Client.Get(ctx, req.NamespacedName, proj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if r.Log != nil {
		r.Log.Info("project reconcile",
			"name", proj.Name,
			"namespace", proj.Namespace,
			"projectSlug", proj.Spec.ProjectSlug,
		)
	}

	upstreamErr := r.sync(ctx, proj)
	result, syncCond, retErr := translateError(upstreamErr)

	setCondition(&proj.Status.Conditions, syncCond)
	if syncCond.Status == metav1.ConditionTrue {
		setCondition(&proj.Status.Conditions, readyTrue())
		proj.Status.ObservedGeneration = proj.Generation
		now := nowMeta()
		proj.Status.LastSyncTime = &now
	}

	if err := r.Client.Status().Update(ctx, proj); err != nil {
		// Status update failure is transient; let controller-runtime
		// retry. Do not mask the original upstream error.
		if retErr == nil {
			retErr = err
		}
	}
	return result, retErr
}

func (r *ProjectReconciler) sync(ctx context.Context, proj *v1alpha1.Project) error {
	// Try Get first; if 404, Create. Idempotent shape.
	_, getErr := r.API.Projects.GetProject(ctx, connect.NewRequest(&pb.GetProjectRequest{
		Slug: proj.Spec.ProjectSlug,
	}))
	if getErr == nil {
		// Project exists upstream. The API has no UpdateProject RPC
		// — display name + strategy changes happen only on create.
		// Demo-quality: no-op on subsequent reconciles.
		return nil
	}
	var ce *connect.Error
	if !errors.As(getErr, &ce) || ce.Code() != connect.CodeNotFound {
		return getErr
	}

	_, err := r.API.Projects.CreateProject(ctx, connect.NewRequest(&pb.CreateProjectRequest{
		Slug:           proj.Spec.ProjectSlug,
		DisplayName:    proj.Spec.DisplayName,
		ConfigStrategy: strategyFromString(proj.Spec.ConfigStrategy),
	}))
	if err != nil {
		var ce *connect.Error
		if errors.As(err, &ce) && ce.Code() == connect.CodeAlreadyExists {
			return nil
		}
		return err
	}
	return nil
}

func strategyFromString(s string) pb.Strategy {
	switch s {
	case "cel":
		return pb.Strategy_STRATEGY_CEL
	case "typescript":
		return pb.Strategy_STRATEGY_TYPESCRIPT
	case "", "json":
		return pb.Strategy_STRATEGY_JSON
	}
	return pb.Strategy_STRATEGY_UNSPECIFIED
}
