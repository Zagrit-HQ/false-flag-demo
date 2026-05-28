package controllers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/operator/clientapi"
	"github.com/depot/falseflag/internal/operator/translate"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

// FlagBindingReconciler reconciles FlagBinding CRs — applies the
// referenced Flag once per environment, optionally with overrides.
type FlagBindingReconciler struct {
	Log    *slog.Logger
	API    *clientapi.Client
	Client client.Client
}

// SetupWithManager registers the reconciler with mgr.
func (r *FlagBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.FlagBinding{}).Complete(r)
}

// Reconcile publishes one flag version per environment in spec.
func (r *FlagBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	binding := &v1alpha1.FlagBinding{}
	if err := r.Client.Get(ctx, req.NamespacedName, binding); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	flag, err := r.fetchFlag(ctx, binding)
	if err != nil {
		setCondition(&binding.Status.Conditions, conditionFalse("Ready", "FlagNotFound", err.Error()))
		_ = r.Client.Status().Update(ctx, binding)
		return ctrl.Result{RequeueAfter: requeueAfterConflict}, nil
	}

	versions, upstreamErr := r.publishAll(ctx, binding, flag)
	result, syncCond, retErr := translateError(upstreamErr)

	setCondition(&binding.Status.Conditions, syncCond)
	if syncCond.Status == metav1.ConditionTrue {
		setCondition(&binding.Status.Conditions, readyTrue())
		setCondition(&binding.Status.Conditions, metav1.Condition{
			Type:    "Published",
			Status:  metav1.ConditionTrue,
			Reason:  "AllEnvironmentsPublished",
			Message: fmt.Sprintf("published %d environments", len(versions)),
		})
		binding.Status.ObservedGeneration = binding.Generation
		now := nowMeta()
		binding.Status.LastSyncTime = &now
		binding.Status.PublishedVersions = versions
	}

	if err := r.Client.Status().Update(ctx, binding); err != nil && retErr == nil {
		retErr = err
	}
	return result, retErr
}

func (r *FlagBindingReconciler) fetchFlag(ctx context.Context, binding *v1alpha1.FlagBinding) (*v1alpha1.Flag, error) {
	list := &v1alpha1.FlagList{}
	if err := r.Client.List(ctx, list, client.InNamespace(binding.Namespace)); err != nil {
		return nil, fmt.Errorf("list flags: %w", err)
	}
	for i := range list.Items {
		f := &list.Items[i]
		if f.Spec.ProjectSlug == binding.Spec.ProjectSlug && f.Spec.Key == binding.Spec.FlagKey {
			return f, nil
		}
	}
	return nil, fmt.Errorf("flag %q not found for project %q", binding.Spec.FlagKey, binding.Spec.ProjectSlug)
}

func (r *FlagBindingReconciler) publishAll(ctx context.Context, binding *v1alpha1.FlagBinding, flag *v1alpha1.Flag) (map[string]int32, error) {
	overrides := decodeOverrides(binding)
	versions := map[string]int32{}

	for _, env := range binding.Spec.Environments {
		sb := &v1alpha1.FlagSpecBinding{Environment: env}
		if val, ok := overrides[env]; ok {
			sb.Default = val
		}
		tree, err := translate.IRForFlag(flag, nil, sb)
		if err != nil {
			return versions, connect.NewError(connect.CodeInvalidArgument, err)
		}
		src, err := translate.PublishSource(tree)
		if err != nil {
			return versions, connect.NewError(connect.CodeInvalidArgument, err)
		}
		srcStruct, err := structpb.NewStruct(src)
		if err != nil {
			return versions, connect.NewError(connect.CodeInvalidArgument, err)
		}
		resp, err := r.API.Flags.PublishFlagVersion(ctx, connect.NewRequest(&pb.PublishFlagVersionRequest{
			ProjectSlug: binding.Spec.ProjectSlug,
			Key:         binding.Spec.FlagKey,
			Strategy:    pb.Strategy_STRATEGY_JSON,
			Source:      srcStruct,
		}))
		if err != nil {
			var ce *connect.Error
			if !errors.As(err, &ce) || ce.Code() != connect.CodeAlreadyExists {
				return versions, err
			}
		}
		if resp != nil && resp.Msg != nil && resp.Msg.Version != nil {
			versions[env] = resp.Msg.Version.Version
		}
	}
	return versions, nil
}

// decodeOverrides parses the RawExtension overrides map into per-env
// raw-extension values. Returns an empty map on any error (the
// override block is optional and best-effort).
func decodeOverrides(binding *v1alpha1.FlagBinding) map[string]*runtimeRawExtensionPtr {
	if binding.Spec.Overrides == nil || len(binding.Spec.Overrides.Raw) == 0 {
		return nil
	}
	return parseOverridesJSON(binding.Spec.Overrides.Raw)
}
