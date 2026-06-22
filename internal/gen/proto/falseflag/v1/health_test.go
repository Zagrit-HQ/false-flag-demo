package falseflagv1

import "testing"

// TestGeneratedHealthMessagesCompileAndConstruct is a compile check asserting
// that the generated proto types are usable. It exists only to make sure
// `make generate` produces something we can link against.
func TestGeneratedHealthMessagesCompileAndConstruct(t *testing.T) {
	t.Parallel()

	req := &CheckRequest{Service: "falseflag-api"}
	if req.GetService() != "falseflag-api" {
		t.Fatalf("CheckRequest.Service round-trip failed")
	}

	res := &CheckResponse{
		Status:  CheckResponse_SERVING_STATUS_SERVING,
		Service: "falseflag-api",
		Version: "dev",
	}
	if res.GetStatus() != CheckResponse_SERVING_STATUS_SERVING {
		t.Fatalf("CheckResponse.Status mismatch")
	}
}
