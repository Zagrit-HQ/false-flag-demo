// 50 TypeScript edit-roundtrip fixtures. Each fixture's testSource is
// a real ff.flag(...) block; the dashboard's `typescript` Shiki
// highlighter renders it on the view route after save. The server
// compiles every fixture via esbuild + goja (slice 8 phase 2).
//
// Asserting per-fixture: a unique rule id appears in the rendered
// source-code element. Rule ids are unique-per-fixture and avoid
// collisions with the surrounding boilerplate.

import type { CorpusFixture } from "./types";

interface TsCase {
  slug: string;
  /** Distinct value type. */
  valueType: "boolean" | "string" | "number";
  /** Default value for the flag. */
  defaultValue: unknown;
  /** Rule id we will assert appears in the rendered source. */
  ruleId: string;
  /** TS body for the rule (the `ff.rule(...)` call). */
  ruleBody: string;
}

// 50 hand-curated cases covering every builder + value-type mix.
const CASES: TsCase[] = [
  // ---- ff.eq -----------------------------------------------------
  {
    slug: "eq-bool-plan-pro",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-eq-001",
    ruleBody: 'ff.eq("user.plan", "pro")',
  },
  {
    slug: "eq-string-country-us",
    valueType: "string",
    defaultValue: "basic",
    ruleId: "ts-eq-002",
    ruleBody: 'ff.eq("user.country", "US")',
  },
  {
    slug: "eq-number-age-30",
    valueType: "number",
    defaultValue: 25,
    ruleId: "ts-eq-003",
    ruleBody: 'ff.eq("user.age", 30)',
  },
  {
    slug: "eq-bool-segment-vip",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-eq-004",
    ruleBody: 'ff.eq("user.segment", "vip")',
  },

  // ---- ff.neq ----------------------------------------------------
  {
    slug: "neq-bool-plan-free",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-neq-001",
    ruleBody: 'ff.neq("user.plan", "free")',
  },
  {
    slug: "neq-string-country-cn",
    valueType: "string",
    defaultValue: "default",
    ruleId: "ts-neq-002",
    ruleBody: 'ff.neq("user.country", "CN")',
  },
  {
    slug: "neq-number-zero",
    valueType: "number",
    defaultValue: 0,
    ruleId: "ts-neq-003",
    ruleBody: 'ff.neq("metric.errors", 0)',
  },

  // ---- ff.in -----------------------------------------------------
  {
    slug: "in-paid-plans",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-in-001",
    ruleBody: 'ff.in("user.plan", ["pro", "enterprise", "team"])',
  },
  {
    slug: "in-string-tiers",
    valueType: "string",
    defaultValue: "basic",
    ruleId: "ts-in-002",
    ruleBody: 'ff.in("user.tier", ["gold", "platinum", "diamond"])',
  },
  {
    slug: "in-numeric-buckets",
    valueType: "number",
    defaultValue: 0,
    ruleId: "ts-in-003",
    ruleBody: 'ff.in("experiment.bucket", [10, 20, 30])',
  },
  {
    slug: "in-locales",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-in-004",
    ruleBody: 'ff.in("request.locale", ["en-US", "en-GB", "en-CA"])',
  },

  // ---- ff.gt / gte / lt / lte -----------------------------------
  {
    slug: "gt-age-65",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-gt-001",
    ruleBody: 'ff.gt("user.age", 65)',
  },
  {
    slug: "gt-cart-1000",
    valueType: "string",
    defaultValue: "small",
    ruleId: "ts-gt-002",
    ruleBody: 'ff.gt("cart.total", 1000)',
  },
  {
    slug: "gte-purchases-5",
    valueType: "number",
    defaultValue: 1,
    ruleId: "ts-gte-001",
    ruleBody: 'ff.gte("user.purchases", 5)',
  },
  {
    slug: "gte-rep-100",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-gte-002",
    ruleBody: 'ff.gte("user.reputation", 100)',
  },
  {
    slug: "lt-age-18",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-lt-001",
    ruleBody: 'ff.lt("user.age", 18)',
  },
  {
    slug: "lt-sessions-5",
    valueType: "number",
    defaultValue: 0,
    ruleId: "ts-lt-002",
    ruleBody: 'ff.lt("user.session_count", 5)',
  },
  {
    slug: "lte-priority-3",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-lte-001",
    ruleBody: 'ff.lte("task.priority", 3)',
  },
  {
    slug: "lte-retries-0",
    valueType: "number",
    defaultValue: 0,
    ruleId: "ts-lte-002",
    ruleBody: 'ff.lte("request.retries", 0)',
  },

  // ---- ff.matches -----------------------------------------------
  {
    slug: "matches-internal-email",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-matches-001",
    ruleBody: 'ff.matches("user.email", ".+@acme\\\\.internal$")',
  },
  {
    slug: "matches-vip-id",
    valueType: "string",
    defaultValue: "basic",
    ruleId: "ts-matches-002",
    ruleBody: 'ff.matches("user.id", "^vip-")',
  },
  {
    slug: "matches-staging-host",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-matches-003",
    ruleBody:
      'ff.matches("request.host", ".*\\\\.staging\\\\.example\\\\.com$")',
  },

  // ---- ff.rollout (the demo-critical builder) -------------------
  {
    slug: "rollout-50pct",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-rollout-001",
    ruleBody: 'ff.rollout("user.id", "rollout-50pct-v1", 50)',
  },
  {
    slug: "rollout-10pct",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-rollout-002",
    ruleBody: 'ff.rollout("user.id", "rollout-10pct-v1", 10)',
  },
  {
    slug: "rollout-99pct",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-rollout-003",
    ruleBody: 'ff.rollout("user.id", "rollout-99pct-v1", 99)',
  },
  {
    slug: "rollout-orgid",
    valueType: "string",
    defaultValue: "old",
    ruleId: "ts-rollout-004",
    ruleBody: 'ff.rollout("organization.id", "rollout-orgid-v1", 33)',
  },

  // ---- ff.cel (CEL via the TS DSL) ------------------------------
  {
    slug: "cel-version-lt",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-cel-001",
    ruleBody: "ff.cel(\"ctx.client.version < '3.5.0'\")",
  },
  {
    slug: "cel-multi-clause",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-cel-002",
    ruleBody: "ff.cel(\"ctx.user.plan == 'pro' && ctx.user.country == 'US'\")",
  },
  {
    slug: "cel-in-list",
    valueType: "string",
    defaultValue: "free",
    ruleId: "ts-cel-003",
    ruleBody: "ff.cel(\"ctx.user.plan in ['pro','enterprise']\")",
  },

  // ---- ff.always ------------------------------------------------
  {
    slug: "always-bool",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-always-001",
    ruleBody: "ff.always()",
  },
  {
    slug: "always-string",
    valueType: "string",
    defaultValue: "default",
    ruleId: "ts-always-002",
    ruleBody: "ff.always()",
  },

  // ---- ff.not ---------------------------------------------------
  {
    slug: "not-eq-plan-free",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-not-001",
    ruleBody: 'ff.not(ff.eq("user.plan", "free"))',
  },
  {
    slug: "not-in-banned",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-not-002",
    ruleBody: 'ff.not(ff.in("user.country", ["CN", "RU", "KP"]))',
  },

  // ---- ff.all (composition) -------------------------------------
  {
    slug: "all-pro-and-us",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-all-001",
    ruleBody: 'ff.all(ff.eq("user.plan", "pro"), ff.eq("user.country", "US"))',
  },
  {
    slug: "all-pro-and-adult-and-rollout",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-all-002",
    ruleBody:
      'ff.all(ff.eq("user.plan", "pro"), ff.gte("user.age", 18), ff.rollout("user.id", "all-002-v1", 50))',
  },
  {
    slug: "all-numeric-range",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-all-003",
    ruleBody:
      'ff.all(ff.gte("user.session_count", 10), ff.lt("user.session_count", 50))',
  },

  // ---- ff.any (composition) -------------------------------------
  {
    slug: "any-paid-plans",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-any-001",
    ruleBody:
      'ff.any(ff.eq("user.plan", "pro"), ff.eq("user.plan", "enterprise"), ff.eq("user.plan", "team"))',
  },
  {
    slug: "any-vip-tiers",
    valueType: "string",
    defaultValue: "basic",
    ruleId: "ts-any-002",
    ruleBody:
      'ff.any(ff.eq("user.tier", "gold"), ff.eq("user.tier", "platinum"))',
  },
  {
    slug: "any-or-internal",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-any-003",
    ruleBody:
      'ff.any(ff.eq("user.plan", "enterprise"), ff.matches("user.email", "@acme\\\\.internal$"))',
  },

  // ---- nested combinators ---------------------------------------
  {
    slug: "nested-any-of-all",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-nested-001",
    ruleBody:
      'ff.any(ff.all(ff.eq("user.plan", "pro"), ff.eq("user.country", "US")), ff.all(ff.eq("user.plan", "enterprise"), ff.eq("user.country", "UK")))',
  },
  {
    slug: "nested-not-of-any",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-nested-002",
    ruleBody:
      'ff.not(ff.any(ff.eq("user.plan", "free"), ff.eq("user.plan", "trial")))',
  },
  {
    slug: "nested-all-with-rollout",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-nested-003",
    ruleBody:
      'ff.all(ff.in("user.country", ["US","UK","CA"]), ff.rollout("user.id", "nested-003-v1", 25))',
  },

  // ---- attribute variety ----------------------------------------
  {
    slug: "attr-device-type",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-attr-001",
    ruleBody: 'ff.eq("device.type", "mobile")',
  },
  {
    slug: "attr-browser-name",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-attr-002",
    ruleBody: 'ff.eq("browser.name", "chrome")',
  },
  {
    slug: "attr-os-name",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-attr-003",
    ruleBody: 'ff.eq("os.name", "Windows")',
  },
  {
    slug: "attr-request-locale",
    valueType: "string",
    defaultValue: "en-US",
    ruleId: "ts-attr-004",
    ruleBody: 'ff.eq("request.locale", "en-GB")',
  },

  // ---- subscription / experiment --------------------------------
  {
    slug: "subscription-active",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-sub-001",
    ruleBody:
      'ff.all(ff.eq("subscription.status", "active"), ff.neq("subscription.tier", "free"))',
  },
  {
    slug: "experiment-bucket",
    valueType: "boolean",
    defaultValue: false,
    ruleId: "ts-exp-001",
    ruleBody:
      'ff.all(ff.eq("experiment.active", true), ff.lt("user.bucket", 50))',
  },

  // ---- mix of value types in one fixture -------------------------
  {
    slug: "string-result-color",
    valueType: "string",
    defaultValue: "blue",
    ruleId: "ts-result-001",
    ruleBody: 'ff.eq("user.theme_pref", "dark")',
  },
  {
    slug: "number-result-throttle",
    valueType: "number",
    defaultValue: 60,
    ruleId: "ts-result-002",
    ruleBody: 'ff.gt("user.session_count", 30)',
  },
];

if (CASES.length !== 50) {
  throw new Error(`TS corpus must be 50 fixtures, got ${CASES.length}`);
}

function ruleResultValue(valueType: "boolean" | "string" | "number"): string {
  if (valueType === "boolean") return "true";
  if (valueType === "string") return '"matched"';
  return "1";
}

/** Build a complete TS source file embedding the rule predicate. */
function tsSource(c: TsCase): string {
  return [
    'import { FalseFlag as ff } from "@falseflag/config";',
    "",
    "export default ff.flag({",
    `  value_type: "${c.valueType}",`,
    `  default: ${JSON.stringify(c.defaultValue)},`,
    "  rules: [",
    "    ff.rule(",
    `      "${c.ruleId}",`,
    `      ${c.ruleBody},`,
    `      ${ruleResultValue(c.valueType)},`,
    "    ),",
    "  ],",
    "});",
    "",
  ].join("\n");
}

/** Minimal TS source used at setup time to fix the flag's strategy.
 *  Identical shape per value-type so Monaco loads instantly. */
function tsSetupSource(valueType: "boolean" | "string" | "number"): string {
  return [
    'import { FalseFlag as ff } from "@falseflag/config";',
    "",
    "export default ff.flag({",
    `  value_type: "${valueType}",`,
    `  default: ${JSON.stringify(
      valueType === "boolean" ? false : valueType === "number" ? 0 : "",
    )},`,
    "  rules: [],",
    "});",
    "",
  ].join("\n");
}

function tsSetupIR(c: TsCase) {
  return {
    value_type: c.valueType,
    default:
      c.valueType === "boolean" ? false : c.valueType === "number" ? 0 : "",
    rules: [],
  };
}

export const TS_CORPUS: CorpusFixture[] = CASES.map((c) => ({
  id: `ts-${c.slug}`,
  strategy: "typescript",
  valueType: c.valueType,
  defaultValue: c.defaultValue,
  setupSource: tsSetupSource(c.valueType),
  setupIR: tsSetupIR(c),
  testSource: tsSource(c),
  assertSubstring: c.ruleId,
}));
