package v1alpha1

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// FlagBindingSpec ties a Flag CR to one or more Environment slugs and
// optionally supplies per-environment value overrides. A FlagBinding
// is the trigger for re-publishing a Flag across its environments.
type FlagBindingSpec struct {
	// ProjectSlug identifies the upstream project.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-]{0,62}$`
	ProjectSlug string `json:"projectSlug"`

	// FlagKey names the Flag this binding republishes.
	// +kubebuilder:validation:Pattern=`^[a-z][a-z0-9-_.]{0,62}$`
	FlagKey string `json:"flagKey"`

	// Environments is the list of environment slugs the binding
	// applies to. The reconciler publishes one flag version per
	// environment.
	// +kubebuilder:validation:MinItems=1
	Environments []string `json:"environments"`

	// Overrides maps an environment slug to a default-value
	// override. Environments missing from this map inherit the
	// parent Flag's default.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +kubebuilder:validation:Type=object
	Overrides *runtime.RawExtension `json:"overrides,omitempty"`
}

// FlagBindingStatus is the observed state of a FlagBinding.
type FlagBindingStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions         []v1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
	ObservedGeneration int64          `json:"observedGeneration,omitempty"`
	LastSyncTime       *v1.Time       `json:"lastSyncTime,omitempty"`

	// PublishedVersions records the most recent successful publish
	// version per environment. Keyed by environment slug.
	PublishedVersions map[string]int32 `json:"publishedVersions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=ffbinding
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectSlug`
// +kubebuilder:printcolumn:name="Flag",type=string,JSONPath=`.spec.flagKey`
// +kubebuilder:printcolumn:name="Envs",type=integer,JSONPath=`.spec.environments.length()`
// +kubebuilder:printcolumn:name="Published",type=string,JSONPath=`.status.conditions[?(@.type=="Published")].status`

// FlagBinding orchestrates per-environment flag publishes. When the
// binding is reconciled it republishes the referenced Flag once per
// environment, applying any per-environment overrides.
type FlagBinding struct {
	v1.TypeMeta   `json:",inline"`
	v1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlagBindingSpec   `json:"spec,omitempty"`
	Status FlagBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FlagBindingList is the list wrapper for FlagBinding.
type FlagBindingList struct {
	v1.TypeMeta `json:",inline"`
	v1.ListMeta `json:"metadata,omitempty"`
	Items       []FlagBinding `json:"items"`
}
