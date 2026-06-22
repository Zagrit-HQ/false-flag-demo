package server_test

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/depot/falseflag/internal/gen/openapi"
	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
)

func containsProject(items []openapi.Project, slug string) bool {
	for _, p := range items {
		if p.Slug == slug {
			return true
		}
	}
	return false
}

func containsProjectProto(items []*pb.Project, slug string) bool {
	for _, p := range items {
		if p.Slug == slug {
			return true
		}
	}
	return false
}

// jsonDecoder wraps a *http.Response body decoder.
func jsonDecoder(resp *http.Response) *json.Decoder {
	return json.NewDecoder(resp.Body)
}

// errorsAs is errors.As as a helper because the contract test reuses
// it across subtests. Returns true if target was populated.
func errorsAs(err error, target any) bool {
	return errors.As(err, target)
}
