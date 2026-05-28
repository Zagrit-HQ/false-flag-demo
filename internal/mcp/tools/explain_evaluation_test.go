package tools

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

// fakeEvaluation captures the EvaluateWithTrace call for assertions.
type fakeEvaluation struct {
	falseflagv1connect.EvaluationServiceClient
	LastReq *pb.EvaluateWithTraceRequest
	Resp    *pb.EvaluateWithTraceResponse
	Err     error
}

func (f *fakeEvaluation) EvaluateWithTrace(_ context.Context, req *connect.Request[pb.EvaluateWithTraceRequest]) (*connect.Response[pb.EvaluateWithTraceResponse], error) {
	f.LastReq = req.Msg
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Resp == nil {
		return connect.NewResponse(&pb.EvaluateWithTraceResponse{}), nil
	}
	return connect.NewResponse(f.Resp), nil
}

func TestExplainEvaluation_NilClient(t *testing.T) {
	t.Parallel()
	h := explainEvaluation(nil)
	res, _, err := h(context.Background(), nil, ExplainEvaluationInput{ProjectSlug: "acme", FlagKey: "x"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !res.IsError || !strings.Contains(firstText(t, res), "evaluation client unavailable") {
		t.Errorf("expected client unavailable error, got %+v / %s", res, firstText(t, res))
	}
}

func TestExplainEvaluation_PassesContextStruct(t *testing.T) {
	t.Parallel()
	matched := "pro-only"
	fake := &fakeEvaluation{Resp: &pb.EvaluateWithTraceResponse{
		Decision: &pb.Decision{Reason: pb.DecisionReason_DECISION_REASON_RULE_MATCHED, RuleId: &matched, Version: 1},
		Trace:    &pb.EvaluationTrace{MatchedRuleId: &matched},
	}}
	h := explainEvaluation(&clientapi.Client{Evaluation: fake})

	res, _, err := h(context.Background(), nil, ExplainEvaluationInput{
		ProjectSlug: "acme-web",
		FlagKey:     "beta-checkout",
		Context:     map[string]any{"user": map[string]any{"plan": "pro"}},
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", firstText(t, res))
	}
	if fake.LastReq.GetProjectSlug() != "acme-web" || fake.LastReq.GetKey() != "beta-checkout" {
		t.Errorf("request fields not forwarded: %+v", fake.LastReq)
	}
	if fake.LastReq.GetContext() == nil {
		t.Fatal("expected context struct to be forwarded")
	}
	if !strings.Contains(firstText(t, res), "pro-only") {
		t.Errorf("expected matched rule id in body, got %s", firstText(t, res))
	}
}

func TestExplainEvaluation_NotFound(t *testing.T) {
	t.Parallel()
	fake := &fakeEvaluation{Err: notFoundErr("flag not found")}
	h := explainEvaluation(&clientapi.Client{Evaluation: fake})
	res, _, err := h(context.Background(), nil, ExplainEvaluationInput{ProjectSlug: "x", FlagKey: "y"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError=true")
	}
	if !strings.Contains(firstText(t, res), "not found") {
		t.Errorf("expected 'not found' in body, got %s", firstText(t, res))
	}
}

func TestExplainEvaluation_NoContextOK(t *testing.T) {
	t.Parallel()
	fake := &fakeEvaluation{Resp: &pb.EvaluateWithTraceResponse{
		Decision: &pb.Decision{Reason: pb.DecisionReason_DECISION_REASON_DEFAULT},
	}}
	h := explainEvaluation(&clientapi.Client{Evaluation: fake})
	res, _, err := h(context.Background(), nil, ExplainEvaluationInput{ProjectSlug: "x", FlagKey: "y"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("nil context should not error: %s", firstText(t, res))
	}
	if fake.LastReq.GetContext() != nil {
		t.Error("expected nil context to be forwarded as nil")
	}
}
