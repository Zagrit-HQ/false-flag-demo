// Package appconfig loads runtime configuration for each FalseFlag binary
// from environment variables.
//
// This is intentionally separate from internal/config, which holds the
// project-scoped configuration strategy implementations (JSON, CEL,
// TypeScript). They sound similar but cover unrelated concerns: appconfig
// answers "what address does this binary listen on?", config answers "how
// is a project's flag configuration authored and compiled?"
package appconfig

import (
	"os"
	"strconv"
)

// APIConfig is the runtime config for cmd/falseflag-api.
type APIConfig struct {
	Addr        string // FALSEFLAG_API_ADDR, default ":8080"
	RPCAddr     string // FALSEFLAG_API_RPC_ADDR, default ":8090"
	DatabaseURL string // FALSEFLAG_DATABASE_URL, optional in slice 1
}

// ProxyConfig is the runtime config for cmd/falseflag-proxy.
type ProxyConfig struct {
	Addr string // FALSEFLAG_PROXY_ADDR, default ":8081"
	// APIBaseURL is the REST API the proxy polls snapshots from.
	APIBaseURL string // FALSEFLAG_API_BASE_URL, default "http://localhost:8080"
	// ProjectSlug scopes snapshot polling to a single project. When
	// empty the proxy still boots and serves /healthz with status
	// "no_project" so the binary is debuggable in compose without
	// failing fast.
	ProjectSlug string // FALSEFLAG_PROXY_PROJECT_SLUG, optional
}

// OperatorConfig is the runtime config for cmd/falseflag-operator.
type OperatorConfig struct {
	MetricsAddr     string // FALSEFLAG_OPERATOR_METRICS_ADDR, default ":8082"
	HealthProbeAddr string // FALSEFLAG_OPERATOR_HEALTH_ADDR, default ":8083"
	LeaderElect     bool   // FALSEFLAG_OPERATOR_LEADER_ELECT, default false

	// APIBaseURL is the FalseFlag Connect endpoint the operator
	// reconciles against. Default points at the in-cluster API
	// service on the slice-3 RPC port.
	APIBaseURL string // FALSEFLAG_API_BASE_URL, default "http://falseflag-api.default.svc.cluster.local:8090"

	// Actor is the X-Actor header value attached to every API
	// request. Demo-only attribution.
	Actor string // FALSEFLAG_OPERATOR_ACTOR, default "controller/falseflag-operator"
}

// MCPConfig is the runtime config for cmd/falseflag-mcp.
type MCPConfig struct {
	// Addr is the Streamable HTTP listener address for the MCP
	// surface (tools/list, tools/call, initialize).
	Addr string // FALSEFLAG_MCP_ADDR, default ":8091"
	// HealthAddr is a separate listener serving /healthz so compose
	// healthchecks don't have to speak MCP.
	HealthAddr string // FALSEFLAG_MCP_HEALTH_ADDR, default ":8092"
	// APIBaseURL is the upstream FalseFlag Connect endpoint the MCP
	// server proxies tool calls to. Defaults at the local API on the
	// slice-3 RPC port.
	APIBaseURL string // FALSEFLAG_API_RPC_ADDR, default "http://localhost:8090"
	// Actor is the X-Actor header value attached to every API
	// request. Demo-only attribution.
	Actor string // FALSEFLAG_MCP_ACTOR, default "mcp/falseflag-mcp"
}

// LoadAPI reads APIConfig from the environment.
func LoadAPI() (APIConfig, error) {
	return APIConfig{
		Addr:        getEnv("FALSEFLAG_API_ADDR", ":8080"),
		RPCAddr:     getEnv("FALSEFLAG_API_RPC_ADDR", ":8090"),
		DatabaseURL: os.Getenv("FALSEFLAG_DATABASE_URL"),
	}, nil
}

// LoadProxy reads ProxyConfig from the environment.
func LoadProxy() (ProxyConfig, error) {
	return ProxyConfig{
		Addr:        getEnv("FALSEFLAG_PROXY_ADDR", ":8081"),
		APIBaseURL:  getEnv("FALSEFLAG_API_BASE_URL", "http://localhost:8080"),
		ProjectSlug: os.Getenv("FALSEFLAG_PROXY_PROJECT_SLUG"),
	}, nil
}

// LoadOperator reads OperatorConfig from the environment.
func LoadOperator() (OperatorConfig, error) {
	return OperatorConfig{
		MetricsAddr:     getEnv("FALSEFLAG_OPERATOR_METRICS_ADDR", ":8082"),
		HealthProbeAddr: getEnv("FALSEFLAG_OPERATOR_HEALTH_ADDR", ":8083"),
		LeaderElect:     getEnvBool("FALSEFLAG_OPERATOR_LEADER_ELECT", false),
		APIBaseURL:      getEnv("FALSEFLAG_API_BASE_URL", "http://falseflag-api.default.svc.cluster.local:8090"),
		Actor:           getEnv("FALSEFLAG_OPERATOR_ACTOR", "controller/falseflag-operator"),
	}, nil
}

// LoadMCP reads MCPConfig from the environment.
func LoadMCP() (MCPConfig, error) {
	return MCPConfig{
		Addr:       getEnv("FALSEFLAG_MCP_ADDR", ":8091"),
		HealthAddr: getEnv("FALSEFLAG_MCP_HEALTH_ADDR", ":8092"),
		APIBaseURL: getEnv("FALSEFLAG_API_RPC_ADDR", "http://localhost:8090"),
		Actor:      getEnv("FALSEFLAG_MCP_ACTOR", "mcp/falseflag-mcp"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}
