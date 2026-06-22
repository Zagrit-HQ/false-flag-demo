package translate_test

import (
	"encoding/json"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/depot/falseflag/internal/operator/translate"
	v1alpha1 "github.com/depot/falseflag/operator/api/v1alpha1"
)

func raw(t *testing.T, v any) runtime.RawExtension {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return runtime.RawExtension{Raw: data}
}

func TestIRForFlag_Static(t *testing.T) {
	t.Parallel()
	v := raw(t, true)
	flag := &v1alpha1.Flag{
		Spec: v1alpha1.FlagSpec{
			Key:       "banner",
			ValueType: "boolean",
			Default:   raw(t, false),
			Rules: []v1alpha1.FlagRule{
				{ID: "all", When: raw(t, map[string]string{"kind": "always"}), Value: &v},
			},
		},
	}
	tree, err := translate.IRForFlag(flag, nil, nil)
	if err != nil {
		t.Fatalf("IRForFlag: %v", err)
	}
	if string(tree.Default) != "false" {
		t.Errorf("default = %s, want false", tree.Default)
	}
	if len(tree.Rules) != 1 || tree.Rules[0].ID != "all" {
		t.Errorf("rules = %+v", tree.Rules)
	}
}

func TestIRForFlag_RolloutRef(t *testing.T) {
	t.Parallel()
	flag := &v1alpha1.Flag{
		Spec: v1alpha1.FlagSpec{
			Key:       "ab",
			ValueType: "boolean",
			Default:   raw(t, false),
			Rules: []v1alpha1.FlagRule{
				{ID: "experiment", When: raw(t, map[string]string{"kind": "always"}), RolloutRef: "ab-policy"},
			},
		},
	}
	policies := map[string]*v1alpha1.RolloutPolicy{
		"ab-policy": {
			Spec: v1alpha1.RolloutPolicySpec{
				Name:      "ab-policy",
				Bucketing: v1alpha1.Bucketing{Attribute: "user_id", Strategy: "fnv1a_64"},
				Variants: []v1alpha1.RolloutVariant{
					{ID: "off", Weight: 30, Value: raw(t, false)},
					{ID: "on", Weight: 70, Value: raw(t, true)},
				},
			},
		},
	}
	tree, err := translate.IRForFlag(flag, policies, nil)
	if err != nil {
		t.Fatalf("IRForFlag: %v", err)
	}
	if len(tree.Rules) != 1 {
		t.Fatalf("rules = %+v", tree.Rules)
	}
	if string(tree.Rules[0].Value) != "true" {
		t.Errorf("variant value = %s, want true (heaviest wins)", tree.Rules[0].Value)
	}
}

func TestIRForFlag_MissingRolloutPolicy(t *testing.T) {
	t.Parallel()
	flag := &v1alpha1.Flag{
		Spec: v1alpha1.FlagSpec{
			ValueType: "boolean",
			Default:   raw(t, false),
			Rules:     []v1alpha1.FlagRule{{ID: "x", When: raw(t, map[string]string{"kind": "always"}), RolloutRef: "missing"}},
		},
	}
	if _, err := translate.IRForFlag(flag, nil, nil); err == nil {
		t.Fatal("expected error for missing rollout ref")
	}
}

func TestIRForFlag_BindingOverrides(t *testing.T) {
	t.Parallel()
	parentVal := raw(t, false)
	bindDefault := raw(t, true)
	flag := &v1alpha1.Flag{
		Spec: v1alpha1.FlagSpec{
			ValueType: "boolean",
			Default:   parentVal,
			Rules:     []v1alpha1.FlagRule{},
		},
	}
	binding := &v1alpha1.FlagSpecBinding{Environment: "prod", Default: &bindDefault}
	tree, err := translate.IRForFlag(flag, nil, binding)
	if err != nil {
		t.Fatalf("IRForFlag: %v", err)
	}
	if string(tree.Default) != "true" {
		t.Errorf("override default = %s, want true", tree.Default)
	}
}

func TestIRForSegment(t *testing.T) {
	t.Parallel()
	seg := &v1alpha1.Segment{
		Spec: v1alpha1.SegmentSpec{
			Key:       "internal",
			Predicate: raw(t, map[string]any{"kind": "eq", "attr": "email", "value": "wito@depot.dev"}),
		},
	}
	pred, err := translate.IRForSegment(seg)
	if err != nil {
		t.Fatalf("IRForSegment: %v", err)
	}
	if pred.Kind != "eq" || pred.Attr != "email" {
		t.Errorf("predicate = %+v", pred)
	}
}

func TestPublishSource(t *testing.T) {
	t.Parallel()
	flag := &v1alpha1.Flag{
		Spec: v1alpha1.FlagSpec{
			ValueType: "boolean",
			Default:   raw(t, false),
		},
	}
	tree, err := translate.IRForFlag(flag, nil, nil)
	if err != nil {
		t.Fatalf("IRForFlag: %v", err)
	}
	src, err := translate.PublishSource(tree)
	if err != nil {
		t.Fatalf("PublishSource: %v", err)
	}
	if _, ok := src["value_type"]; !ok {
		t.Errorf("missing value_type in %+v", src)
	}
}
