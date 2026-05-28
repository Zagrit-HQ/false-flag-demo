package rpc

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/depot/falseflag/internal/eval"
	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/store"
)

// Translation helpers from internal store domain types to the
// protobuf-generated wire types. The REST handlers have an equivalent
// translation layer in internal/server/handlers/convert.go; the two
// must agree on the value shapes (modulo casing differences).

var strategyToProto = map[string]pb.Strategy{
	"json":       pb.Strategy_STRATEGY_JSON,
	"cel":        pb.Strategy_STRATEGY_CEL,
	"typescript": pb.Strategy_STRATEGY_TYPESCRIPT,
}

func strategyFromProto(s pb.Strategy) string {
	switch s {
	case pb.Strategy_STRATEGY_JSON:
		return "json"
	case pb.Strategy_STRATEGY_CEL:
		return "cel"
	case pb.Strategy_STRATEGY_TYPESCRIPT:
		return "typescript"
	}
	return ""
}

func protoStrategy(s string) pb.Strategy { return strategyToProto[s] }

var valueTypeToProto = map[string]pb.ValueType{
	"boolean": pb.ValueType_VALUE_TYPE_BOOLEAN,
	"string":  pb.ValueType_VALUE_TYPE_STRING,
	"number":  pb.ValueType_VALUE_TYPE_NUMBER,
	"object":  pb.ValueType_VALUE_TYPE_OBJECT,
}

func valueTypeFromProto(v pb.ValueType) string {
	switch v {
	case pb.ValueType_VALUE_TYPE_BOOLEAN:
		return "boolean"
	case pb.ValueType_VALUE_TYPE_STRING:
		return "string"
	case pb.ValueType_VALUE_TYPE_NUMBER:
		return "number"
	case pb.ValueType_VALUE_TYPE_OBJECT:
		return "object"
	}
	return ""
}

func protoValueType(v string) pb.ValueType { return valueTypeToProto[v] }

func decisionReasonToProto(r string) pb.DecisionReason {
	switch r {
	case eval.ReasonDefault:
		return pb.DecisionReason_DECISION_REASON_DEFAULT
	case eval.ReasonRuleMatched:
		return pb.DecisionReason_DECISION_REASON_RULE_MATCHED
	case eval.ReasonRolloutInBucket:
		return pb.DecisionReason_DECISION_REASON_ROLLOUT_IN_BUCKET
	case eval.ReasonRolloutOutOfBucket:
		return pb.DecisionReason_DECISION_REASON_ROLLOUT_OUT_OF_BUCKET
	case eval.ReasonTypeMismatch:
		return pb.DecisionReason_DECISION_REASON_TYPE_MISMATCH
	case eval.ReasonError:
		return pb.DecisionReason_DECISION_REASON_ERROR
	}
	return pb.DecisionReason_DECISION_REASON_UNSPECIFIED
}

func projectToProto(p store.Project) *pb.Project {
	return &pb.Project{
		Id:             p.ID.String(),
		Slug:           p.Slug,
		DisplayName:    p.DisplayName,
		ConfigStrategy: protoStrategy(p.ConfigStrategy),
		CreatedAt:      timestamppb.New(p.CreatedAt),
		UpdatedAt:      timestamppb.New(p.UpdatedAt),
	}
}

func environmentToProto(e store.Environment) *pb.Environment {
	return &pb.Environment{
		Id:        e.ID.String(),
		ProjectId: e.ProjectID.String(),
		Slug:      e.Slug,
		Name:      e.Name,
		CreatedAt: timestamppb.New(e.CreatedAt),
	}
}

func flagToProto(f store.Flag) *pb.Flag {
	defVal, _ := jsonRawToValue(f.DefaultValue)
	return &pb.Flag{
		Id:           f.ID.String(),
		ProjectId:    f.ProjectID.String(),
		Key:          f.Key,
		Name:         f.Name,
		Description:  f.Description,
		ValueType:    protoValueType(f.ValueType),
		DefaultValue: defVal,
		CreatedAt:    timestamppb.New(f.CreatedAt),
		UpdatedAt:    timestamppb.New(f.UpdatedAt),
	}
}

func flagVersionToProto(v store.FlagVersion) *pb.FlagVersion {
	src, _ := jsonRawToStruct(v.Source)
	comp, _ := jsonRawToStruct(v.Compiled)
	return &pb.FlagVersion{
		Id:          v.ID.String(),
		FlagId:      v.FlagID.String(),
		Version:     int32(v.Version),
		Strategy:    protoStrategy(v.Strategy),
		Source:      src,
		Compiled:    comp,
		SourceText:  v.SourceText,
		PublishedAt: timestamppb.New(v.PublishedAt),
	}
}

func segmentToProto(s store.Segment) *pb.Segment {
	pred, _ := jsonRawToStruct(s.Predicate)
	return &pb.Segment{
		Id:          s.ID.String(),
		ProjectId:   s.ProjectID.String(),
		Key:         s.Key,
		Name:        s.Name,
		Description: s.Description,
		Predicate:   pred,
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
	}
}

func snapshotToProto(s store.Snapshot) *pb.Snapshot {
	comp, _ := jsonRawToStruct(s.Compiled)
	out := &pb.Snapshot{
		Id:        s.ID.String(),
		ProjectId: s.ProjectID.String(),
		Version:   int32(s.Version),
		Compiled:  comp,
		CreatedAt: timestamppb.New(s.CreatedAt),
	}
	if s.EnvironmentID.Valid {
		envID := s.EnvironmentID.UUID.String()
		out.EnvironmentId = &envID
	}
	return out
}

func auditEventToProto(a store.AuditEvent) *pb.AuditEvent {
	payload, _ := jsonRawToStruct(a.Payload)
	out := &pb.AuditEvent{
		Id:        a.ID.String(),
		Action:    a.Action,
		Payload:   payload,
		CreatedAt: timestamppb.New(a.CreatedAt),
	}
	if a.ProjectID.Valid {
		s := a.ProjectID.UUID.String()
		out.ProjectId = &s
	}
	if a.FlagID.Valid {
		s := a.FlagID.UUID.String()
		out.FlagId = &s
	}
	if a.Actor != "" {
		out.Actor = &a.Actor
	}
	return out
}

func decisionToProto(d eval.Decision) *pb.Decision {
	val, _ := goValueToProto(d.Value)
	out := &pb.Decision{
		Value:   val,
		Reason:  decisionReasonToProto(d.Reason),
		Version: int32(d.Version),
	}
	if d.RuleID != "" {
		out.RuleId = &d.RuleID
	}
	return out
}

func traceToProto(t eval.Trace) *pb.EvaluationTrace {
	rules := make([]*pb.TraceRule, 0, len(t.EvaluatedRules))
	for _, r := range t.EvaluatedRules {
		tr := &pb.TraceRule{
			RuleId:    r.RuleID,
			Matched:   r.Matched,
			Predicate: traceNodeToProto(r.Predicate),
		}
		if r.Error != "" {
			tr.Error = &r.Error
		}
		rules = append(rules, tr)
	}
	out := &pb.EvaluationTrace{
		EvaluatedRules: rules,
		DefaultUsed:    t.DefaultUsed,
	}
	if t.MatchedRuleID != "" {
		out.MatchedRuleId = &t.MatchedRuleID
	}
	return out
}

func traceNodeToProto(n eval.TraceNode) *pb.TraceNode {
	out := &pb.TraceNode{Kind: n.Kind, Result: n.Result}
	if n.Attr != "" {
		out.Attr = &n.Attr
	}
	if n.AttrValue != nil {
		if v, err := goValueToProto(n.AttrValue); err == nil {
			out.AttrValue = v
		}
	}
	if n.Expected != nil {
		if v, err := goValueToProto(n.Expected); err == nil {
			out.Expected = v
		}
	}
	if len(n.ExpectedValues) > 0 {
		vals := make([]*structpb.Value, 0, len(n.ExpectedValues))
		for _, v := range n.ExpectedValues {
			pv, err := goValueToProto(v)
			if err != nil {
				continue
			}
			vals = append(vals, pv)
		}
		out.ExpectedValues = vals
	}
	if n.Pattern != "" {
		out.Pattern = &n.Pattern
	}
	if n.Salt != "" {
		out.Salt = &n.Salt
	}
	if n.Percent != 0 {
		p := int32(n.Percent)
		out.Percent = &p
	}
	if n.Bucket != 0 {
		b := int32(n.Bucket)
		out.Bucket = &b
	}
	if n.Source != "" {
		out.Source = &n.Source
	}
	if len(n.Children) > 0 {
		children := make([]*pb.TraceNode, 0, len(n.Children))
		for _, c := range n.Children {
			children = append(children, traceNodeToProto(c))
		}
		out.Children = children
	}
	return out
}

// jsonRawToStruct decodes raw JSON object bytes into a structpb.Struct.
// Empty input returns nil with no error.
func jsonRawToStruct(raw json.RawMessage) (*structpb.Struct, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return structpb.NewStruct(m)
}

// jsonRawToValue decodes any JSON value bytes into a structpb.Value.
func jsonRawToValue(raw json.RawMessage) (*structpb.Value, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	return goValueToProto(v)
}

func goValueToProto(v any) (*structpb.Value, error) {
	return structpb.NewValue(v)
}

// structToJSONRaw marshals a structpb.Struct back to json.RawMessage.
// Used by handlers that store struct fields as jsonb.
func structToJSONRaw(s *structpb.Struct) (json.RawMessage, error) {
	if s == nil {
		return json.RawMessage(`{}`), nil
	}
	return json.Marshal(s.AsMap())
}

// valueToAny unwraps a structpb.Value into a Go any (string/bool/number/map/slice).
func valueToAny(v *structpb.Value) any {
	if v == nil {
		return nil
	}
	return v.AsInterface()
}

// _ silences "imported and not used" if a helper above becomes unused.
var _ = fmt.Errorf
