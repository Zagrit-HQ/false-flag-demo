package appconfig

import (
	"testing"
	"time"
)

func TestLoadAPI_Defaults(t *testing.T) {
	t.Setenv("FALSEFLAG_API_ADDR", "")
	t.Setenv("FALSEFLAG_API_RPC_ADDR", "")
	t.Setenv("FALSEFLAG_DATABASE_URL", "")

	cfg, err := LoadAPI()
	if err != nil {
		t.Fatalf("LoadAPI returned error: %v", err)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", cfg.Addr)
	}
	if cfg.RPCAddr != ":8090" {
		t.Errorf("RPCAddr = %q, want :8090", cfg.RPCAddr)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
}

func TestLoadAPI_EnvOverride(t *testing.T) {
	t.Setenv("FALSEFLAG_API_ADDR", ":9090")
	t.Setenv("FALSEFLAG_API_RPC_ADDR", ":9191")
	t.Setenv("FALSEFLAG_DATABASE_URL", "postgres://user:pw@localhost/falseflag")

	cfg, err := LoadAPI()
	if err != nil {
		t.Fatalf("LoadAPI returned error: %v", err)
	}
	if cfg.Addr != ":9090" {
		t.Errorf("Addr = %q, want :9090", cfg.Addr)
	}
	if cfg.RPCAddr != ":9191" {
		t.Errorf("RPCAddr = %q, want :9191", cfg.RPCAddr)
	}
	if cfg.DatabaseURL != "postgres://user:pw@localhost/falseflag" {
		t.Errorf("DatabaseURL was not overridden: %q", cfg.DatabaseURL)
	}
}

func TestLoadProxy_Defaults(t *testing.T) {
	t.Setenv("FALSEFLAG_PROXY_ADDR", "")
	t.Setenv("FALSEFLAG_PROXY_POLL_INTERVAL", "")
	cfg, err := LoadProxy()
	if err != nil {
		t.Fatalf("LoadProxy returned error: %v", err)
	}
	if cfg.Addr != ":8081" {
		t.Errorf("Addr = %q, want :8081", cfg.Addr)
	}
	if cfg.PollInterval != 0 {
		t.Errorf("PollInterval = %s, want SDK default marker 0", cfg.PollInterval)
	}
}

func TestLoadProxy_PollInterval(t *testing.T) {
	t.Setenv("FALSEFLAG_PROXY_POLL_INTERVAL", "250ms")
	cfg, err := LoadProxy()
	if err != nil {
		t.Fatalf("LoadProxy returned error: %v", err)
	}
	if cfg.PollInterval != 250*time.Millisecond {
		t.Errorf("PollInterval = %s, want 250ms", cfg.PollInterval)
	}
}

func TestLoadProxy_InvalidPollInterval(t *testing.T) {
	t.Setenv("FALSEFLAG_PROXY_POLL_INTERVAL", "soon")
	if _, err := LoadProxy(); err == nil {
		t.Fatal("expected invalid poll interval to return an error")
	}
}

func TestLoadOperator_LeaderElectParsing(t *testing.T) {
	t.Setenv("FALSEFLAG_OPERATOR_LEADER_ELECT", "true")
	cfg, err := LoadOperator()
	if err != nil {
		t.Fatalf("LoadOperator returned error: %v", err)
	}
	if !cfg.LeaderElect {
		t.Errorf("LeaderElect = false, want true")
	}
	if cfg.MetricsAddr != ":8082" {
		t.Errorf("MetricsAddr = %q, want :8082", cfg.MetricsAddr)
	}
}

func TestLoadOperator_LeaderElectInvalidFallsBack(t *testing.T) {
	t.Setenv("FALSEFLAG_OPERATOR_LEADER_ELECT", "definitely-not-a-bool")
	cfg, err := LoadOperator()
	if err != nil {
		t.Fatalf("LoadOperator returned error: %v", err)
	}
	if cfg.LeaderElect {
		t.Errorf("LeaderElect = true on invalid input, want false fallback")
	}
}
