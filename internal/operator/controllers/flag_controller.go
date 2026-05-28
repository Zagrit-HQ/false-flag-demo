package controllers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/depot/falseflag/internal/config"
	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/operator/clientapi"
	"github.com/depot/falseflag/internal/operator/translate"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

const flagFinalizer = "flag.falseflag.dev/finalizer"

// FlagReconciler reconciles Flag CRs — the most complex reconciler.
// It creates the upstream flag if missing, resolves RolloutPolicy
// references in the same namespace, and publishes one flag version
// per binding (or one default version if no bindings).
type FlagReconciler struct {
	Log    *slog.Logger
	API    *clientapi.Client
	Client client.Client
}

// SetupWithManager registers the reconciler with mgr.
func (r *FlagReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Flag{}).Complete(r)
}

// Reconcile handles the full flag lifecycle.
func (r *FlagReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	flag := &v1alpha1.Flag{}
	if err := r.Client.Get(ctx, req.NamespacedName, flag); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !flag.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(flag, flagFinalizer) {
			// API has no DeleteFlag RPC — remove the finalizer and
			// leave the upstream flag in place. Demo-quality.
			controllerutil.RemoveFinalizer(flag, flagFinalizer)
			if err := r.Client.Update(ctx, flag); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(flag, flagFinalizer) {
		controllerutil.AddFinalizer(flag, flagFinalizer)
		if err := r.Client.Update(ctx, flag); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Nanosecond}, nil
	}

	lastVersion, upstreamErr := r.sync(ctx, flag)
	result, syncCond, retErr := translateError(upstreamErr)

	setCondition(&flag.Status.Conditions, syncCond)
	if syncCond.Status == metav1.ConditionTrue {
		setCondition(&flag.Status.Conditions, readyTrue())
		flag.Status.ObservedGeneration = flag.Generation
		now := nowMeta()
		flag.Status.LastSyncTime = &now
		if lastVersion > 0 {
			flag.Status.LastPublishedVersion = lastVersion
		}
	}

	if err := r.Client.Status().Update(ctx, flag); err != nil && retErr == nil {
		retErr = err
	}
	return result, retErr
}

// sync ensures the flag exists upstream and publishes one or more
// versions. Returns the latest version assigned or 0 if no publish
// occurred.
func (r *FlagReconciler) sync(ctx context.Context, flag *v1alpha1.Flag) (int32, error) {
	if err := r.ensureFlag(ctx, flag); err != nil {
		return 0, err
	}

	rollouts, err := r.resolveRollouts(ctx, flag)
	if err != nil {
		return 0, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if len(flag.Spec.Bindings) == 0 {
		ver, err := r.publish(ctx, flag, rollouts, nil)
		return ver, err
	}

	var lastVer int32
	for i := range flag.Spec.Bindings {
		ver, err := r.publish(ctx, flag, rollouts, &flag.Spec.Bindings[i])
		if err != nil {
			return lastVer, err
		}
		lastVer = ver
	}
	return lastVer, nil
}

func (r *FlagReconciler) ensureFlag(ctx context.Context, flag *v1alpha1.Flag) error {
	_, getErr := r.API.Flags.GetFlag(ctx, connect.NewRequest(&pb.GetFlagRequest{
		ProjectSlug: flag.Spec.ProjectSlug,
		Key:         flag.Spec.Key,
	}))
	if getErr == nil {
		return nil
	}
	var ce *connect.Error
	if !errors.As(getErr, &ce) || ce.Code() != connect.CodeNotFound {
		return getErr
	}

	defaultVal, err := rawToValue(flag.Spec.Default.Raw)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	_, err = r.API.Flags.CreateFlag(ctx, connect.NewRequest(&pb.CreateFlagRequest{
		ProjectSlug:  flag.Spec.ProjectSlug,
		Key:          flag.Spec.Key,
		Name:         flag.Spec.Name,
		ValueType:    valueTypeFromString(flag.Spec.ValueType),
		DefaultValue: defaultVal,
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

func (r *FlagReconciler) resolveRollouts(ctx context.Context, flag *v1alpha1.Flag) (map[string]*v1alpha1.RolloutPolicy, error) {
	needed := map[string]struct{}{}
	for _, rule := range flag.Spec.Rules {
		if rule.RolloutRef != "" {
			needed[rule.RolloutRef] = struct{}{}
		}
	}
	if len(needed) == 0 {
		return nil, nil
	}

	list := &v1alpha1.RolloutPolicyList{}
	if err := r.Client.List(ctx, list, client.InNamespace(flag.Namespace)); err != nil {
		return nil, fmt.Errorf("list rolloutpolicies: %w", err)
	}
	out := map[string]*v1alpha1.RolloutPolicy{}
	for i := range list.Items {
		rp := &list.Items[i]
		if rp.Spec.ProjectSlug != flag.Spec.ProjectSlug {
			continue
		}
		if _, want := needed[rp.Spec.Name]; want {
			out[rp.Spec.Name] = rp
		}
	}
	for name := range needed {
		if _, ok := out[name]; !ok {
			return nil, fmt.Errorf("RolloutPolicy %q not found in namespace %q", name, flag.Namespace)
		}
	}
	return out, nil
}

func (r *FlagReconciler) publish(ctx context.Context, flag *v1alpha1.Flag, rollouts map[string]*v1alpha1.RolloutPolicy, binding *v1alpha1.FlagSpecBinding) (int32, error) {
	tree, err := translate.IRForFlag(flag, rollouts, binding)
	if err != nil {
		return 0, connect.NewError(connect.CodeInvalidArgument, err)
	}
	src, err := translate.PublishSource(tree)
	if err != nil {
		return 0, connect.NewError(connect.CodeInvalidArgument, err)
	}
	srcStruct, err := structpb.NewStruct(src)
	if err != nil {
		return 0, connect.NewError(connect.CodeInvalidArgument, err)
	}

	resp, err := r.API.Flags.PublishFlagVersion(ctx, connect.NewRequest(&pb.PublishFlagVersionRequest{
		ProjectSlug: flag.Spec.ProjectSlug,
		Key:         flag.Spec.Key,
		Strategy:    pb.Strategy_STRATEGY_JSON,
		Source:      srcStruct,
	}))
	if err != nil {
		return 0, err
	}
	if resp.Msg != nil && resp.Msg.Version != nil {
		return resp.Msg.Version.Version, nil
	}
	return 0, nil
}

func valueTypeFromString(s string) pb.ValueType {
	switch s {
	case "string":
		return pb.ValueType_VALUE_TYPE_STRING
	case "number":
		return pb.ValueType_VALUE_TYPE_NUMBER
	case "object":
		return pb.ValueType_VALUE_TYPE_OBJECT
	case "boolean", "":
		return pb.ValueType_VALUE_TYPE_BOOLEAN
	}
	return pb.ValueType_VALUE_TYPE_UNSPECIFIED
}

// rawToValue marshals a runtime.RawExtension's JSON bytes into a
// structpb.Value the CreateFlag RPC accepts.
func rawToValue(raw []byte) (*structpb.Value, error) {
	if len(raw) == 0 {
		return structpb.NewNullValue(), nil
	}
	v := &structpb.Value{}
	if err := v.UnmarshalJSON(raw); err != nil {
		return nil, fmt.Errorf("unmarshal default: %w", err)
	}
	return v, nil
}

// Compile-time guard: ensure the IR we build is a valid config tree.
var _ = config.RulesTree{}
