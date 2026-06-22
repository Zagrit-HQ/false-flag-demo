package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnvironmentSpec defines the desired state of an Environment.
type EnvironmentSpec struct {
	// ProjectSlug identifies the upstream FalseFlag project this
	// environment belongs to.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]{0,62}$`
	ProjectSlug string `json:"projectSlug"`

	// Slug is the environment's identifier within the project.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]{0,62}$`
	Slug string `json:"slug"`

	// Name is the human-readable label.
	Name string `json:"name"`
}

// EnvironmentStatus is the observed state of an Environment.
type EnvironmentStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions         []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	LastSyncTime       *metav1.Time       `json:"lastSyncTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ffenv
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectSlug`
// +kubebuilder:printcolumn:name="Slug",type=string,JSONPath=`.spec.slug`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Environment maps to a FalseFlag environment row within a project.
type Environment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentSpec   `json:"spec,omitempty"`
	Status EnvironmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvironmentList is the list wrapper for Environment.
type EnvironmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Environment `json:"items"`
}
