package controllers

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
)

// runtimeRawExtensionPtr is an alias used by the FlagBinding
// controller's override decoding. Kept named so the public method
// signature stays readable.
type runtimeRawExtensionPtr = runtime.RawExtension

// parseOverridesJSON decodes the binding.spec.overrides JSON object
// into a map of environment slug → per-environment default value.
// Unknown shapes are silently dropped — demo-quality.
func parseOverridesJSON(data []byte) map[string]*runtimeRawExtensionPtr {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	out := make(map[string]*runtimeRawExtensionPtr, len(m))
	for k, v := range m {
		out[k] = &runtime.RawExtension{Raw: v}
	}
	return out
}
