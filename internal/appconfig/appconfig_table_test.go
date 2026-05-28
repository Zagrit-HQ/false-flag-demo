package appconfig

import (
	"fmt"
	"testing"
)

func TestGetEnv_Table(t *testing.T) {
	cases := []struct {
		name     string
		set      bool
		val      string
		fallback string
		want     string
	}{
		{"unset", false, "", "fb", "fb"},
		{"empty", true, "", "fb", "fb"},
		{"populated", true, "x", "fb", "x"},
		{"whitespace", true, " ", "fb", " "},
		{"unicode", true, "漢字", "fb", "漢字"},
		{"colon-prefix", true, ":9999", "fb", ":9999"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			key := "FALSEFLAG_TEST_GETENV_" + tc.name
			if tc.set {
				t.Setenv(key, tc.val)
			}
			if got := getEnv(key, tc.fallback); got != tc.want {
				t.Errorf("getEnv(%q,%q) = %q want %q", tc.val, tc.fallback, got, tc.want)
			}
		})
	}
}

func TestGetEnvBool_Table(t *testing.T) {
	cases := []struct {
		name     string
		set      bool
		val      string
		fallback bool
		want     bool
	}{
		{"unset-fb-false", false, "", false, false},
		{"unset-fb-true", false, "", true, true},
		{"empty-fb-true", true, "", true, true},
		{"true", true, "true", false, true},
		{"TRUE", true, "TRUE", false, true},
		{"1", true, "1", false, true},
		{"T", true, "T", false, true},
		{"false", true, "false", true, false},
		{"FALSE", true, "FALSE", true, false},
		{"0", true, "0", true, false},
		{"F", true, "F", true, false},
		{"yes-invalid", true, "yes", true, true},
		{"no-invalid", true, "no", false, false},
		{"junk-invalid", true, "definitely-not", true, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			key := "FALSEFLAG_TEST_GETENVBOOL_" + tc.name
			if tc.set {
				t.Setenv(key, tc.val)
			}
			if got := getEnvBool(key, tc.fallback); got != tc.want {
				t.Errorf("getEnvBool(%q,%v) = %v want %v", tc.val, tc.fallback, got, tc.want)
			}
		})
	}
}

func TestLoadAPI_Variants(t *testing.T) {
	cases := []struct {
		name     string
		addr     string
		rpcAddr  string
		db       string
		wantAddr string
		wantRPC  string
		wantDB   string
	}{
		{"defaults", "", "", "", ":8080", ":8090", ""},
		{"addr-only", ":9080", "", "", ":9080", ":8090", ""},
		{"rpc-only", "", ":9090", "", ":8080", ":9090", ""},
		{"with-db", "", "", "postgres://x/y", ":8080", ":8090", "postgres://x/y"},
		{"all-set", ":9080", ":9090", "postgres://a/b", ":9080", ":9090", "postgres://a/b"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FALSEFLAG_API_ADDR", tc.addr)
			t.Setenv("FALSEFLAG_API_RPC_ADDR", tc.rpcAddr)
			t.Setenv("FALSEFLAG_DATABASE_URL", tc.db)
			cfg, err := LoadAPI()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Addr != tc.wantAddr || cfg.RPCAddr != tc.wantRPC || cfg.DatabaseURL != tc.wantDB {
				t.Errorf("got %+v", cfg)
			}
		})
	}
}

func TestLoadProxy_Variants(t *testing.T) {
	cases := []struct {
		name     string
		addr     string
		base     string
		slug     string
		wantAddr string
		wantBase string
		wantSlug string
	}{
		{"defaults", "", "", "", ":8081", "http://localhost:8080", ""},
		{"slug", "", "", "demo", ":8081", "http://localhost:8080", "demo"},
		{"base", "", "http://api.example.com", "", ":8081", "http://api.example.com", ""},
		{"all", ":9081", "http://api", "x", ":9081", "http://api", "x"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FALSEFLAG_PROXY_ADDR", tc.addr)
			t.Setenv("FALSEFLAG_API_BASE_URL", tc.base)
			t.Setenv("FALSEFLAG_PROXY_PROJECT_SLUG", tc.slug)
			cfg, err := LoadProxy()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Addr != tc.wantAddr || cfg.APIBaseURL != tc.wantBase || cfg.ProjectSlug != tc.wantSlug {
				t.Errorf("got %+v", cfg)
			}
		})
	}
}

func TestLoadOperator_Variants(t *testing.T) {
	cases := []struct {
		name        string
		leaderElect string
		want        bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"1", "1", true},
		{"0", "0", false},
		{"invalid", "yes", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FALSEFLAG_OPERATOR_LEADER_ELECT", tc.leaderElect)
			cfg, err := LoadOperator()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.LeaderElect != tc.want {
				t.Errorf("LeaderElect = %v want %v", cfg.LeaderElect, tc.want)
			}
		})
	}
}

func TestLoadMCP_Defaults(t *testing.T) {
	t.Setenv("FALSEFLAG_MCP_ADDR", "")
	t.Setenv("FALSEFLAG_MCP_HEALTH_ADDR", "")
	t.Setenv("FALSEFLAG_API_RPC_ADDR", "")
	t.Setenv("FALSEFLAG_MCP_ACTOR", "")
	cfg, err := LoadMCP()
	if err != nil {
		t.Fatal(err)
	}
	want := MCPConfig{
		Addr:       ":8091",
		HealthAddr: ":8092",
		APIBaseURL: "http://localhost:8090",
		Actor:      "mcp/falseflag-mcp",
	}
	if cfg != want {
		t.Errorf("got %+v want %+v", cfg, want)
	}
}

func TestLoadMCP_Override(t *testing.T) {
	t.Setenv("FALSEFLAG_MCP_ADDR", ":9191")
	t.Setenv("FALSEFLAG_MCP_HEALTH_ADDR", ":9292")
	t.Setenv("FALSEFLAG_API_RPC_ADDR", "http://api:8090")
	t.Setenv("FALSEFLAG_MCP_ACTOR", "alice")
	cfg, _ := LoadMCP()
	if cfg.Addr != ":9191" || cfg.HealthAddr != ":9292" || cfg.APIBaseURL != "http://api:8090" || cfg.Actor != "alice" {
		t.Errorf("got %+v", cfg)
	}
}

func TestLoadOperator_AllOverrides(t *testing.T) {
	t.Setenv("FALSEFLAG_OPERATOR_METRICS_ADDR", ":9000")
	t.Setenv("FALSEFLAG_OPERATOR_HEALTH_ADDR", ":9001")
	t.Setenv("FALSEFLAG_OPERATOR_LEADER_ELECT", "true")
	t.Setenv("FALSEFLAG_API_BASE_URL", "http://x")
	t.Setenv("FALSEFLAG_OPERATOR_ACTOR", "bot")
	cfg, _ := LoadOperator()
	want := OperatorConfig{
		MetricsAddr: ":9000", HealthProbeAddr: ":9001", LeaderElect: true,
		APIBaseURL: "http://x", Actor: "bot",
	}
	if cfg != want {
		t.Errorf("got %+v want %+v", cfg, want)
	}
}

func TestLoadOperator_Defaults(t *testing.T) {
	for _, k := range []string{
		"FALSEFLAG_OPERATOR_METRICS_ADDR",
		"FALSEFLAG_OPERATOR_HEALTH_ADDR",
		"FALSEFLAG_OPERATOR_LEADER_ELECT",
		"FALSEFLAG_API_BASE_URL",
		"FALSEFLAG_OPERATOR_ACTOR",
	} {
		t.Setenv(k, "")
	}
	cfg, _ := LoadOperator()
	if cfg.MetricsAddr != ":8082" {
		t.Errorf("MetricsAddr = %q", cfg.MetricsAddr)
	}
	if cfg.HealthProbeAddr != ":8083" {
		t.Errorf("HealthProbeAddr = %q", cfg.HealthProbeAddr)
	}
	if cfg.LeaderElect {
		t.Errorf("LeaderElect = true want false")
	}
	if cfg.Actor != "controller/falseflag-operator" {
		t.Errorf("Actor = %q", cfg.Actor)
	}
	if cfg.APIBaseURL == "" {
		t.Errorf("APIBaseURL empty")
	}
}

func TestGetEnv_LotsOfKeysIndependent(t *testing.T) {
	// Each subtest hits a unique key so they can run in parallel under
	// t.Setenv (which only blocks parallel with siblings sharing keys).
	for i := 0; i < 32; i++ {
		i := i
		t.Run(fmt.Sprintf("k-%d", i), func(t *testing.T) {
			key := fmt.Sprintf("FALSEFLAG_TEST_INDEP_%d", i)
			val := fmt.Sprintf("val-%d", i)
			t.Setenv(key, val)
			if got := getEnv(key, "fb"); got != val {
				t.Errorf("got %q want %q", got, val)
			}
		})
	}
}
