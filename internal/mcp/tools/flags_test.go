package tools

import (
	"context"
	"strings"
	"testing"

	pb "github.com/depot/falseflag/internal/gen/proto/falseflag/v1"
	"github.com/depot/falseflag/internal/operator/clientapi"
)

func TestListFlags_PassesProjectSlug(t *testing.T) {
	t.Parallel()
	fake := &fakeFlags{List: []*pb.Flag{{Key: "beta-checkout"}}}
	h := listFlags(&clientapi.Client{Flags: fake})

	res, _, err := h(context.Background(), nil, ListFlagsInput{ProjectSlug: "acme-web"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", firstText(t, res))
	}
	if got := fake.LastListReq.GetProjectSlug(); got != "acme-web" {
		t.Errorf("project slug not forwarded: got %q", got)
	}
	if !strings.Contains(firstText(t, res), "beta-checkout") {
		t.Errorf("expected flag key in body, got %s", firstText(t, res))
	}
}

func TestListFlags_NotFound(t *testing.T) {
	t.Parallel()
	fake := &fakeFlags{ListErr: notFoundErr("project missing")}
	h := listFlags(&clientapi.Client{Flags: fake})
	res, _, err := h(context.Background(), nil, ListFlagsInput{ProjectSlug: "nope"})
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

func TestGetFlag_PublishedVersion(t *testing.T) {
	t.Parallel()
	fake := &fakeFlags{
		Get:           &pb.Flag{Key: "beta-checkout"},
		LatestVersion: &pb.FlagVersion{Version: 3},
	}
	h := getFlag(&clientapi.Client{Flags: fake})

	res, _, err := h(context.Background(), nil, GetFlagInput{ProjectSlug: "acme", FlagKey: "beta-checkout"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", firstText(t, res))
	}
	body := firstText(t, res)
	if !strings.Contains(body, `"version":3`) {
		t.Errorf("expected version 3 in body, got %s", body)
	}
	if strings.Contains(body, `"published_version":null`) {
		t.Error("expected published_version to be populated, but it was null")
	}
	if strings.Contains(body, `"note"`) {
		t.Error("note field should be absent when a version exists")
	}
}

func TestGetFlag_NoPublishedVersion(t *testing.T) {
	t.Parallel()
	fake := &fakeFlags{
		Get:           &pb.Flag{Key: "new-flag"},
		LatestVersion: nil,
	}
	h := getFlag(&clientapi.Client{Flags: fake})

	res, _, err := h(context.Background(), nil, GetFlagInput{ProjectSlug: "acme", FlagKey: "new-flag"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected error result: %s", firstText(t, res))
	}
	body := firstText(t, res)
	if !strings.Contains(body, `"published_version":null`) {
		t.Errorf("expected null published_version, got %s", body)
	}
	if !strings.Contains(body, "never been published") {
		t.Errorf("expected 'never been published' note, got %s", body)
	}
}

func TestGetFlag_NotFound(t *testing.T) {
	t.Parallel()
	fake := &fakeFlags{GetErr: notFoundErr("flag not found")}
	h := getFlag(&clientapi.Client{Flags: fake})
	res, _, err := h(context.Background(), nil, GetFlagInput{ProjectSlug: "acme", FlagKey: "missing"})
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
