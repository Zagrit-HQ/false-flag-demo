package v1alpha1

import v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// FlagSnapshotSpec defines the desired state of a FlagSnapshot CR.
// FlagSnapshot is read-only: the operator polls the upstream API for
// the latest snapshot and writes the version into status.
type FlagSnapshotSpec struct {
	// ProjectSlug identifies the upstream project.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]{0,62}$`
	ProjectSlug string `json:"projectSlug"`
}

// FlagSnapshotStatus is the observed state of a FlagSnapshot.
type FlagSnapshotStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions         []v1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	ObservedGeneration int64          `json:"observedGeneration,omitempty"`
	LastSyncTime       *v1.Time       `json:"lastSyncTime,omitempty"`

	// CompiledVersion is the snapshot version most recently returned
	// by GetLatestSnapshot. Zero when no snapshot exists yet.
	CompiledVersion int32 `json:"compiledVersion,omitempty"`

	// FlagCount records how many flags were present in the latest
	// snapshot.
	FlagCount int32 `json:"flagCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ffsnap
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectSlug`
// +kubebuilder:printcolumn:name="Version",type=integer,JSONPath=`.status.compiledVersion`
// +kubebuilder:printcolumn:name="Flags",type=integer,JSONPath=`.status.flagCount`
// +kubebuilder:printcolumn:name="Compiled",type=string,JSONPath=`.status.conditions[?(@.type=="Compiled")].status`

// FlagSnapshot mirrors the latest immutable compiled snapshot for a
// project. Read-only on the user side; written by the operator.
type FlagSnapshot struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlagSnapshotSpec   `json:"spec,omitempty"`
	Status FlagSnapshotStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FlagSnapshotList is the list wrapper for FlagSnapshot.
type FlagSnapshotList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata,omitempty"`
	Items       []FlagSnapshot `json:"items"`
}
