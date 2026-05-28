package buildinfo

import (
	"context"
	"errors"
	"testing"
)

func TestServiceName(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"":         "falseflag",
		"api":      "falseflag-api",
		"proxy":    "falseflag-proxy",
		"operator": "falseflag-operator",
	}
	for suffix, want := range cases {
		if got := ServiceName(suffix); got != want {
			t.Errorf("ServiceName(%q) = %q, want %q", suffix, got, want)
		}
	}
}

func TestWithGracefulShutdown_CleanReturn(t *testing.T) {
	t.Parallel()

	code := WithGracefulShutdown("test", func(ctx context.Context) error {
		return nil
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestWithGracefulShutdown_ErrorReturn(t *testing.T) {
	t.Parallel()

	code := WithGracefulShutdown("test", func(ctx context.Context) error {
		return errors.New("boom")
	})
	if code != 1 {
		t.Fatalf("expected exit code 1 on error, got %d", code)
	}
}

func TestWithGracefulShutdown_ContextCancelledIsClean(t *testing.T) {
	t.Parallel()

	code := WithGracefulShutdown("test", func(ctx context.Context) error {
		return context.Canceled
	})
	if code != 0 {
		t.Fatalf("expected exit code 0 for context.Canceled, got %d", code)
	}
}
