package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// FlagRule is one CR-shaped rule that the operator translates into an
// IR Rule when publishing. Either Value or RolloutRef is set.
type FlagRule struct {
	// ID names the rule for trace/audit purposes.
	ID string `json:"id"`

	// When is the predicate that decides whether this rule fires.
	// Permissive shape; validated by the upstream API on publish.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	When runtime.RawExtension `json:"when"`

	// Value is the static value served when the rule fires. Mutually
	// exclusive with RolloutRef.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Value *runtime.RawExtension `json:"value,omitempty"`

	// RolloutRef names a RolloutPolicy CR in the same namespace whose
	// bucketing+variants are inlined at publish time. Mutually
	// exclusive with Value.
	RolloutRef string `json:"rolloutRef,omitempty"`
}

// FlagBinding is an optional per-environment override embedded in
// FlagSpec. When non-empty the operator publishes one flag version
// per binding; otherwise it publishes a single default version.
type FlagSpecBinding struct {
	// Environment names an Environment slug in the same project.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]{0,62}$`
	Environment string `json:"environment"`

	// Default overrides the parent flag's default for this
	// environment.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Default *runtime.RawExtension `json:"default,omitempty"`

	// Rules optionally replaces the parent flag's rules entirely.
	// When nil, the parent rules are inherited.
	Rules []FlagRule `json:"rules,omitempty"`
}

// FlagSpec defines the desired state of a Flag.
type FlagSpec struct {
	// ProjectSlug identifies the upstream project.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]{0,62}$`
	ProjectSlug string `json:"projectSlug"`

	// Key is the flag's identifier within the project.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-_.]{0,62}$`
	Key string `json:"key"`

	// Name is the human-readable label.
	Name string `json:"name"`

	// ValueType matches the IR ValueType enum.
	// +kubebuilder:validation:Enum=boolean;string;number;object
	// +kubebuilder:default=boolean
	ValueType string `json:"valueType"`

	// Default is the value served when no rule matches.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	Default runtime.RawExtension `json:"default"`

	// Rules is the ordered list of targeting rules.
	Rules []FlagRule `json:"rules,omitempty"`

	// Bindings optionally publishes per-environment versions. Empty
	// means publish a single default version with no environment
	// scope.
	Bindings []FlagSpecBinding `json:"bindings,omitempty"`
}

// FlagStatus is the observed state of a Flag.
type FlagStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []v1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	ObservedGeneration int64    `json:"observedGeneration,omitempty"`
	LastSyncTime       *v1.Time `json:"lastSyncTime,omitempty"`

	// LastPublishedVersion is the version returned by the most recent
	// successful PublishFlagVersion call. When per-environment
	// bindings are in use this records the last call across all of
	// them.
	LastPublishedVersion int32 `json:"lastPublishedVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ffflag
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectSlug`
// +kubebuilder:printcolumn:name="Key",type=string,JSONPath=`.spec.key`
// +kubebuilder:printcolumn:name="Version",type=integer,JSONPath=`.status.lastPublishedVersion`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Flag is the declarative representation of a feature flag. The
// operator translates Spec into the slice-2 IR RulesTree shape and
// publishes through the FalseFlag API.
type Flag struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlagSpec   `json:"spec,omitempty"`
	Status FlagStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FlagList is the list wrapper for Flag.
type FlagList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata,omitempty"`
	Items       []Flag `json:"items"`
}
