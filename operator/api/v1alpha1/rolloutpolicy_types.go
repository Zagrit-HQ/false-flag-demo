package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Bucketing chooses the hashing strategy for percentage-based
// rollouts. Mirrors the IR's rollout predicate shape.
type Bucketing struct {
	// Attribute names the evaluation-context attribute used to hash.
	Attribute string `json:"attribute"`

	// Strategy is currently always "fnv1a_64" — the FalseFlag IR's
	// canonical bucketing function. Kept explicit so the CRD can
	// evolve without a breaking change later.
	// +kubebuilder:validation:Enum=fnv1a_64
	// +kubebuilder:default=fnv1a_64
	Strategy string `json:"strategy,omitempty"`

	// Salt is mixed into the hash so two policies on the same
	// attribute can produce independent splits.
	Salt string `json:"salt,omitempty"`
}

// RolloutVariant is a single weighted outcome.
type RolloutVariant struct {
	// ID identifies the variant for traces and analytics.
	ID string `json:"id"`

	// Weight is the variant's share of the rollout. Weights across
	// all variants must sum to 100 — validated at translation time.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Weight int32 `json:"weight"`

	// Value is the IR value emitted when this variant wins. Shape
	// matches the parent flag's ValueType.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Value runtime.RawExtension `json:"value"`
}

// RolloutPolicySpec defines the desired state of a RolloutPolicy.
type RolloutPolicySpec struct {
	// ProjectSlug identifies the upstream project.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]{0,62}$`
	ProjectSlug string `json:"projectSlug"`

	// Name is the policy's identifier — Flag CRs reference it.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]{0,62}$`
	Name string `json:"name"`

	// Bucketing is the hash strategy for the variants.
	Bucketing Bucketing `json:"bucketing"`

	// Variants is the list of weighted outcomes.
	// +kubebuilder:validation:MinItems=1
	Variants []RolloutVariant `json:"variants"`
}

// RolloutPolicyStatus is the observed state.
type RolloutPolicyStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions         []v1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	ObservedGeneration int64          `json:"observedGeneration,omitempty"`
	LastSyncTime       *v1.Time       `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ffrollout
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectSlug`
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Variants",type=integer,JSONPath=`.spec.variants.length()`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`

// RolloutPolicy describes a reusable bucketing+variants policy that
// Flag CRs reference by name. It is never published to the upstream
// API as a standalone resource — it inlines into Flag publishes.
type RolloutPolicy struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RolloutPolicySpec   `json:"spec,omitempty"`
	Status RolloutPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RolloutPolicyList is the list wrapper for RolloutPolicy.
type RolloutPolicyList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata,omitempty"`
	Items       []RolloutPolicy `json:"items"`
}
