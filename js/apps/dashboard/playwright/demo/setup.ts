type DemoProject = {
  slug: string;
  displayName: string;
  strategy: "json" | "cel" | "typescript";
  environments: Array<{ slug: string; name: string }>;
  flags: DemoFlag[];
};

type DemoFlag = {
  key: string;
  name: string;
  description: string;
  valueType: "boolean" | "string" | "number" | "object";
  defaultValue: unknown;
  strategy: "json" | "cel" | "typescript";
  source: Record<string, unknown>;
  sourceText: string;
};

const API_BASE = process.env.FALSEFLAG_API_BASE_URL ?? "http://localhost:8080";

const demoProjects: DemoProject[] = [
  {
    slug: "acme-web",
    displayName: "Acme Web App",
    strategy: "json",
    environments: [
      { slug: "production", name: "Production" },
      { slug: "staging", name: "Staging" },
      { slug: "dev", name: "Development" },
    ],
    flags: [
      {
        key: "checkout-redesign",
        name: "Checkout Redesign",
        description: "New checkout flow for paying users.",
        valueType: "boolean",
        defaultValue: false,
        strategy: "json",
        source: {
          value_type: "boolean",
          default: false,
          rules: [
            {
              id: "paid",
              when: {
                kind: "in",
                attr: "user.plan",
                values: ["pro", "enterprise"],
              },
              value: true,
            },
          ],
        },
        sourceText: `{
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
        key: "max-cart-items",
        name: "Max Cart Items",
        description: "Cart-size cap.",
        valueType: "number",
        defaultValue: 25,
        strategy: "json",
        source: {
          value_type: "number",
          default: 25,
          rules: [
            {
              id: "vip",
              when: { kind: "eq", attr: "user.plan", value: "enterprise" },
              value: 250,
            },
          ],
        },
        sourceText: `{
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
        key: "checkout-banner-text",
        name: "Checkout Banner Text",
        description: "Promo banner copy.",
        valueType: "string",
        defaultValue: "Free shipping on orders over $50",
        strategy: "json",
        source: {
          value_type: "string",
          default: "Free shipping on orders over $50",
          rules: [
            {
              id: "us",
              when: { kind: "eq", attr: "request.country", value: "us" },
              value: "Free shipping on US orders over $25",
            },
          ],
        },
        sourceText: `{
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
        key: "proxy-readiness-bool",
        name: "Proxy Readiness",
        description: "Used by proxy readiness and dashboard E2E checks.",
        valueType: "boolean",
        defaultValue: false,
        strategy: "json",
        source: {
          value_type: "boolean",
          default: false,
          rules: [
            {
              id: "pro-only",
              when: { kind: "eq", attr: "user.plan", value: "pro" },
              value: true,
            },
          ],
        },
        sourceText: `{
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
    ],
  },
  {
    slug: "acme-mobile",
    displayName: "Acme Mobile App",
    strategy: "cel",
    environments: [
      { slug: "production", name: "Production" },
      { slug: "staging", name: "Staging" },
    ],
    flags: [
      {
        key: "force-update-required",
        name: "Force Update Required",
        description: "Block clients below min version.",
        valueType: "boolean",
        defaultValue: false,
        strategy: "cel",
        source: {
          value_type: "boolean",
          default: false,
          rules: [
            {
              id: "older-than-3.5",
              when: { kind: "cel", source: "ctx.client.version < '3.5.0'" },
              value: true,
            },
          ],
        },
        sourceText: `{
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
        key: "push-notification-cadence",
        name: "Push Cadence (min)",
        description: "Throttle in minutes.",
        valueType: "number",
        defaultValue: 60,
        strategy: "cel",
        source: {
          value_type: "number",
          default: 60,
          rules: [
            {
              id: "active-users",
              when: { kind: "cel", source: "ctx.user.session_count > 30" },
              value: 15,
            },
          ],
        },
        sourceText: `{
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
    ],
  },
  {
    slug: "acme-internal",
    displayName: "Acme Internal Tools",
    strategy: "typescript",
    environments: [{ slug: "production", name: "Production" }],
    flags: [
      {
        key: "feature-x",
        name: "Feature X",
        description: "Internal beta feature, gated to @acme.internal emails.",
        valueType: "boolean",
        defaultValue: false,
        strategy: "typescript",
        source: {
          value_type: "boolean",
          default: false,
          rules: [
            {
              id: "internal-only",
              when: {
                kind: "matches",
                attr: "user.email",
                pattern: ".+@acme\\.internal$",
              },
              value: true,
            },
          ],
        },
        sourceText: `import { FalseFlag as ff } from "@falseflag/config";

export default ff.flag({
  value_type: "boolean",
  default: false,
  rules: [
    ff.rule(
      "internal-only",
      ff.matches("user.email", ".+@acme\\\\.internal$"),
      true,
    ),
  ],
});
`,
      },
    ],
  },
];

async function api(
  method: string,
  path: string,
  body?: unknown,
): Promise<{ status: number; data: unknown }> {
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers: {
      "content-type": "application/json",
      "x-actor": "playwright/demo",
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text();
  return { status: res.status, data: text ? safeJSON(text) : null };
}

function safeJSON(s: string): unknown {
  try {
    return JSON.parse(s);
  } catch {
    return s;
  }
}

function assertStatus(
  label: string,
  res: { status: number; data: unknown },
  accepted: number[],
): void {
  if (!accepted.includes(res.status)) {
    throw new Error(`${label} HTTP ${res.status}: ${JSON.stringify(res.data)}`);
  }
}

async function seedProject(project: DemoProject): Promise<void> {
  assertStatus(
    `seedDemoDashboard(${project.slug}): create project`,
    await api("POST", "/v1/projects", {
      slug: project.slug,
      display_name: project.displayName,
      config_strategy: project.strategy,
    }),
    [200, 201, 409],
  );

  for (const env of project.environments) {
    assertStatus(
      `seedDemoDashboard(${project.slug}/${env.slug}): create environment`,
      await api("POST", `/v1/projects/${project.slug}/environments`, env),
      [200, 201, 409],
    );
  }

  for (const flag of project.flags) {
    assertStatus(
      `seedDemoDashboard(${project.slug}/${flag.key}): create flag`,
      await api("POST", `/v1/projects/${project.slug}/flags`, {
        key: flag.key,
        name: flag.name,
        description: flag.description,
        value_type: flag.valueType,
        default_value: flag.defaultValue,
      }),
      [200, 201, 409],
    );

    assertStatus(
      `seedDemoDashboard(${project.slug}/${flag.key}): publish flag`,
      await api("PUT", `/v1/projects/${project.slug}/flags/${flag.key}`, {
        strategy: flag.strategy,
        source: flag.source,
        source_text: flag.sourceText,
      }),
      [200, 201],
    );
  }

  assertStatus(
    `seedDemoDashboard(${project.slug}): compile snapshot`,
    await api("POST", `/v1/projects/${project.slug}/snapshots`, {}),
    [200, 201],
  );
}

export async function seedDemoDashboard(): Promise<void> {
  for (const project of demoProjects) {
    await seedProject(project);
  }
}
