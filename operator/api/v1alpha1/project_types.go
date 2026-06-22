package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of a FalseFlag Project.
type ProjectSpec struct {
	// ProjectSlug is the natural identifier used by the FalseFlag API.
	// It is what every child CR (Environment, Flag, ...) references.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]{0,62}$`
	ProjectSlug string `json:"projectSlug"`

	// DisplayName is the human-readable label shown in the dashboard.
	DisplayName string `json:"displayName"`

	// ConfigStrategy selects which configuration authoring mode the
	// project uses. Exactly one of "json", "cel", or "typescript".
	// +kubebuilder:validation:Enum=json;cel;typescript
	// +kubebuilder:default=json
	ConfigStrategy string `json:"configStrategy,omitempty"`
}

// ProjectStatus is the observed state of a Project.
type ProjectStatus struct {
	// Conditions track the operator's view of the resource. Always
	// includes "Ready" and "Synced".
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ObservedGeneration tracks the spec generation last reconciled.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastSyncTime records the wall-clock time of the most recent
	// successful upstream sync.
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ffproject
// +kubebuilder:printcolumn:name="Slug",type=string,JSONPath=`.spec.projectSlug`
// +kubebuilder:printcolumn:name="Strategy",type=string,JSONPath=`.spec.configStrategy`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Project is a top-level FalseFlag resource grouping environments, flags,
// segments, and rollout policies. A Project CR mirrors the upstream
// project addressed by ProjectSlug; the operator creates or updates the
// upstream project on every reconcile.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList is the list wrapper for Project.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}
