package handlers

import (
	"encoding/json"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/depot/falseflag/internal/eval"
	"github.com/depot/falseflag/internal/gen/openapi"
	"github.com/depot/falseflag/internal/store"
)

// Converters from internal store domain types to the OpenAPI-generated
// wire types. One place for the translation keeps handler bodies
// focused on orchestration.

func projectToAPI(p store.Project) openapi.Project {
	return openapi.Project{
		Id:             openapi_types.UUID(p.ID),
		Slug:           p.Slug,
		DisplayName:    p.DisplayName,
		ConfigStrategy: openapi.Strategy(p.ConfigStrategy),
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func flagToAPI(f store.Flag) openapi.Flag {
	return openapi.Flag{
		Id:           openapi_types.UUID(f.ID),
		ProjectId:    openapi_types.UUID(f.ProjectID),
		Key:          f.Key,
		Name:         f.Name,
		Description:  f.Description,
		ValueType:    openapi.ValueType(f.ValueType),
		DefaultValue: rawToAny(f.DefaultValue),
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
}

func flagVersionToAPI(v store.FlagVersion) openapi.FlagVersion {
	out := openapi.FlagVersion{
		Id:          openapi_types.UUID(v.ID),
		FlagId:      openapi_types.UUID(v.FlagID),
		Version:     int32(v.Version),
		Strategy:    openapi.Strategy(v.Strategy),
		Source:      rawToAny(v.Source),
		Compiled:    rawToAny(v.Compiled),
		PublishedAt: v.PublishedAt,
	}
	if v.SourceText != "" {
		s := v.SourceText
		out.SourceText = &s
	}
	return out
}

func environmentToAPI(e store.Environment) openapi.Environment {
	return openapi.Environment{
		Id:        openapi_types.UUID(e.ID),
		ProjectId: openapi_types.UUID(e.ProjectID),
		Slug:      e.Slug,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
	}
}

func segmentToAPI(s store.Segment) openapi.Segment {
	return openapi.Segment{
		Id:          openapi_types.UUID(s.ID),
		ProjectId:   openapi_types.UUID(s.ProjectID),
		Key:         s.Key,
		Name:        s.Name,
		Description: s.Description,
		Predicate:   rawToAny(s.Predicate),
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}

func snapshotToAPI(s store.Snapshot) openapi.Snapshot {
	out := openapi.Snapshot{
		Id:        openapi_types.UUID(s.ID),
		ProjectId: openapi_types.UUID(s.ProjectID),
		Version:   int32(s.Version),
		Compiled:  rawToAny(s.Compiled),
		CreatedAt: s.CreatedAt,
	}
	if s.EnvironmentID.Valid {
		envID := openapi_types.UUID(s.EnvironmentID.UUID)
		out.EnvironmentId = &envID
	}
	return out
}

func auditEventToAPI(a store.AuditEvent) openapi.AuditEvent {
	out := openapi.AuditEvent{
		Id:        openapi_types.UUID(a.ID),
		Action:    a.Action,
		Payload:   rawToAny(a.Payload),
		CreatedAt: a.CreatedAt,
	}
	if a.ProjectID.Valid {
		id := openapi_types.UUID(a.ProjectID.UUID)
		out.ProjectId = &id
	}
	if a.FlagID.Valid {
		id := openapi_types.UUID(a.FlagID.UUID)
		out.FlagId = &id
	}
	if a.Actor != "" {
		out.Actor = &a.Actor
	}
	return out
}

func decisionToAPI(d eval.Decision) openapi.Decision {
	out := openapi.Decision{
		Value:   d.Value,
		Reason:  openapi.DecisionReason(d.Reason),
		Version: int32(d.Version),
	}
	if d.RuleID != "" {
		out.RuleId = &d.RuleID
	}
	return out
}

func traceToAPI(t eval.Trace) openapi.EvaluationTrace {
	rules := make([]openapi.TraceRule, 0, len(t.EvaluatedRules))
	for _, r := range t.EvaluatedRules {
		tr := openapi.TraceRule{
			RuleId:    r.RuleID,
			Matched:   r.Matched,
			Predicate: traceNodeToAPI(r.Predicate),
		}
		if r.Error != "" {
			tr.Error = &r.Error
		}
		rules = append(rules, tr)
	}
	out := openapi.EvaluationTrace{
		EvaluatedRules: rules,
		DefaultUsed:    t.DefaultUsed,
	}
	if t.MatchedRuleID != "" {
		out.MatchedRuleId = &t.MatchedRuleID
	}
	return out
}

func traceNodeToAPI(n eval.TraceNode) openapi.TraceNode {
	out := openapi.TraceNode{
		Kind:   n.Kind,
		Result: n.Result,
	}
	if n.Attr != "" {
		out.Attr = &n.Attr
	}
	if n.AttrValue != nil {
		out.AttrValue = n.AttrValue
	}
	if n.Expected != nil {
		out.Expected = n.Expected
	}
	if len(n.ExpectedValues) > 0 {
		copyVals := append([]any(nil), n.ExpectedValues...)
		out.ExpectedValues = &copyVals
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
		children := make([]openapi.TraceNode, 0, len(n.Children))
		for _, c := range n.Children {
			children = append(children, traceNodeToAPI(c))
		}
		out.Children = &children
	}
	return out
}

func rawToAny(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	return v
}
