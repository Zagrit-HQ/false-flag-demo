package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

const (
	validBooleanFlag = `{
		"value_type": "boolean",
		"default": false,
		"rules": []
	}`

	validRolloutFlag = `{
		"value_type": "boolean",
		"default": false,
		"rules": [
			{
				"id": "ramp-50",
				"when": {"kind": "rollout", "attr": "user.id", "salt": "x", "percent": 50},
				"value": true
			}
		]
	}`

	validCELFlag = `{
		"value_type": "string",
		"default": "basic",
		"rules": [
			{
				"id": "premium",
				"when": {"kind": "cel", "source": "ctx.user.age >= 21"},
				"value": "premium"
			}
		]
	}`
)

func TestValidateConfig_UnknownStrategy(t *testing.T) {
	t.Parallel()
	h := validateConfig()
	res, _, err := h(context.Background(), nil, ValidateConfigInput{Strategy: "rust", Source: validBooleanFlag})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true for unknown strategy")
	}
	if !strings.Contains(firstText(t, res), "unknown strategy") {
		t.Errorf("unexpected error body: %s", firstText(t, res))
	}
}

func TestValidateConfig_EmptySource(t *testing.T) {
	t.Parallel()
	h := validateConfig()
	res, _, err := h(context.Background(), nil, ValidateConfigInput{Strategy: "json", Source: ""})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("empty source should be a tool-level invalid, not a tool error: %s", firstText(t, res))
	}
	var out ValidateConfigOutput
	if jsonErr := json.Unmarshal([]byte(firstText(t, res)), &out); jsonErr != nil {
		t.Fatalf("invalid JSON body: %v", jsonErr)
	}
	if out.Valid {
		t.Fatal("expected Valid=false for empty source")
	}
	if len(out.Errors) == 0 || !strings.Contains(out.Errors[0], "empty source") {
		t.Errorf("expected empty source error, got %v", out.Errors)
	}
}

func TestValidateConfig_Strategies(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		strategy    string
		source      string
		wantValid   bool
		wantSummary func(*testing.T, *ValidateConfigSummary)
		wantErrIn   string
	}{
		{
			name:      "json valid",
			strategy:  "json",
			source:    validBooleanFlag,
			wantValid: true,
			wantSummary: func(t *testing.T, s *ValidateConfigSummary) {
				if s.ValueType != "boolean" {
					t.Errorf("value_type: %q", s.ValueType)
				}
				if s.RuleCount != 0 {
					t.Errorf("rule_count: %d", s.RuleCount)
				}
				if s.HasRollout {
					t.Error("has_rollout should be false")
				}
			},
		},
		{
			name:      "json with rollout",
			strategy:  "json",
			source:    validRolloutFlag,
			wantValid: true,
			wantSummary: func(t *testing.T, s *ValidateConfigSummary) {
				if !s.HasRollout {
					t.Error("expected has_rollout=true")
				}
				if s.RuleCount != 1 {
					t.Errorf("rule_count: %d", s.RuleCount)
				}
			},
		},
		{
			name:      "cel valid",
			strategy:  "cel",
			source:    validCELFlag,
			wantValid: true,
			wantSummary: func(t *testing.T, s *ValidateConfigSummary) {
				if !s.HasCEL || s.CELPrograms < 1 {
					t.Errorf("expected has_cel=true and cel_program_count>=1, got %+v", s)
				}
			},
		},
		{
			name:     "typescript valid",
			strategy: "typescript",
			// Server-side compile expects real TS source (per slice 8).
			// The JSON-IR-as-source shape from earlier slices is gone.
			source: `import { FalseFlag } from "@falseflag/config";
export default FalseFlag.flag({
  value_type: "boolean",
  default: false,
  rules: [],
});`,
			wantValid: true,
			wantSummary: func(t *testing.T, s *ValidateConfigSummary) {
				if s.ValueType != "boolean" {
					t.Errorf("value_type: %q", s.ValueType)
				}
			},
		},
		{
			name:      "json malformed",
			strategy:  "json",
			source:    `{not even json`,
			wantValid: false,
			wantErrIn: "invalid",
		},
		{
			name:     "cel bad source",
			strategy: "cel",
			source: `{
				"value_type":"string","default":"a","rules":[
					{"id":"x","when":{"kind":"cel","source":"this is not valid cel ((("},"value":"b"}
				]
			}`,
			wantValid: false,
			wantErrIn: "cel",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := validateConfig()
			res, _, err := h(context.Background(), nil, ValidateConfigInput{Strategy: tc.strategy, Source: tc.source})
			if err != nil {
				t.Fatalf("handler returned error: %v", err)
			}
			if res.IsError {
				t.Fatalf("application-level errors should not set IsError: %s", firstText(t, res))
			}
			var out ValidateConfigOutput
			if jsonErr := json.Unmarshal([]byte(firstText(t, res)), &out); jsonErr != nil {
				t.Fatalf("invalid JSON body: %v\n%s", jsonErr, firstText(t, res))
			}
			if out.Valid != tc.wantValid {
				t.Fatalf("Valid: got %v want %v (errors=%v)", out.Valid, tc.wantValid, out.Errors)
			}
			if tc.wantValid {
				if out.IRSummary == nil {
					t.Fatal("expected IRSummary on valid result")
				}
				tc.wantSummary(t, out.IRSummary)
			} else {
				if len(out.Errors) == 0 {
					t.Fatal("expected errors on invalid result")
				}
				if tc.wantErrIn != "" && !strings.Contains(strings.ToLower(strings.Join(out.Errors, " ")), tc.wantErrIn) {
					t.Errorf("expected error to contain %q, got %v", tc.wantErrIn, out.Errors)
				}
			}
		})
	}
}
