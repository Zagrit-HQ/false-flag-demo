package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/operator/clientapi"
	"github.com/depot/falseflag/internal/operator/translate"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

const segmentFinalizer = "segment.falseflag.dev/finalizer"

// SegmentReconciler reconciles Segment CRs.
type SegmentReconciler struct {
	Log    *slog.Logger
	API    *clientapi.Client
	Client client.Client
}

// SetupWithManager registers the reconciler with mgr.
func (r *SegmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.Segment{}).Complete(r)
}

// Reconcile upserts the upstream segment. Finalizer is best-effort:
// segments inline at publish time on the upstream side, so removing
// a segment is safe for already-compiled snapshots. The API does not
// expose a Delete RPC, so the finalizer just removes itself.
func (r *SegmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	seg := &v1alpha1.Segment{}
	if err := r.Client.Get(ctx, req.NamespacedName, seg); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !seg.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(seg, segmentFinalizer) {
			controllerutil.RemoveFinalizer(seg, segmentFinalizer)
			if err := r.Client.Update(ctx, seg); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(seg, segmentFinalizer) {
		controllerutil.AddFinalizer(seg, segmentFinalizer)
		if err := r.Client.Update(ctx, seg); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Nanosecond}, nil
	}

	upstreamErr := r.sync(ctx, seg)
	result, syncCond, retErr := translateError(upstreamErr)

	setCondition(&seg.Status.Conditions, syncCond)
	if syncCond.Status == metav1.ConditionTrue {
		setCondition(&seg.Status.Conditions, readyTrue())
		seg.Status.ObservedGeneration = seg.Generation
		now := nowMeta()
		seg.Status.LastSyncTime = &now
	}

	if err := r.Client.Status().Update(ctx, seg); err != nil && retErr == nil {
		retErr = err
	}
	return result, retErr
}

func (r *SegmentReconciler) sync(ctx context.Context, seg *v1alpha1.Segment) error {
	pred, err := translate.IRForSegment(seg)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	predStruct, err := predicateToStruct(pred)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Try Get; on 404 Create; otherwise Update.
	_, getErr := r.API.Segments.GetSegment(ctx, connect.NewRequest(&pb.GetSegmentRequest{
		ProjectSlug: seg.Spec.ProjectSlug,
		SegKey:      seg.Spec.Key,
	}))
	if getErr == nil {
		_, err := r.API.Segments.UpdateSegment(ctx, connect.NewRequest(&pb.UpdateSegmentRequest{
			ProjectSlug: seg.Spec.ProjectSlug,
			SegKey:      seg.Spec.Key,
			Name:        seg.Spec.Name,
			Description: seg.Spec.Description,
			Predicate:   predStruct,
		}))
		return err
	}
	var ce *connect.Error
	if !errors.As(getErr, &ce) || ce.Code() != connect.CodeNotFound {
		return getErr
	}

	_, createErr := r.API.Segments.CreateSegment(ctx, connect.NewRequest(&pb.CreateSegmentRequest{
		ProjectSlug: seg.Spec.ProjectSlug,
		Key:         seg.Spec.Key,
		Name:        seg.Spec.Name,
		Description: seg.Spec.Description,
		Predicate:   predStruct,
	}))
	if createErr != nil {
		var ce *connect.Error
		if errors.As(createErr, &ce) && ce.Code() == connect.CodeAlreadyExists {
			return nil
		}
		return createErr
	}
	return nil
}

// predicateToStruct converts a config.Predicate value to the
// structpb.Struct the API accepts. Round-trips through JSON because
// our IR types use json.RawMessage which structpb can't reflect on.
func predicateToStruct(p any) (*structpb.Struct, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return structpb.NewStruct(m)
}
