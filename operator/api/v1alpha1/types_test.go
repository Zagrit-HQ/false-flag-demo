package v1alpha1_test

import (
	"encoding/json"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

// TestDeepCopyRoundTrip exercises the controller-gen-generated
// DeepCopy methods for every spec-bearing CRD. We construct a non-zero
// CR, deep-copy it, and assert the copy is independent and identical.
func TestDeepCopyRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		obj  runtime.Object
	}{
		{"Project", newProject(t)},
		{"Environment", newEnvironment(t)},
		{"Segment", newSegment(t)},
		{"RolloutPolicy", newRolloutPolicy(t)},
		{"Flag", newFlag(t)},
		{"FlagBinding", newFlagBinding(t)},
		{"FlagSnapshot", newFlagSnapshot(t)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cp := tc.obj.DeepCopyObject()
			if !reflect.DeepEqual(tc.obj, cp) {
				t.Fatalf("%s: deep copy differs from original", tc.name)
			}
			if cp == tc.obj {
				t.Fatalf("%s: DeepCopyObject returned the same pointer", tc.name)
			}
		})
	}
}

// TestAddToScheme ensures every type is registered. A missing
// registration is a common mistake when adding a new CRD.
func TestAddToScheme(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme: %v", err)
	}
	want := []string{
		"Project", "ProjectList",
		"Environment", "EnvironmentList",
		"Segment", "SegmentList",
		"RolloutPolicy", "RolloutPolicyList",
		"Flag", "FlagList",
		"FlagBinding", "FlagBindingList",
		"FlagSnapshot", "FlagSnapshotList",
	}
	for _, kind := range want {
		gvk := v1alpha1.GroupVersion.WithKind(kind)
		if _, err := scheme.New(gvk); err != nil {
			t.Errorf("scheme missing %s: %v", kind, err)
		}
	}
}

func raw(t *testing.T, v any) runtime.RawExtension {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return runtime.RawExtension{Raw: data}
}

func newProject(t *testing.T) *v1alpha1.Project {
	t.Helper()
	return &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		Spec: v1alpha1.ProjectSpec{
			ProjectSlug:    "demo",
			DisplayName:    "Demo",
			ConfigStrategy: "json",
		},
		Status: v1alpha1.ProjectStatus{
			Conditions: []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Synced", Message: ""}},
		},
	}
}

func newEnvironment(t *testing.T) *v1alpha1.Environment {
	t.Helper()
	return &v1alpha1.Environment{
		ObjectMeta: metav1.ObjectMeta{Name: "prod", Namespace: "default"},
		Spec:       v1alpha1.EnvironmentSpec{ProjectSlug: "demo", Slug: "prod", Name: "Production"},
	}
}

func newSegment(t *testing.T) *v1alpha1.Segment {
	t.Helper()
	pred := map[string]any{"kind": "eq", "attr": "email", "value": "wito@depot.dev"}
	return &v1alpha1.Segment{
		ObjectMeta: metav1.ObjectMeta{Name: "internal", Namespace: "default"},
		Spec: v1alpha1.SegmentSpec{
			ProjectSlug: "demo",
			Key:         "internal",
			Name:        "Internal Users",
			Predicate:   raw(t, pred),
		},
	}
}

func newRolloutPolicy(t *testing.T) *v1alpha1.RolloutPolicy {
	t.Helper()
	return &v1alpha1.RolloutPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "ab-test", Namespace: "default"},
		Spec: v1alpha1.RolloutPolicySpec{
			ProjectSlug: "demo",
			Name:        "ab-test",
			Bucketing:   v1alpha1.Bucketing{Attribute: "user_id", Strategy: "fnv1a_64"},
			Variants: []v1alpha1.RolloutVariant{
				{ID: "control", Weight: 50, Value: raw(t, false)},
				{ID: "treatment", Weight: 50, Value: raw(t, true)},
			},
		},
	}
}

func newFlag(t *testing.T) *v1alpha1.Flag {
	t.Helper()
	when := map[string]any{"kind": "always"}
	val := raw(t, true)
	return &v1alpha1.Flag{
		ObjectMeta: metav1.ObjectMeta{Name: "banner", Namespace: "default"},
		Spec: v1alpha1.FlagSpec{
			ProjectSlug: "demo",
			Key:         "banner",
			Name:        "Banner",
			ValueType:   "boolean",
			Default:     raw(t, false),
			Rules: []v1alpha1.FlagRule{
				{ID: "all", When: raw(t, when), Value: &val},
			},
		},
	}
}

func newFlagBinding(t *testing.T) *v1alpha1.FlagBinding {
	t.Helper()
	overrides := raw(t, map[string]any{"prod": true})
	return &v1alpha1.FlagBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "banner-binding", Namespace: "default"},
		Spec: v1alpha1.FlagBindingSpec{
			ProjectSlug:  "demo",
			FlagKey:      "banner",
			Environments: []string{"staging", "prod"},
			Overrides:    &overrides,
		},
	}
}

func newFlagSnapshot(t *testing.T) *v1alpha1.FlagSnapshot {
	t.Helper()
	return &v1alpha1.FlagSnapshot{
		ObjectMeta: metav1.ObjectMeta{Name: "demo-snap", Namespace: "default"},
		Spec:       v1alpha1.FlagSnapshotSpec{ProjectSlug: "demo"},
		Status:     v1alpha1.FlagSnapshotStatus{CompiledVersion: 7, FlagCount: 3},
	}
}
