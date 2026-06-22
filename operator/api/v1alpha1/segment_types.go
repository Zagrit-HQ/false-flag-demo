package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SegmentSpec defines the desired state of a Segment.
type SegmentSpec struct {
	// ProjectSlug identifies the upstream project.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]{0,62}$`
	ProjectSlug string `json:"projectSlug"`

	// Key is the segment identifier within the project, used by Flag
	// rules that reference this segment via {kind: "segment", key: ...}.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]{0,62}$`
	Key string `json:"key"`

	// Name is the human-readable label.
	Name string `json:"name"`

	// Description is optional prose for the dashboard.
	Description string `json:"description,omitempty"`

	// Predicate is the IR Predicate JSON. Validated by the upstream
	// API at write time; the CRD schema is permissive.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	Predicate runtime.RawExtension `json:"predicate"`
}

// SegmentStatus is the observed state of a Segment.
type SegmentStatus struct {
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
// +kubebuilder:resource:scope=Namespaced,shortName=ffseg
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectSlug`
// +kubebuilder:printcolumn:name="Key",type=string,JSONPath=`.spec.key`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Segment maps to a FalseFlag segment row. Carries a single Predicate.
// Segments inline at publish time on the upstream side, so deleting a
// Segment CR is safe for already-compiled snapshots.
type Segment struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SegmentSpec   `json:"spec,omitempty"`
	Status SegmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SegmentList is the list wrapper for Segment.
type SegmentList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata,omitempty"`
	Items       []Segment `json:"items"`
}
