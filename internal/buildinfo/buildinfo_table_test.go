package buildinfo

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestServiceName_Table(t *testing.T) {
	t.Parallel()
	cases := []struct {
		suffix string
		want   string
	}{
		{"", "falseflag"},
		{"api", "falseflag-api"},
		{"proxy", "falseflag-proxy"},
		{"operator", "falseflag-operator"},
		{"mcp", "falseflag-mcp"},
		{"seed", "falseflag-seed"},
		{"loadgen", "falseflag-loadgen"},
		{"x", "falseflag-x"},
		{"1234", "falseflag-1234"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.suffix, func(t *testing.T) {
			t.Parallel()
			if got := ServiceName(tc.suffix); got != tc.want {
				t.Errorf("ServiceName(%q) = %q, want %q", tc.suffix, got, tc.want)
			}
		})
	}
}

func TestVersionAndCommitDefaults(t *testing.T) {
	t.Parallel()
	if Version == "" {
		t.Errorf("Version should not be empty (default %q)", Version)
	}
	if Commit == "" {
		t.Errorf("Commit should not be empty (default %q)", Commit)
	}
}

func TestName_Constant(t *testing.T) {
	t.Parallel()
	if Name != "falseflag" {
		t.Errorf("Name = %q, want %q", Name, "falseflag")
	}
}

func TestWithGracefulShutdown_VariousErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		code int
	}{
		{"nil", nil, 0},
		{"context-canceled", context.Canceled, 0},
		{"wrapped-canceled", fmt.Errorf("outer: %w", context.Canceled), 0},
		{"deadline-exceeded", context.DeadlineExceeded, 1},
		{"other", errors.New("boom"), 1},
		{"wrapped-other", fmt.Errorf("step: %w", errors.New("io")), 1},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			code := WithGracefulShutdown("svc-"+tc.name, func(ctx context.Context) error {
				return tc.err
			})
			if code != tc.code {
				t.Errorf("code = %d, want %d", code, tc.code)
			}
		})
	}
}

func TestWithGracefulShutdown_ContextIsLive(t *testing.T) {
	t.Parallel()
	var seen bool
	code := WithGracefulShutdown("live", func(ctx context.Context) error {
		if ctx == nil {
			return errors.New("nil ctx")
		}
		select {
		case <-ctx.Done():
			return errors.New("ctx unexpectedly done")
		default:
		}
		seen = true
		return nil
	})
	if !seen || code != 0 {
		t.Errorf("seen=%v code=%d", seen, code)
	}
}

func TestWithGracefulShutdown_SignalCancellation(t *testing.T) {
	// not parallel: signal delivery is process-wide. Keep it brief.
	var wg sync.WaitGroup
	wg.Add(1)
	var got int
	go func() {
		defer wg.Done()
		got = WithGracefulShutdown("sig", func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
	}()
	// Give the goroutine time to install the signal handler.
	time.Sleep(50 * time.Millisecond)
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	wg.Wait()
	if got != 0 {
		t.Errorf("code = %d on SIGTERM-induced cancellation, want 0", got)
	}
}
