// Package sdkgo is the in-process Go SDK for FalseFlag. It polls
// snapshots from the control-plane API and evaluates flags locally
// using internal/eval — the same evaluator the server uses, so a
// decision computed in-SDK is byte-identical to one returned by the
// API's /v1/projects/{slug}/flags/{key}/evaluate endpoint.
//
// The provider interface is OpenFeature-shaped (see
// docs/sdk-openfeature.md). It deliberately omits OpenFeature's
// hooks, finally stages, and provider registry; this is a demo
// SDK, not a full spec implementation.
package sdkgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/depot/falseflag/internal/config"
	"github.com/depot/falseflag/internal/eval"
)

// Decision is re-exported from internal/eval so SDK consumers don't
// import internal/eval directly.
type Decision = eval.Decision

// EvalContext is the attribute map evaluators read. It supports
// nested objects (e.g. {"user": {"id": "u-1"}}).
type EvalContext = map[string]any

// ProviderMetadata identifies the provider in OpenFeature-style
// telemetry. Slice 5 only uses the name.
type ProviderMetadata struct {
	Name string
}

// Provider is the OpenFeature-shaped resolution surface. Demo-quality:
// four resolution methods, no hooks, no finally stage.
type Provider interface {
	Metadata() ProviderMetadata
	BooleanEvaluation(ctx context.Context, key string, def bool, evalCtx EvalContext) Decision
	StringEvaluation(ctx context.Context, key string, def string, evalCtx EvalContext) Decision
	NumberEvaluation(ctx context.Context, key string, def float64, evalCtx EvalContext) Decision
	ObjectEvaluation(ctx context.Context, key string, def any, evalCtx EvalContext) Decision
}

// Snapshot is the per-project bundle the SDK polls. It is immutable
// once loaded; new poll results replace the whole pointer.
type Snapshot struct {
	ID        string
	Version   int
	CreatedAt time.Time
	Flags     map[string]*config.Compiled
}

// Options configures a Client.
type Options struct {
	// BaseURL is the FalseFlag REST API base URL, e.g. "http://localhost:8080".
	BaseURL string
	// ProjectSlug scopes every snapshot poll to one project.
	ProjectSlug string
	// PollInterval defaults to 10s. Set to a negative duration to disable
	// background polling (Start still runs one poll).
	PollInterval time.Duration
	// HTTPClient lets callers inject a custom transport. Defaults to
	// http.DefaultClient.
	HTTPClient *http.Client
	// Logger receives warnings (poll errors, compile failures). Defaults
	// to slog.Default().
	Logger *slog.Logger
}

// Client holds the snapshot cache and the polling goroutine.
type Client struct {
	opts     Options
	http     *http.Client
	logger   *slog.Logger
	snap     atomic.Pointer[Snapshot]
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewClient constructs a Client. Start must be called before any
// evaluation to populate the snapshot cache.
func NewClient(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, errors.New("sdkgo: BaseURL is required")
	}
	if opts.ProjectSlug == "" {
		return nil, errors.New("sdkgo: ProjectSlug is required")
	}
	if opts.PollInterval == 0 {
		opts.PollInterval = 10 * time.Second
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Client{
		opts:   opts,
		http:   opts.HTTPClient,
		logger: opts.Logger,
		stopCh: make(chan struct{}),
	}, nil
}

// Start performs the first snapshot poll synchronously, then (unless
// PollInterval<0) spawns a goroutine to refresh the snapshot on the
// configured interval. Start returns the error from the first poll.
func (c *Client) Start(ctx context.Context) error {
	if err := c.pollOnce(ctx); err != nil {
		return err
	}
	if c.opts.PollInterval > 0 {
		go c.loop(ctx)
	}
	return nil
}

// Stop signals the polling goroutine to exit. Safe to call multiple
// times.
func (c *Client) Stop() {
	c.stopOnce.Do(func() { close(c.stopCh) })
}

// Snapshot returns the current snapshot, or nil if Start has not
// completed a successful poll yet.
func (c *Client) Snapshot() *Snapshot { return c.snap.Load() }

func (c *Client) loop(ctx context.Context) {
	t := time.NewTicker(c.opts.PollInterval)
	defer t.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-ctx.Done():
			return
		case <-t.C:
			if err := c.pollOnce(ctx); err != nil {
				c.logger.Warn("sdkgo: poll error",
					"project", c.opts.ProjectSlug,
					"err", err,
				)
			}
		}
	}
}

func (c *Client) pollOnce(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1/projects/%s/snapshots/latest",
		c.opts.BaseURL, c.opts.ProjectSlug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through
	case http.StatusNotFound:
		// No snapshot compiled yet. Keep any last-good snapshot.
		return nil
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("sdkgo: snapshot poll status %d: %s",
			resp.StatusCode, string(body))
	}

	var body struct {
		ID        string    `json:"id"`
		Version   int       `json:"version"`
		CreatedAt time.Time `json:"created_at"`
		Compiled  struct {
			Flags map[string]json.RawMessage `json:"flags"`
		} `json:"compiled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("sdkgo: decoding snapshot: %w", err)
	}

	flags := make(map[string]*config.Compiled, len(body.Compiled.Flags))
	for key, raw := range body.Compiled.Flags {
		// Snapshots may contain predicates from any strategy. CEL is
		// the superset, so compile with StrategyCEL — that allows the
		// full predicate set including cel{} blocks. Non-CEL flags
		// compile fine since their predicate trees are a CEL-strategy
		// subset.
		compiled, err := config.Compile(config.StrategyCEL, raw)
		if err != nil {
			c.logger.Warn("sdkgo: compile failed",
				"flag", key,
				"err", err,
			)
			continue
		}
		flags[key] = compiled
	}

	c.snap.Store(&Snapshot{
		ID:        body.ID,
		Version:   body.Version,
		CreatedAt: body.CreatedAt,
		Flags:     flags,
	})
	return nil
}

// Evaluate runs the local evaluator against the cached snapshot.
//
// If no snapshot is loaded, returns Decision{Reason: "error"} with
// nil value — callers should pass a default down through the
// provider's typed methods.
//
// If the flag is missing from the snapshot, returns
// Decision{Reason: "default"} with nil value — the caller's default
// is substituted by the provider.
func (c *Client) Evaluate(key string, evalCtx EvalContext) Decision {
	snap := c.snap.Load()
	if snap == nil {
		return Decision{Reason: eval.ReasonError}
	}
	compiled, ok := snap.Flags[key]
	if !ok {
		return Decision{Reason: eval.ReasonDefault, Version: snap.Version}
	}
	d, err := eval.Evaluate(compiled, evalCtx, snap.Version)
	if err != nil {
		return Decision{Reason: eval.ReasonError, Version: snap.Version}
	}
	return d
}
