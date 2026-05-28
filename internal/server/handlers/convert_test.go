package handlers

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/depot/falseflag/internal/eval"
	"github.com/depot/falseflag/internal/store"
)

// fixedID returns a deterministic UUID for test cases by index. The
// converters are pure functions of their input, so deterministic IDs
// keep failures easy to reproduce.
func fixedID(i int) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("convert-test-%d", i)))
}

func fixedTime(i int) time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Minute)
}

func TestProjectToAPI(t *testing.T) {
	t.Parallel()
	strategies := []string{"json", "cel", "typescript"}
	for i := 0; i < 64; i++ {
		i := i
		t.Run(fmt.Sprintf("case-%02d", i), func(t *testing.T) {
			t.Parallel()
			p := store.Project{
				ID:             fixedID(i),
				Slug:           fmt.Sprintf("proj-%d", i),
				DisplayName:    fmt.Sprintf("Project %d", i),
				ConfigStrategy: strategies[i%len(strategies)],
				CreatedAt:      fixedTime(i),
				UpdatedAt:      fixedTime(i + 1),
			}
			out := projectToAPI(p)
			if out.Id != openapi_types.UUID(p.ID) {
				t.Errorf("Id mismatch")
			}
			if out.Slug != p.Slug {
				t.Errorf("Slug = %q want %q", out.Slug, p.Slug)
			}
			if out.DisplayName != p.DisplayName {
				t.Errorf("DisplayName = %q", out.DisplayName)
			}
			if string(out.ConfigStrategy) != p.ConfigStrategy {
				t.Errorf("ConfigStrategy = %q", out.ConfigStrategy)
			}
			if !out.CreatedAt.Equal(p.CreatedAt) {
				t.Errorf("CreatedAt mismatch")
			}
			if !out.UpdatedAt.Equal(p.UpdatedAt) {
				t.Errorf("UpdatedAt mismatch")
			}
		})
	}
}

func TestFlagToAPI(t *testing.T) {
	t.Parallel()
	valueTypes := []string{"boolean", "string", "number", "json"}
	defaults := []json.RawMessage{
		json.RawMessage(`true`),
		json.RawMessage(`"alpha"`),
		json.RawMessage(`42`),
		json.RawMessage(`{"k":"v"}`),
		json.RawMessage(`null`),
		nil,
	}
	for i := 0; i < 96; i++ {
		i := i
		t.Run(fmt.Sprintf("case-%02d", i), func(t *testing.T) {
			t.Parallel()
			f := store.Flag{
				ID:           fixedID(i),
				ProjectID:    fixedID(i + 1000),
				Key:          fmt.Sprintf("flag-%d", i),
				Name:         fmt.Sprintf("Flag %d", i),
				Description:  fmt.Sprintf("desc %d", i),
				ValueType:    valueTypes[i%len(valueTypes)],
				DefaultValue: defaults[i%len(defaults)],
				CreatedAt:    fixedTime(i),
				UpdatedAt:    fixedTime(i + 5),
			}
			out := flagToAPI(f)
			if out.Key != f.Key {
				t.Errorf("Key = %q", out.Key)
			}
			if out.Id != openapi_types.UUID(f.ID) {
				t.Errorf("Id mismatch")
			}
			if string(out.ValueType) != f.ValueType {
				t.Errorf("ValueType = %q", out.ValueType)
			}
		})
	}
}

func TestFlagVersionToAPI_SourceText(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		text   string
		wantPt bool
	}{
		{"empty", "", false},
		{"populated", "export default { value_type: 'boolean' }", true},
		{"whitespace", " ", true},
		{"newline-only", "\n", true},
		{"unicode", "// 漢字", true},
		{"trailing-null", "x\x00", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := store.FlagVersion{
				ID:          fixedID(0),
				FlagID:      fixedID(1),
				Version:     3,
				Strategy:    "typescript",
				Source:      json.RawMessage(`{"x":1}`),
				Compiled:    json.RawMessage(`{"y":2}`),
				SourceText:  tc.text,
				PublishedAt: fixedTime(42),
			}
			out := flagVersionToAPI(v)
			if (out.SourceText != nil) != tc.wantPt {
				t.Errorf("SourceText nilness = %v, want non-nil=%v", out.SourceText, tc.wantPt)
			}
			if out.SourceText != nil && *out.SourceText != tc.text {
				t.Errorf("SourceText = %q, want %q", *out.SourceText, tc.text)
			}
			if out.Version != 3 {
				t.Errorf("Version = %d", out.Version)
			}
		})
	}
}

func TestEnvironmentToAPI(t *testing.T) {
	t.Parallel()
	for i := 0; i < 32; i++ {
		i := i
		t.Run(fmt.Sprintf("case-%02d", i), func(t *testing.T) {
			t.Parallel()
			e := store.Environment{
				ID:        fixedID(i),
				ProjectID: fixedID(i + 100),
				Slug:      fmt.Sprintf("env-%d", i),
				Name:      fmt.Sprintf("Env %d", i),
				CreatedAt: fixedTime(i),
			}
			out := environmentToAPI(e)
			if out.Slug != e.Slug || out.Name != e.Name {
				t.Errorf("Slug/Name mismatch")
			}
		})
	}
}

func TestSegmentToAPI(t *testing.T) {
	t.Parallel()
	predicates := []json.RawMessage{
		json.RawMessage(`{"kind":"always"}`),
		json.RawMessage(`{"kind":"eq","attr":"env","value":"prod"}`),
		json.RawMessage(`{"kind":"in","attr":"plan","values":["pro","ent"]}`),
		json.RawMessage(`null`),
		nil,
	}
	for i := 0; i < 32; i++ {
		i := i
		t.Run(fmt.Sprintf("case-%02d", i), func(t *testing.T) {
			t.Parallel()
			s := store.Segment{
				ID:          fixedID(i),
				ProjectID:   fixedID(i + 100),
				Key:         fmt.Sprintf("seg-%d", i),
				Name:        fmt.Sprintf("Seg %d", i),
				Description: "desc",
				Predicate:   predicates[i%len(predicates)],
				CreatedAt:   fixedTime(i),
				UpdatedAt:   fixedTime(i + 1),
			}
			out := segmentToAPI(s)
			if out.Key != s.Key {
				t.Errorf("Key mismatch")
			}
		})
	}
}

func TestSnapshotToAPI_EnvironmentNullable(t *testing.T) {
	t.Parallel()
	envID := fixedID(1)
	cases := []struct {
		name   string
		envID  uuid.NullUUID
		expect bool
	}{
		{"no-env", uuid.NullUUID{Valid: false}, false},
		{"with-env", uuid.NullUUID{Valid: true, UUID: envID}, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := snapshotToAPI(store.Snapshot{
				ID:            fixedID(0),
				ProjectID:     fixedID(2),
				EnvironmentID: tc.envID,
				Version:       7,
				Compiled:      json.RawMessage(`{}`),
				CreatedAt:     fixedTime(0),
			})
			if (out.EnvironmentId != nil) != tc.expect {
				t.Errorf("EnvironmentId nilness = %v, want non-nil=%v", out.EnvironmentId, tc.expect)
			}
			if out.Version != 7 {
				t.Errorf("Version = %d", out.Version)
			}
		})
	}
}

func TestAuditEventToAPI_NullableFields(t *testing.T) {
	t.Parallel()
	projID := fixedID(1)
	flagID := fixedID(2)
	cases := []struct {
		name     string
		ev       store.AuditEvent
		wantProj bool
		wantFlag bool
		wantAct  bool
	}{
		{
			name:     "bare",
			ev:       store.AuditEvent{ID: fixedID(0), Action: "noop", Payload: json.RawMessage(`{}`), CreatedAt: fixedTime(0)},
			wantProj: false, wantFlag: false, wantAct: false,
		},
		{
			name: "with-project",
			ev: store.AuditEvent{
				ID:        fixedID(0),
				ProjectID: uuid.NullUUID{Valid: true, UUID: projID},
				Action:    "project.create",
				Payload:   json.RawMessage(`{"slug":"x"}`),
				CreatedAt: fixedTime(0),
			},
			wantProj: true,
		},
		{
			name: "with-flag",
			ev: store.AuditEvent{
				ID:        fixedID(0),
				FlagID:    uuid.NullUUID{Valid: true, UUID: flagID},
				Action:    "flag.publish",
				Payload:   json.RawMessage(`{}`),
				CreatedAt: fixedTime(0),
			},
			wantFlag: true,
		},
		{
			name: "with-actor",
			ev: store.AuditEvent{
				ID:        fixedID(0),
				Actor:     "alice@example.com",
				Action:    "session",
				Payload:   json.RawMessage(`{}`),
				CreatedAt: fixedTime(0),
			},
			wantAct: true,
		},
		{
			name: "all-fields",
			ev: store.AuditEvent{
				ID:        fixedID(0),
				ProjectID: uuid.NullUUID{Valid: true, UUID: projID},
				FlagID:    uuid.NullUUID{Valid: true, UUID: flagID},
				Action:    "flag.toggle",
				Actor:     "bot",
				Payload:   json.RawMessage(`{"on":true}`),
				CreatedAt: fixedTime(0),
			},
			wantProj: true, wantFlag: true, wantAct: true,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := auditEventToAPI(tc.ev)
			if (out.ProjectId != nil) != tc.wantProj {
				t.Errorf("ProjectId nilness = %v want non-nil=%v", out.ProjectId, tc.wantProj)
			}
			if (out.FlagId != nil) != tc.wantFlag {
				t.Errorf("FlagId nilness = %v want non-nil=%v", out.FlagId, tc.wantFlag)
			}
			if (out.Actor != nil) != tc.wantAct {
				t.Errorf("Actor nilness = %v want non-nil=%v", out.Actor, tc.wantAct)
			}
			if out.Action != tc.ev.Action {
				t.Errorf("Action = %q", out.Action)
			}
		})
	}
}

func TestDecisionToAPI(t *testing.T) {
	t.Parallel()
	reasons := []string{
		eval.ReasonDefault,
		eval.ReasonRuleMatched,
		eval.ReasonRolloutInBucket,
		eval.ReasonRolloutOutOfBucket,
		eval.ReasonTypeMismatch,
		eval.ReasonError,
	}
	for i, r := range reasons {
		i, r := i, r
		t.Run(r, func(t *testing.T) {
			t.Parallel()
			d := eval.Decision{
				Value:   i,
				Reason:  r,
				RuleID:  fmt.Sprintf("rule-%d", i),
				Version: i + 1,
			}
			out := decisionToAPI(d)
			if string(out.Reason) != string(r) {
				t.Errorf("Reason = %q", out.Reason)
			}
			if out.RuleId == nil || *out.RuleId != d.RuleID {
				t.Errorf("RuleId pointer wrong: %v", out.RuleId)
			}
			if out.Version != int32(d.Version) {
				t.Errorf("Version = %d", out.Version)
			}
		})
	}
}

func TestDecisionToAPI_NoRuleID(t *testing.T) {
	t.Parallel()
	d := eval.Decision{Value: true, Reason: eval.ReasonDefault, Version: 1}
	out := decisionToAPI(d)
	if out.RuleId != nil {
		t.Errorf("RuleId = %v, want nil", out.RuleId)
	}
}

func TestRawToAny(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   json.RawMessage
		want any
	}{
		{"nil", nil, nil},
		{"empty", json.RawMessage{}, nil},
		{"bool-true", json.RawMessage(`true`), true},
		{"bool-false", json.RawMessage(`false`), false},
		{"number", json.RawMessage(`42`), float64(42)},
		{"string", json.RawMessage(`"hi"`), "hi"},
		{"null", json.RawMessage(`null`), nil},
		{"empty-obj", json.RawMessage(`{}`), map[string]any{}},
		{"empty-arr", json.RawMessage(`[]`), []any{}},
		{"invalid-json", json.RawMessage(`{bad`), "{bad"},
		{"large-int", json.RawMessage(`9007199254740992`), float64(9007199254740992)},
		{"neg-num", json.RawMessage(`-3.14`), -3.14},
		{"unicode-string", json.RawMessage(`"漢字"`), "漢字"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := rawToAny(tc.in)
			switch want := tc.want.(type) {
			case nil:
				if got != nil {
					t.Errorf("got %v, want nil", got)
				}
			default:
				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(want)
				if string(gotJSON) != string(wantJSON) {
					t.Errorf("got %s, want %s", gotJSON, wantJSON)
				}
			}
		})
	}
}

func TestTraceNodeToAPI_AllKinds(t *testing.T) {
	t.Parallel()
	kinds := []string{"eq", "neq", "in", "gt", "gte", "lt", "lte", "matches", "rollout", "all", "any", "not", "cel", "always"}
	for _, k := range kinds {
		k := k
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			n := eval.TraceNode{
				Kind:      k,
				Attr:      "user.id",
				AttrValue: "u1",
				Expected:  "u1",
				Pattern:   "^u.*",
				Salt:      "salt",
				Percent:   50,
				Bucket:    25,
				Source:    "user.id == 'u1'",
				Result:    true,
			}
			out := traceNodeToAPI(n)
			if out.Kind != k {
				t.Errorf("Kind = %q", out.Kind)
			}
			if out.Attr == nil || *out.Attr != "user.id" {
				t.Errorf("Attr pointer = %v", out.Attr)
			}
			if out.Pattern == nil || *out.Pattern != "^u.*" {
				t.Errorf("Pattern pointer = %v", out.Pattern)
			}
			if out.Percent == nil || *out.Percent != 50 {
				t.Errorf("Percent pointer = %v", out.Percent)
			}
			if out.Bucket == nil || *out.Bucket != 25 {
				t.Errorf("Bucket pointer = %v", out.Bucket)
			}
		})
	}
}

func TestTraceNodeToAPI_ZeroOmits(t *testing.T) {
	t.Parallel()
	n := eval.TraceNode{Kind: "always", Result: true}
	out := traceNodeToAPI(n)
	if out.Attr != nil {
		t.Errorf("Attr should be nil")
	}
	if out.Pattern != nil {
		t.Errorf("Pattern should be nil")
	}
	if out.Percent != nil {
		t.Errorf("Percent should be nil")
	}
	if out.Bucket != nil {
		t.Errorf("Bucket should be nil")
	}
	if out.Children != nil {
		t.Errorf("Children should be nil")
	}
}

func TestTraceNodeToAPI_ExpectedValuesCopiedAndOrdered(t *testing.T) {
	t.Parallel()
	n := eval.TraceNode{
		Kind:           "in",
		Attr:           "plan",
		ExpectedValues: []any{"pro", "ent", "team"},
	}
	out := traceNodeToAPI(n)
	if out.ExpectedValues == nil {
		t.Fatal("ExpectedValues should not be nil")
	}
	got := *out.ExpectedValues
	want := []any{"pro", "ent", "team"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("[%d] = %v want %v", i, got[i], v)
		}
	}
	// Mutating the source must not affect the returned slice.
	n.ExpectedValues[0] = "changed"
	if got[0] != "pro" {
		t.Errorf("returned slice aliased the source: got[0]=%v", got[0])
	}
}

func TestTraceNodeToAPI_NestedChildren(t *testing.T) {
	t.Parallel()
	n := eval.TraceNode{
		Kind: "all",
		Children: []eval.TraceNode{
			{Kind: "eq", Attr: "env", Result: true},
			{Kind: "any", Children: []eval.TraceNode{
				{Kind: "eq", Attr: "plan", Result: false},
				{Kind: "rollout", Attr: "user.id", Percent: 25, Bucket: 12, Result: true},
			}},
		},
		Result: true,
	}
	out := traceNodeToAPI(n)
	if out.Children == nil {
		t.Fatal("Children nil")
	}
	if len(*out.Children) != 2 {
		t.Fatalf("len Children = %d", len(*out.Children))
	}
	inner := (*out.Children)[1].Children
	if inner == nil || len(*inner) != 2 {
		t.Fatalf("inner children = %v", inner)
	}
}

func TestTraceToAPI(t *testing.T) {
	t.Parallel()
	tr := eval.Trace{
		EvaluatedRules: []eval.TraceRule{
			{RuleID: "r1", Matched: true, Predicate: eval.TraceNode{Kind: "always", Result: true}},
			{RuleID: "r2", Matched: false, Predicate: eval.TraceNode{Kind: "eq", Attr: "env", Result: false}, Error: "boom"},
		},
		DefaultUsed:   false,
		MatchedRuleID: "r1",
	}
	out := traceToAPI(tr)
	if len(out.EvaluatedRules) != 2 {
		t.Fatalf("rules len = %d", len(out.EvaluatedRules))
	}
	if out.EvaluatedRules[0].RuleId != "r1" {
		t.Errorf("rule0 id = %q", out.EvaluatedRules[0].RuleId)
	}
	if out.EvaluatedRules[1].Error == nil || *out.EvaluatedRules[1].Error != "boom" {
		t.Errorf("rule1 error = %v", out.EvaluatedRules[1].Error)
	}
	if out.MatchedRuleId == nil || *out.MatchedRuleId != "r1" {
		t.Errorf("MatchedRuleId = %v", out.MatchedRuleId)
	}
}

func TestTraceToAPI_EmptyRulesNoMatch(t *testing.T) {
	t.Parallel()
	tr := eval.Trace{EvaluatedRules: nil, DefaultUsed: true}
	out := traceToAPI(tr)
	if len(out.EvaluatedRules) != 0 {
		t.Errorf("rules should be empty, got %d", len(out.EvaluatedRules))
	}
	if out.MatchedRuleId != nil {
		t.Errorf("MatchedRuleId should be nil")
	}
	if !out.DefaultUsed {
		t.Errorf("DefaultUsed = false, want true")
	}
}
