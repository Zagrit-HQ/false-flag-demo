package main

type demoProject struct {
	Slug         string
	DisplayName  string
	Strategy     string // json | cel | typescript — the project's primary mode
	Environments []demoEnvironment
	Flags        []demoFlag
}

type demoEnvironment struct {
	Slug        string
	DisplayName string
}

type demoFlag struct {
	Key         string
	Name        string
	Description string
	ValueType   string         // boolean | string | number | object
	Strategy    string         // json | cel | typescript
	IR          map[string]any // already-IR-shaped (the API accepts the IR for any strategy)
	// SourceText is the raw author-authored source the dashboard renders
	// with Shiki. For json/cel flags this is the IR serialized as
	// canonical JSON. For typescript flags it is a real `ff.flag(...)`
	// block compiled server-side via esbuild + goja.
	SourceText string
}

// demoProjects is the seed dataset. Three projects with realistic
// names and a mix of strategies. The proxy compose service is
// configured to scope to acme-web by default.
var demoProjects = []demoProject{
	{
		Slug:        "acme-web",
		DisplayName: "Acme Web App",
		Strategy:    "json",
		Environments: []demoEnvironment{
			{"production", "Production"},
			{"staging", "Staging"},
			{"dev", "Development"},
		},
		Flags: []demoFlag{
			{
				Key: "checkout-redesign", Name: "Checkout Redesign", Description: "New checkout flow for paying users.",
				ValueType: "boolean", Strategy: "json",
				IR: map[string]any{
					"value_type": "boolean", "default": false,
					"rules": []any{
						map[string]any{
							"id":    "paid",
							"when":  map[string]any{"kind": "in", "attr": "user.plan", "values": []any{"pro", "enterprise"}},
							"value": true,
						},
					},
				},
				SourceText: `{
  "value_type": "boolean",
  "default": false,
  "rules": [
    {
      "id": "paid",
      "when": { "kind": "in", "attr": "user.plan", "values": ["pro", "enterprise"] },
      "value": true
    }
  ]
}`,
			},
			{
				Key: "max-cart-items", Name: "Max Cart Items", Description: "Cart-size cap.",
				ValueType: "number", Strategy: "json",
				IR: map[string]any{
					"value_type": "number", "default": 25,
					"rules": []any{
						map[string]any{
							"id":    "vip",
							"when":  map[string]any{"kind": "eq", "attr": "user.plan", "value": "enterprise"},
							"value": 250,
						},
					},
				},
				SourceText: `{
  "value_type": "number",
  "default": 25,
  "rules": [
    {
      "id": "vip",
      "when": { "kind": "eq", "attr": "user.plan", "value": "enterprise" },
      "value": 250
    }
  ]
}`,
			},
			{
				Key: "checkout-banner-text", Name: "Checkout Banner Text", Description: "Promo banner copy.",
				ValueType: "string", Strategy: "json",
				IR: map[string]any{
					"value_type": "string", "default": "Free shipping on orders over $50",
					"rules": []any{
						map[string]any{
							"id":    "us",
							"when":  map[string]any{"kind": "eq", "attr": "request.country", "value": "us"},
							"value": "Free shipping on US orders over $25",
						},
					},
				},
				SourceText: `{
  "value_type": "string",
  "default": "Free shipping on orders over $50",
  "rules": [
    {
      "id": "us",
      "when": { "kind": "eq", "attr": "request.country", "value": "us" },
      "value": "Free shipping on US orders over $25"
    }
  ]
}`,
			},
			{
				Key: "proxy-readiness-bool", Name: "Proxy Readiness", Description: "Used by proxy readiness and dashboard E2E checks.",
				ValueType: "boolean", Strategy: "json",
				IR: map[string]any{
					"value_type": "boolean", "default": false,
					"rules": []any{
						map[string]any{
							"id":    "pro-only",
							"when":  map[string]any{"kind": "eq", "attr": "user.plan", "value": "pro"},
							"value": true,
						},
					},
				},
				SourceText: `{
  "value_type": "boolean",
  "default": false,
  "rules": [
    {
      "id": "pro-only",
      "when": { "kind": "eq", "attr": "user.plan", "value": "pro" },
      "value": true
    }
  ]
}`,
			},
		},
	},
	{
		Slug:        "acme-mobile",
		DisplayName: "Acme Mobile App",
		Strategy:    "cel",
		Environments: []demoEnvironment{
			{"production", "Production"},
			{"staging", "Staging"},
		},
		Flags: []demoFlag{
			{
				Key: "force-update-required", Name: "Force Update Required", Description: "Block clients below min version.",
				ValueType: "boolean", Strategy: "cel",
				IR: map[string]any{
					"value_type": "boolean", "default": false,
					"rules": []any{
						map[string]any{
							"id":    "older-than-3.5",
							"when":  map[string]any{"kind": "cel", "source": "ctx.client.version < '3.5.0'"},
							"value": true,
						},
					},
				},
				SourceText: `{
  "value_type": "boolean",
  "default": false,
  "rules": [
    {
      "id": "older-than-3.5",
      "when": { "kind": "cel", "source": "ctx.client.version < '3.5.0'" },
      "value": true
    }
  ]
}`,
			},
			{
				Key: "push-notification-cadence", Name: "Push Cadence (min)", Description: "Throttle in minutes.",
				ValueType: "number", Strategy: "cel",
				IR: map[string]any{
					"value_type": "number", "default": 60,
					"rules": []any{
						map[string]any{
							"id":    "active-users",
							"when":  map[string]any{"kind": "cel", "source": "ctx.user.session_count > 30"},
							"value": 15,
						},
					},
				},
				SourceText: `{
  "value_type": "number",
  "default": 60,
  "rules": [
    {
      "id": "active-users",
      "when": { "kind": "cel", "source": "ctx.user.session_count > 30" },
      "value": 15
    }
  ]
}`,
			},
		},
	},
	{
		Slug:        "acme-internal",
		DisplayName: "Acme Internal Tools",
		Strategy:    "typescript",
		Environments: []demoEnvironment{
			{"production", "Production"},
		},
		Flags: []demoFlag{
			{
				Key: "feature-x", Name: "Feature X", Description: "Internal beta feature, gated to @acme.internal emails.",
				ValueType: "boolean", Strategy: "typescript",
				IR: map[string]any{
					"value_type": "boolean", "default": false,
					"rules": []any{
						map[string]any{
							"id":    "internal-only",
							"when":  map[string]any{"kind": "matches", "attr": "user.email", "pattern": ".+@acme\\.internal$"},
							"value": true,
						},
					},
				},
				SourceText: `import { FalseFlag as ff } from "@falseflag/config";

export default ff.flag({
  value_type: "boolean",
  default: false,
  rules: [
    ff.rule(
      "internal-only",
      ff.matches("user.email", ".+@acme\\.internal$"),
      true,
    ),
  ],
});
`,
			},
			{
				Key: "dark-mode-default", Name: "Dark Mode Default", Description: "Roll dark mode out to 50% of pro users.",
				ValueType: "boolean", Strategy: "typescript",
				IR: map[string]any{
					"value_type": "boolean", "default": false,
					"rules": []any{
						map[string]any{
							"id": "pro-rollout-50",
							"when": map[string]any{
								"kind": "all",
								"of": []any{
									map[string]any{"kind": "eq", "attr": "user.plan", "value": "pro"},
									map[string]any{"kind": "rollout", "attr": "user.id", "salt": "dark-mode-default-v1", "percent": 50},
								},
							},
							"value": true,
						},
					},
				},
				SourceText: `import { FalseFlag as ff } from "@falseflag/config";

export default ff.flag({
  value_type: "boolean",
  default: false,
  rules: [
    ff.rule(
      "pro-rollout-50",
      ff.all(
        ff.eq("user.plan", "pro"),
        ff.rollout("user.id", "dark-mode-default-v1", 50),
      ),
      true,
    ),
  ],
});
`,
			},
		},
	},
}
