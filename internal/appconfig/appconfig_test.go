package appconfig

import "testing"

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
	cfg, err := LoadProxy()
	if err != nil {
		t.Fatalf("LoadProxy returned error: %v", err)
	}
	if cfg.Addr != ":8081" {
		t.Errorf("Addr = %q, want :8081", cfg.Addr)
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
