package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/depot/falseflag/internal/appconfig"
	"github.com/depot/falseflag/internal/gen/openapi"
	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/gen/proto/falseflag/v1/falseflagv1connect"
	"github.com/depot/falseflag/internal/logging"
	"github.com/depot/falseflag/internal/server"
	"github.com/depot/falseflag/internal/store"
)

// TestRESTConnectParity boots the full Server in-process, then drives
// the same happy-path workflow over the REST handler (via httptest) and
// the Connect handler (via an in-process Connect client). The two must
// agree at the domain-type level for every operation.
//
// Gated on FALSEFLAG_TEST_DATABASE_URL because it needs a live DB to
// exercise the store. Skipped otherwise.
func TestRESTConnectParity(t *testing.T) {
	url := os.Getenv("FALSEFLAG_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set FALSEFLAG_TEST_DATABASE_URL to enable the contract test")
	}

	ctx := context.Background()
	s, err := store.Open(ctx, url)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(s.Close)
	if err := s.Migrate(ctx, nil); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := s.TruncateForTest(ctx); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	srv, err := server.New(ctx, appconfig.APIConfig{Addr: ":0", RPCAddr: ":0"}, logging.New("test"), server.Deps{Store: s})
	if err != nil {
		t.Fatalf("server new: %v", err)
	}

	restTS := httptest.NewServer(srv.Handler())
	t.Cleanup(restTS.Close)
	rpcTS := httptest.NewServer(srv.RPCHandler())
	t.Cleanup(rpcTS.Close)

	restURL := restTS.URL
	rpcClient := falseflagv1connect.NewProjectsServiceClient(rpcTS.Client(), rpcTS.URL)
	envClient := falseflagv1connect.NewEnvironmentsServiceClient(rpcTS.Client(), rpcTS.URL)
	flagClient := falseflagv1connect.NewFlagsServiceClient(rpcTS.Client(), rpcTS.URL)
	snapClient := falseflagv1connect.NewSnapshotsServiceClient(rpcTS.Client(), rpcTS.URL)
	evalClient := falseflagv1connect.NewEvaluationServiceClient(rpcTS.Client(), rpcTS.URL)
	auditClient := falseflagv1connect.NewAuditServiceClient(rpcTS.Client(), rpcTS.URL)

	t.Run("create project parity", func(t *testing.T) {
		restCreate(t, restURL+"/v1/projects",
			`{"slug":"parity","display_name":"Parity","config_strategy":"json"}`,
			http.StatusCreated)

		// Listing via REST + Connect must both surface the project. We
		// check containment rather than length so the test stays stable
		// against any seed data from concurrent store tests.
		restList := restGet[openapi.ProjectList](t, restURL+"/v1/projects")
		if !containsProject(restList.Items, "parity") {
			t.Fatalf("REST list missing parity: %+v", restList.Items)
		}
		rpcList, err := rpcClient.ListProjects(ctx, connect.NewRequest(&pb.ListProjectsRequest{}))
		if err != nil {
			t.Fatalf("rpc list: %v", err)
		}
		if !containsProjectProto(rpcList.Msg.Items, "parity") {
			t.Fatalf("RPC list missing parity: %+v", rpcList.Msg.Items)
		}
	})

	t.Run("create environment via RPC, fetch via REST", func(t *testing.T) {
		_, err := envClient.CreateEnvironment(ctx, connect.NewRequest(&pb.CreateEnvironmentRequest{
			ProjectSlug: "parity",
			Slug:        "prod",
			Name:        "Production",
		}))
		if err != nil {
			t.Fatalf("rpc create env: %v", err)
		}
		body := restGet[openapi.Environment](t, restURL+"/v1/projects/parity/environments/prod")
		if body.Slug != "prod" || body.Name != "Production" {
			t.Errorf("env via REST = %+v", body)
		}
	})

	t.Run("create + publish flag via REST, evaluate via RPC", func(t *testing.T) {
		restCreate(t, restURL+"/v1/projects/parity/flags",
			`{"key":"banner","name":"Banner","value_type":"boolean","default_value":false}`,
			http.StatusCreated)

		restCreate(t, restURL+"/v1/projects/parity/flags/banner",
			`{"strategy":"json","source":{"value_type":"boolean","default":false,"rules":[{"id":"all","when":{"kind":"always"},"value":true}]}}`,
			http.StatusOK,
			"PUT")

		ctxStruct, _ := structpb.NewStruct(map[string]any{})
		dec, err := evalClient.Evaluate(ctx, connect.NewRequest(&pb.EvaluateRequest{
			ProjectSlug: "parity",
			Key:         "banner",
			Context:     ctxStruct,
		}))
		if err != nil {
			t.Fatalf("rpc evaluate: %v", err)
		}
		if dec.Msg.Decision.Reason != pb.DecisionReason_DECISION_REASON_RULE_MATCHED {
			t.Errorf("RPC reason = %v, want RULE_MATCHED", dec.Msg.Decision.Reason)
		}
	})

	t.Run("compile snapshot via RPC, fetch latest via REST", func(t *testing.T) {
		snapResp, err := snapClient.CompileSnapshot(ctx, connect.NewRequest(&pb.CompileSnapshotRequest{
			ProjectSlug: "parity",
		}))
		if err != nil {
			t.Fatalf("rpc compile: %v", err)
		}
		got := restGet[openapi.Snapshot](t, restURL+"/v1/projects/parity/snapshots/latest")
		if int32(got.Version) != snapResp.Msg.Snapshot.Version {
			t.Errorf("snapshot version mismatch REST=%d RPC=%d", got.Version, snapResp.Msg.Snapshot.Version)
		}
	})

	t.Run("audit search returns publish_version actor records", func(t *testing.T) {
		// REST publish above is unauthored; we add one with X-Actor here.
		req := httptest.NewRequest("PUT", restURL+"/v1/projects/parity/flags/banner",
			strings.NewReader(`{"strategy":"json","source":{"value_type":"boolean","default":true,"rules":[]}}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Actor", "contract-test")
		req.RequestURI = ""
		req.URL, _ = req.URL.Parse(restURL + "/v1/projects/parity/flags/banner")
		resp, err := restTS.Client().Do(req)
		if err != nil {
			t.Fatalf("publish-with-actor: %v", err)
		}
		_ = resp.Body.Close()

		// RPC audit list must show the actor.
		audit, err := auditClient.ListAuditEvents(ctx, connect.NewRequest(&pb.ListAuditEventsRequest{
			ProjectSlug: "parity",
			Limit:       50,
		}))
		if err != nil {
			t.Fatalf("rpc audit: %v", err)
		}
		var sawActor bool
		for _, ev := range audit.Msg.Items {
			if ev.Actor != nil && *ev.Actor == "contract-test" {
				sawActor = true
			}
		}
		if !sawActor {
			t.Errorf("audit search did not return contract-test actor")
		}
	})

	t.Run("connect not-found is HTTP 404 via REST too", func(t *testing.T) {
		_, err := flagClient.GetFlag(ctx, connect.NewRequest(&pb.GetFlagRequest{
			ProjectSlug: "parity",
			Key:         "does-not-exist",
		}))
		if err == nil {
			t.Fatalf("expected error")
		}
		var ce *connect.Error
		if !errorsAs(err, &ce) {
			t.Fatalf("not a connect.Error: %v", err)
		}
		if ce.Code() != connect.CodeNotFound {
			t.Errorf("code = %v, want NotFound", ce.Code())
		}

		resp, _ := restTS.Client().Get(restURL + "/v1/projects/parity/flags/does-not-exist")
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("REST status = %d, want 404", resp.StatusCode)
		}
		_ = resp.Body.Close()
	})
}

// restCreate POSTs JSON and asserts the HTTP code. The optional method
// arg lets the same helper drive PUT requests too.
func restCreate(t *testing.T, url, body string, want int, method ...string) {
	t.Helper()
	m := http.MethodPost
	if len(method) > 0 {
		m = method[0]
	}
	req, err := http.NewRequest(m, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("new req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", m, url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != want {
		t.Fatalf("%s %s: status = %d, want %d", m, url, resp.StatusCode, want)
	}
}

// restGet GETs the URL and decodes the JSON body into T.
func restGet[T any](t *testing.T, url string) T {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status = %d", url, resp.StatusCode)
	}
	var v T
	if err := decodeJSON(resp, &v); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
	return v
}

func decodeJSON(resp *http.Response, v any) error {
	return jsonDecoder(resp).Decode(v)
}
