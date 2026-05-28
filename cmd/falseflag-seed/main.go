// Command falseflag-seed populates the running FalseFlag API with a
// believable demo dataset: 3 projects, ~25 flags spread across
// strategies, environments, segments, and a compiled snapshot per
// project.
//
// It is idempotent: 409 ALREADY_EXISTS responses from the API are
// treated as success so re-running the seed is safe.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/depot/falseflag/internal/logging"
)

func main() {
	log := logging.New("seed")
	if err := run(context.Background(), log); err != nil {
		log.Error("seed failed", "err", err)
		os.Exit(1)
	}
	log.Info("seed complete")
}

func run(ctx context.Context, log *slog.Logger) error {
	baseURL := getenv("FALSEFLAG_API_BASE_URL", "http://localhost:8080")
	actor := getenv("FALSEFLAG_SEED_ACTOR", "seed/falseflag-seed")
	c := &client{base: baseURL, actor: actor, http: http.DefaultClient}

	if err := c.waitForHealth(ctx, 30*time.Second); err != nil {
		return fmt.Errorf("api never became healthy: %w", err)
	}
	log.Info("api ready", "base", baseURL)

	for _, proj := range demoProjects {
		if err := c.createProject(ctx, proj); err != nil {
			return fmt.Errorf("create project %s: %w", proj.Slug, err)
		}
		log.Info("project ok", "slug", proj.Slug)

		for _, env := range proj.Environments {
			if err := c.createEnvironment(ctx, proj.Slug, env); err != nil {
				return fmt.Errorf("create env %s/%s: %w", proj.Slug, env.Slug, err)
			}
		}
		for _, flag := range proj.Flags {
			if err := c.publishFlag(ctx, proj.Slug, flag); err != nil {
				return fmt.Errorf("publish flag %s/%s: %w", proj.Slug, flag.Key, err)
			}
			log.Info("flag ok", "project", proj.Slug, "flag", flag.Key, "strategy", flag.Strategy)
		}
		if err := c.compileSnapshot(ctx, proj.Slug); err != nil {
			return fmt.Errorf("compile snapshot %s: %w", proj.Slug, err)
		}
		log.Info("snapshot ok", "project", proj.Slug)
	}
	return nil
}

type client struct {
	base  string
	actor string
	http  *http.Client
}

func (c *client) waitForHealth(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/healthz", nil)
		resp, err := c.http.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return errors.New("timed out waiting for API /healthz")
		}
		time.Sleep(time.Second)
	}
}

func (c *client) do(ctx context.Context, method, path string, body any) ([]byte, int, error) {
	var b io.Reader
	if body != nil {
		j, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		b = bytes.NewReader(j)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, b)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Actor", c.actor)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	return data, resp.StatusCode, nil
}

func (c *client) createProject(ctx context.Context, p demoProject) error {
	body := map[string]any{
		"slug":            p.Slug,
		"display_name":    p.DisplayName,
		"config_strategy": p.Strategy,
	}
	data, status, err := c.do(ctx, http.MethodPost, "/v1/projects", body)
	if err != nil {
		return err
	}
	if status == 200 || status == 201 || status == 409 {
		return nil
	}
	return fmt.Errorf("HTTP %d: %s", status, data)
}

func (c *client) createEnvironment(ctx context.Context, projectSlug string, e demoEnvironment) error {
	body := map[string]any{
		"slug": e.Slug,
		"name": e.DisplayName,
	}
	data, status, err := c.do(ctx, http.MethodPost,
		"/v1/projects/"+projectSlug+"/environments", body)
	if err != nil {
		return err
	}
	if status == 200 || status == 201 || status == 409 {
		return nil
	}
	return fmt.Errorf("HTTP %d: %s", status, data)
}

func (c *client) publishFlag(ctx context.Context, projectSlug string, f demoFlag) error {
	// Step 1: ensure the flag row exists. POST /flags creates it.
	createBody := map[string]any{
		"key":           f.Key,
		"name":          f.Name,
		"description":   f.Description,
		"value_type":    f.ValueType,
		"default_value": f.IR["default"],
	}
	data, status, err := c.do(ctx, http.MethodPost,
		"/v1/projects/"+projectSlug+"/flags", createBody)
	if err != nil {
		return err
	}
	if status != 200 && status != 201 && status != 409 {
		return fmt.Errorf("create flag HTTP %d: %s", status, data)
	}

	// Step 2: publish a version. Always include source_text so the
	// dashboard view route renders real author-authored source instead
	// of falling back to the "compiled IR — original source not stored"
	// caption. For TS flags the server compiles source_text via
	// esbuild+goja and overrides the IR we pass in `source`.
	publishBody := map[string]any{
		"strategy":    f.Strategy,
		"source":      f.IR,
		"source_text": f.SourceText,
	}
	data, status, err = c.do(ctx, http.MethodPut,
		"/v1/projects/"+projectSlug+"/flags/"+f.Key, publishBody)
	if err != nil {
		return err
	}
	if status == 200 || status == 201 {
		return nil
	}
	return fmt.Errorf("publish HTTP %d: %s", status, data)
}

func (c *client) compileSnapshot(ctx context.Context, projectSlug string) error {
	data, status, err := c.do(ctx, http.MethodPost,
		"/v1/projects/"+projectSlug+"/snapshots", map[string]any{})
	if err != nil {
		return err
	}
	if status == 200 || status == 201 {
		return nil
	}
	return fmt.Errorf("compile HTTP %d: %s", status, data)
}

func getenv(k, fallback string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return fallback
}
