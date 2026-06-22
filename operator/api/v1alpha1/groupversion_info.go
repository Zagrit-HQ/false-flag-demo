// Package v1alpha1 holds the FalseFlag CRD types. Slice 4 expands the
// surface to Project (namespaced), Environment, Flag, Segment,
// RolloutPolicy, FlagBinding, and FlagSnapshot. The IR shapes embedded
// inside Flag/Segment specs match internal/config so the operator can
// translate CR specs into upstream API requests without a separate IR
// model.
//
// +kubebuilder:object:generate=true
// +groupName=falseflag.dev
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersion is the API group/version for FalseFlag CRDs.
var GroupVersion = schema.GroupVersion{Group: "falseflag.dev", Version: "v1alpha1"}

// SchemeBuilder collects the v1alpha1 type registrations for the operator.
var SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

// AddToScheme adds v1alpha1 types to the supplied runtime.Scheme.
var AddToScheme = SchemeBuilder.AddToScheme

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&Project{}, &ProjectList{},
		&Environment{}, &EnvironmentList{},
		&Segment{}, &SegmentList{},
		&RolloutPolicy{}, &RolloutPolicyList{},
		&Flag{}, &FlagList{},
		&FlagBinding{}, &FlagBindingList{},
		&FlagSnapshot{}, &FlagSnapshotList{},
	)
	return nil
}
