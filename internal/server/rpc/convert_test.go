package rpc

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/depot/falseflag/internal/eval"
	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/store"
)

func id(i int) uuid.UUID {
	return uuid.NewSHA1(uuid.NameSpaceOID, fmt.Appendf(nil, "rpc-convert-%d", i))
}

func ts(i int) time.Time {
	return time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Minute)
}

func TestStrategyRoundTrip(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"json", "cel", "typescript"} {
		s := s
		t.Run(s, func(t *testing.T) {
			t.Parallel()
			got := strategyFromProto(protoStrategy(s))
			if got != s {
				t.Errorf("round-trip lost %q -> %q", s, got)
			}
		})
	}
}

func TestStrategyFromProto_Unspecified(t *testing.T) {
	t.Parallel()
	if got := strategyFromProto(pb.Strategy_STRATEGY_UNSPECIFIED); got != "" {
		t.Errorf("got %q want empty", got)
	}
}

func TestProtoStrategy_UnknownReturnsUnspecified(t *testing.T) {
	t.Parallel()
	if got := protoStrategy("yaml"); got != pb.Strategy_STRATEGY_UNSPECIFIED {
		t.Errorf("got %v", got)
	}
}

func TestValueTypeRoundTrip(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"boolean", "string", "number", "object"} {
		s := s
		t.Run(s, func(t *testing.T) {
			t.Parallel()
			got := valueTypeFromProto(protoValueType(s))
			if got != s {
				t.Errorf("round-trip lost %q -> %q", s, got)
			}
		})
	}
}

func TestValueTypeFromProto_Unspecified(t *testing.T) {
	t.Parallel()
	if got := valueTypeFromProto(pb.ValueType_VALUE_TYPE_UNSPECIFIED); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestProtoValueType_Unknown(t *testing.T) {
	t.Parallel()
	if got := protoValueType("date"); got != pb.ValueType_VALUE_TYPE_UNSPECIFIED {
		t.Errorf("got %v", got)
	}
}

func TestDecisionReasonToProto(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want pb.DecisionReason
	}{
		{eval.ReasonDefault, pb.DecisionReason_DECISION_REASON_DEFAULT},
		{eval.ReasonRuleMatched, pb.DecisionReason_DECISION_REASON_RULE_MATCHED},
		{eval.ReasonRolloutInBucket, pb.DecisionReason_DECISION_REASON_ROLLOUT_IN_BUCKET},
		{eval.ReasonRolloutOutOfBucket, pb.DecisionReason_DECISION_REASON_ROLLOUT_OUT_OF_BUCKET},
		{eval.ReasonTypeMismatch, pb.DecisionReason_DECISION_REASON_TYPE_MISMATCH},
		{eval.ReasonError, pb.DecisionReason_DECISION_REASON_ERROR},
		{"unknown", pb.DecisionReason_DECISION_REASON_UNSPECIFIED},
		{"", pb.DecisionReason_DECISION_REASON_UNSPECIFIED},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := decisionReasonToProto(tc.in); got != tc.want {
				t.Errorf("decisionReasonToProto(%q) = %v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestProjectToProto(t *testing.T) {
	t.Parallel()
	for i := 0; i < 32; i++ {
		i := i
		t.Run(fmt.Sprintf("case-%02d", i), func(t *testing.T) {
			t.Parallel()
			p := store.Project{
				ID:             id(i),
				Slug:           fmt.Sprintf("p-%d", i),
				DisplayName:    fmt.Sprintf("Project %d", i),
				ConfigStrategy: []string{"json", "cel", "typescript"}[i%3],
				CreatedAt:      ts(i),
				UpdatedAt:      ts(i + 1),
			}
			out := projectToProto(p)
			if out.Id != p.ID.String() {
				t.Errorf("Id = %q", out.Id)
			}
			if out.Slug != p.Slug {
				t.Errorf("Slug = %q", out.Slug)
			}
			if !out.CreatedAt.AsTime().Equal(p.CreatedAt) {
				t.Errorf("CreatedAt mismatch")
			}
		})
	}
}

func TestEnvironmentToProto(t *testing.T) {
	t.Parallel()
	for i := 0; i < 16; i++ {
		i := i
		t.Run(fmt.Sprintf("env-%d", i), func(t *testing.T) {
			t.Parallel()
			e := store.Environment{
				ID:        id(i),
				ProjectID: id(i + 1000),
				Slug:      fmt.Sprintf("env-%d", i),
				Name:      fmt.Sprintf("Env %d", i),
				CreatedAt: ts(i),
			}
			out := environmentToProto(e)
			if out.Slug != e.Slug || out.Name != e.Name {
				t.Errorf("Slug/Name mismatch")
			}
			if out.ProjectId != e.ProjectID.String() {
				t.Errorf("ProjectId = %q", out.ProjectId)
			}
		})
	}
}

func TestFlagToProto_DefaultValues(t *testing.T) {
	t.Parallel()
	cases := []json.RawMessage{
		json.RawMessage(`true`),
		json.RawMessage(`false`),
		json.RawMessage(`"hi"`),
		json.RawMessage(`42`),
		json.RawMessage(`{"k":"v"}`),
		nil,
		json.RawMessage(`null`),
	}
	for i, raw := range cases {
		i, raw := i, raw
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			t.Parallel()
			f := store.Flag{
				ID:           id(i),
				ProjectID:    id(i + 100),
				Key:          fmt.Sprintf("f-%d", i),
				Name:         "F",
				Description:  "d",
				ValueType:    "boolean",
				DefaultValue: raw,
				CreatedAt:    ts(0),
				UpdatedAt:    ts(0),
			}
			out := flagToProto(f)
			if out.Key != f.Key {
				t.Errorf("Key = %q", out.Key)
			}
			if string(out.ValueType) == "" {
				t.Errorf("ValueType not set")
			}
		})
	}
}

func TestFlagVersionToProto_SourceText(t *testing.T) {
	t.Parallel()
	v := store.FlagVersion{
		ID:          id(0),
		FlagID:      id(1),
		Version:     5,
		Strategy:    "typescript",
		Source:      json.RawMessage(`{"k":"v"}`),
		Compiled:    json.RawMessage(`{"c":1}`),
		SourceText:  "export default {}",
		PublishedAt: ts(0),
	}
	out := flagVersionToProto(v)
	if out.Version != 5 || out.SourceText != v.SourceText {
		t.Errorf("got %+v", out)
	}
}

func TestSegmentToProto(t *testing.T) {
	t.Parallel()
	s := store.Segment{
		ID:          id(0),
		ProjectID:   id(1),
		Key:         "seg",
		Name:        "Seg",
		Description: "d",
		Predicate:   json.RawMessage(`{"kind":"always"}`),
		CreatedAt:   ts(0),
		UpdatedAt:   ts(1),
	}
	out := segmentToProto(s)
	if out.Key != "seg" {
		t.Errorf("Key = %q", out.Key)
	}
	if out.Predicate == nil {
		t.Errorf("Predicate nil")
	}
}

func TestSnapshotToProto_EnvironmentNullable(t *testing.T) {
	t.Parallel()
	envID := id(99)
	cases := []struct {
		name string
		env  uuid.NullUUID
		want bool
	}{
		{"absent", uuid.NullUUID{}, false},
		{"present", uuid.NullUUID{Valid: true, UUID: envID}, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := snapshotToProto(store.Snapshot{
				ID:            id(0),
				ProjectID:     id(1),
				EnvironmentID: tc.env,
				Version:       3,
				Compiled:      json.RawMessage(`{}`),
				CreatedAt:     ts(0),
			})
			if (out.EnvironmentId != nil) != tc.want {
				t.Errorf("EnvironmentId nilness = %v want non-nil=%v", out.EnvironmentId, tc.want)
			}
		})
	}
}

func TestAuditEventToProto(t *testing.T) {
	t.Parallel()
	projID := id(2)
	flagID := id(3)
	cases := []struct {
		name string
		ev   store.AuditEvent
	}{
		{"bare", store.AuditEvent{ID: id(0), Action: "noop", Payload: json.RawMessage(`{}`), CreatedAt: ts(0)}},
		{"project", store.AuditEvent{ID: id(0), ProjectID: uuid.NullUUID{Valid: true, UUID: projID}, Action: "p", Payload: json.RawMessage(`{}`), CreatedAt: ts(0)}},
		{"flag", store.AuditEvent{ID: id(0), FlagID: uuid.NullUUID{Valid: true, UUID: flagID}, Action: "f", Payload: json.RawMessage(`{}`), CreatedAt: ts(0)}},
		{"actor", store.AuditEvent{ID: id(0), Actor: "alice", Action: "a", Payload: json.RawMessage(`{}`), CreatedAt: ts(0)}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := auditEventToProto(tc.ev)
			if out.Action != tc.ev.Action {
				t.Errorf("Action mismatch")
			}
			if (out.ProjectId != nil) != tc.ev.ProjectID.Valid {
				t.Errorf("ProjectId nilness wrong")
			}
			if (out.FlagId != nil) != tc.ev.FlagID.Valid {
				t.Errorf("FlagId nilness wrong")
			}
			if (out.Actor != nil) != (tc.ev.Actor != "") {
				t.Errorf("Actor nilness wrong")
			}
		})
	}
}

func TestDecisionToProto(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		dec  eval.Decision
	}{
		{"with-rule", eval.Decision{Value: true, Reason: eval.ReasonRuleMatched, RuleID: "r1", Version: 1}},
		{"default", eval.Decision{Value: false, Reason: eval.ReasonDefault, Version: 2}},
		{"string", eval.Decision{Value: "alpha", Reason: eval.ReasonRolloutInBucket, RuleID: "r2", Version: 3}},
		{"number", eval.Decision{Value: 3.14, Reason: eval.ReasonRuleMatched, RuleID: "r3", Version: 4}},
		{"obj", eval.Decision{Value: map[string]any{"k": "v"}, Reason: eval.ReasonRuleMatched, RuleID: "r4", Version: 5}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := decisionToProto(tc.dec)
			if out.Version != int32(tc.dec.Version) {
				t.Errorf("Version = %d", out.Version)
			}
			if (out.RuleId != nil) != (tc.dec.RuleID != "") {
				t.Errorf("RuleId nilness wrong")
			}
			if out.Value == nil {
				t.Errorf("Value nil")
			}
		})
	}
}

func TestTraceToProto_RoundFields(t *testing.T) {
	t.Parallel()
	tr := eval.Trace{
		EvaluatedRules: []eval.TraceRule{
			{RuleID: "r1", Matched: true, Predicate: eval.TraceNode{Kind: "always", Result: true}},
			{RuleID: "r2", Matched: false, Predicate: eval.TraceNode{Kind: "eq", Attr: "env"}, Error: "boom"},
		},
		DefaultUsed:   false,
		MatchedRuleID: "r1",
	}
	out := traceToProto(tr)
	if len(out.EvaluatedRules) != 2 {
		t.Fatalf("len = %d", len(out.EvaluatedRules))
	}
	if out.EvaluatedRules[1].Error == nil || *out.EvaluatedRules[1].Error != "boom" {
		t.Errorf("Error pointer = %v", out.EvaluatedRules[1].Error)
	}
	if out.MatchedRuleId == nil || *out.MatchedRuleId != "r1" {
		t.Errorf("MatchedRuleId = %v", out.MatchedRuleId)
	}
}

func TestTraceNodeToProto_AllOptionalFields(t *testing.T) {
	t.Parallel()
	n := eval.TraceNode{
		Kind: "rollout", Attr: "user.id", AttrValue: "u1",
		Expected: "u1", ExpectedValues: []any{"a", "b"},
		Pattern: "^x", Salt: "salt", Percent: 50, Bucket: 25,
		Source: "user.id == 'u1'", Result: true,
	}
	out := traceNodeToProto(n)
	if out.Kind != "rollout" {
		t.Errorf("Kind = %q", out.Kind)
	}
	if out.Attr == nil || *out.Attr != "user.id" {
		t.Errorf("Attr = %v", out.Attr)
	}
	if out.Percent == nil || *out.Percent != 50 {
		t.Errorf("Percent = %v", out.Percent)
	}
	if out.Bucket == nil || *out.Bucket != 25 {
		t.Errorf("Bucket = %v", out.Bucket)
	}
	if len(out.ExpectedValues) != 2 {
		t.Errorf("ExpectedValues len = %d", len(out.ExpectedValues))
	}
}

func TestTraceNodeToProto_ZeroValues(t *testing.T) {
	t.Parallel()
	out := traceNodeToProto(eval.TraceNode{Kind: "always", Result: true})
	if out.Attr != nil || out.Pattern != nil || out.Percent != nil || out.Bucket != nil {
		t.Errorf("zero values should not produce pointer fields: %+v", out)
	}
}

func TestTraceNodeToProto_NestedChildren(t *testing.T) {
	t.Parallel()
	n := eval.TraceNode{
		Kind: "all",
		Children: []eval.TraceNode{
			{Kind: "eq"},
			{Kind: "any", Children: []eval.TraceNode{{Kind: "eq"}, {Kind: "eq"}}},
		},
	}
	out := traceNodeToProto(n)
	if len(out.Children) != 2 {
		t.Fatalf("len = %d", len(out.Children))
	}
	if len(out.Children[1].Children) != 2 {
		t.Errorf("nested len = %d", len(out.Children[1].Children))
	}
}

func TestJSONRawToStruct(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		in      json.RawMessage
		wantNil bool
		wantErr bool
	}{
		{"empty", nil, true, false},
		{"null", json.RawMessage(`null`), true, false},
		{"obj", json.RawMessage(`{"k":"v"}`), false, false},
		{"array", json.RawMessage(`[1,2]`), true, true},
		{"malformed", json.RawMessage(`{bad`), true, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := jsonRawToStruct(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v wantErr=%v", err, tc.wantErr)
			}
			if (got == nil) != tc.wantNil {
				t.Errorf("got nilness = %v wantNil=%v", got == nil, tc.wantNil)
			}
		})
	}
}

func TestJSONRawToValue(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		in      json.RawMessage
		wantNil bool
		wantErr bool
	}{
		{"empty", nil, true, false},
		{"null", json.RawMessage(`null`), true, false},
		{"bool", json.RawMessage(`true`), false, false},
		{"num", json.RawMessage(`42`), false, false},
		{"string", json.RawMessage(`"hi"`), false, false},
		{"obj", json.RawMessage(`{"k":1}`), false, false},
		{"array", json.RawMessage(`[1,2,3]`), false, false},
		{"malformed", json.RawMessage(`{`), true, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := jsonRawToValue(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v wantErr=%v", err, tc.wantErr)
			}
			if (got == nil) != tc.wantNil {
				t.Errorf("got nilness = %v wantNil=%v (val=%v)", got == nil, tc.wantNil, got)
			}
		})
	}
}

func TestStructToJSONRaw(t *testing.T) {
	t.Parallel()
	t.Run("nil-returns-empty-object", func(t *testing.T) {
		t.Parallel()
		got, err := structToJSONRaw(nil)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if string(got) != "{}" {
			t.Errorf("got %q want {}", got)
		}
	})
	t.Run("populated", func(t *testing.T) {
		t.Parallel()
		s, _ := structpb.NewStruct(map[string]any{"k": "v"})
		got, err := structToJSONRaw(s)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(got, &m); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if m["k"] != "v" {
			t.Errorf("m[k] = %v", m["k"])
		}
	})
}

func TestValueToAny(t *testing.T) {
	t.Parallel()
	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		if valueToAny(nil) != nil {
			t.Errorf("not nil")
		}
	})
	t.Run("string", func(t *testing.T) {
		t.Parallel()
		v, _ := structpb.NewValue("hi")
		if valueToAny(v) != "hi" {
			t.Errorf("got %v", valueToAny(v))
		}
	})
	t.Run("number", func(t *testing.T) {
		t.Parallel()
		v, _ := structpb.NewValue(3.14)
		if valueToAny(v) != 3.14 {
			t.Errorf("got %v", valueToAny(v))
		}
	})
	t.Run("bool", func(t *testing.T) {
		t.Parallel()
		v, _ := structpb.NewValue(true)
		if valueToAny(v) != true {
			t.Errorf("got %v", valueToAny(v))
		}
	})
}

func TestGoValueToProto_Errors(t *testing.T) {
	t.Parallel()
	// unsupported types come back as an error from structpb.
	if _, err := goValueToProto(make(chan int)); err == nil {
		t.Errorf("expected error for chan")
	}
	if _, err := goValueToProto("ok"); err != nil {
		t.Errorf("ok value should not error: %v", err)
	}
}
