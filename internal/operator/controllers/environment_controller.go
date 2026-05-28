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

// EnvironmentReconciler reconciles Environment CRs.
type EnvironmentReconciler struct {
	Log    *slog.Logger
	API    *clientapi.Client
	Client client.Client
}

// SetupWithManager registers the reconciler with mgr.
func (r *EnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Environment{}).Complete(r)
}

// Reconcile upserts the upstream environment.
func (r *EnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	env := &v1alpha1.Environment{}
	if err := r.Client.Get(ctx, req.NamespacedName, env); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	upstreamErr := r.sync(ctx, env)
	result, syncCond, retErr := translateError(upstreamErr)

	setCondition(&env.Status.Conditions, syncCond)
	if syncCond.Status == metav1.ConditionTrue {
		setCondition(&env.Status.Conditions, readyTrue())
		env.Status.ObservedGeneration = env.Generation
		now := nowMeta()
		env.Status.LastSyncTime = &now
	}

	if err := r.Client.Status().Update(ctx, env); err != nil && retErr == nil {
		retErr = err
	}
	return result, retErr
}

func (r *EnvironmentReconciler) sync(ctx context.Context, env *v1alpha1.Environment) error {
	_, getErr := r.API.Environments.GetEnvironment(ctx, connect.NewRequest(&pb.GetEnvironmentRequest{
		ProjectSlug: env.Spec.ProjectSlug,
		EnvSlug:     env.Spec.Slug,
	}))
	if getErr == nil {
		return nil
	}
	var ce *connect.Error
	if !errors.As(getErr, &ce) || ce.Code() != connect.CodeNotFound {
		return getErr
	}

	_, err := r.API.Environments.CreateEnvironment(ctx, connect.NewRequest(&pb.CreateEnvironmentRequest{
		ProjectSlug: env.Spec.ProjectSlug,
		Slug:        env.Spec.Slug,
		Name:        env.Spec.Name,
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
