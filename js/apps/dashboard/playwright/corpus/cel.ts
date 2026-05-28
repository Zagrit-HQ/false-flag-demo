// 50 CEL edit-roundtrip fixtures. Each fixture's testSource is an IR
// JSON object the dashboard renders via Shiki's `javascript` lang
// (the slice 5 strategy-to-language map). The `when.kind: "cel"`
// predicate's `source` field carries the actual CEL expression we
// want to round-trip through the edit pipeline.

import type { CorpusFixture } from "./types";

interface CelCase {
  /** kebab-case slug, becomes the unique flag/project suffix. */
  slug: string;
  /** CEL expression the rule's `when.source` field will carry. */
  expr: string;
}

// Curated CEL subset: idents, dot access, comparisons, &&, ||, !,
// `in` with list literals, boolean attrs, string/number literals.
// Matches what cel-lite.ts in @falseflag/sdk supports.
const CASES: CelCase[] = [
  // -- equality / inequality --------------------------------------
  { slug: "eq-plan-pro", expr: "ctx.user.plan == 'pro'" },
  { slug: "eq-plan-enterprise", expr: "ctx.user.plan == 'enterprise'" },
  { slug: "eq-plan-team", expr: "ctx.user.plan == 'team'" },
  { slug: "eq-country-us", expr: "ctx.user.country == 'US'" },
  { slug: "eq-country-uk", expr: "ctx.user.country == 'UK'" },
  { slug: "eq-country-ca", expr: "ctx.user.country == 'CA'" },
  { slug: "eq-region-eu", expr: "ctx.request.region == 'eu-west-1'" },
  { slug: "eq-region-apac", expr: "ctx.request.region == 'ap-southeast-2'" },
  { slug: "neq-plan-free", expr: "ctx.user.plan != 'free'" },
  { slug: "neq-country-cn", expr: "ctx.user.country != 'CN'" },

  // -- numeric comparisons ----------------------------------------
  { slug: "lt-age-18", expr: "ctx.user.age < 18" },
  { slug: "lt-sessions-5", expr: "ctx.user.session_count < 5" },
  { slug: "lt-cart-size-3", expr: "ctx.cart.size < 3" },
  { slug: "lte-age-21", expr: "ctx.user.age <= 21" },
  { slug: "lte-priority-3", expr: "ctx.task.priority <= 3" },
  { slug: "lte-retries-0", expr: "ctx.request.retries <= 0" },
  { slug: "gt-age-65", expr: "ctx.user.age > 65" },
  { slug: "gt-sessions-50", expr: "ctx.user.session_count > 50" },
  { slug: "gt-cart-total-1000", expr: "ctx.cart.total > 1000" },
  { slug: "gte-age-18", expr: "ctx.user.age >= 18" },
  { slug: "gte-purchases-3", expr: "ctx.user.purchases >= 3" },
  { slug: "gte-rep-100", expr: "ctx.user.reputation >= 100" },

  // -- logical AND ------------------------------------------------
  {
    slug: "and-plan-and-country",
    expr: "ctx.user.plan == 'pro' && ctx.user.country == 'US'",
  },
  {
    slug: "and-plan-and-age",
    expr: "ctx.user.plan == 'pro' && ctx.user.age >= 18",
  },
  {
    slug: "and-three-clauses",
    expr: "ctx.user.plan == 'pro' && ctx.user.country == 'US' && ctx.user.age >= 18",
  },
  {
    slug: "and-four-clauses",
    expr: "ctx.user.plan == 'pro' && ctx.user.country == 'US' && ctx.user.age >= 18 && ctx.user.verified",
  },

  // -- logical OR -------------------------------------------------
  {
    slug: "or-pro-or-enterprise",
    expr: "ctx.user.plan == 'pro' || ctx.user.plan == 'enterprise'",
  },
  {
    slug: "or-us-or-uk-or-ca",
    expr: "ctx.user.country == 'US' || ctx.user.country == 'UK' || ctx.user.country == 'CA'",
  },
  {
    slug: "or-young-or-senior",
    expr: "ctx.user.age < 18 || ctx.user.age > 65",
  },

  // -- logical NOT ------------------------------------------------
  { slug: "not-free-plan", expr: "!(ctx.user.plan == 'free')" },
  { slug: "not-young", expr: "!(ctx.user.age < 18)" },
  { slug: "not-bool-attr", expr: "!ctx.user.opted_in" },

  // -- mixed / nested compositions --------------------------------
  {
    slug: "mixed-and-not",
    expr: "ctx.user.plan == 'pro' && !(ctx.user.country == 'CN')",
  },
  {
    slug: "mixed-or-and",
    expr: "(ctx.user.plan == 'pro' || ctx.user.plan == 'enterprise') && ctx.user.age >= 18",
  },
  {
    slug: "double-not",
    expr: "!(!(ctx.user.plan == 'pro'))",
  },
  {
    slug: "session-mid-range",
    expr: "ctx.user.session_count >= 10 && ctx.user.session_count < 50",
  },

  // -- `in` over list literals ------------------------------------
  {
    slug: "in-paid-plans",
    expr: "ctx.user.plan in ['pro','enterprise','team']",
  },
  {
    slug: "in-tier-1-countries",
    expr: "ctx.user.country in ['US','UK','CA','AU','NZ']",
  },
  {
    slug: "in-experiment-buckets",
    expr: "ctx.experiment.bucket in [10,20,30]",
  },
  { slug: "in-single", expr: "ctx.user.tags in ['beta']" },

  // -- version-string compare -------------------------------------
  {
    slug: "version-lt-3-5",
    expr: "ctx.client.version < '3.5.0'",
  },
  {
    slug: "version-gte-4-0",
    expr: "ctx.client.version >= '4.0.0'",
  },

  // -- boolean attrs (CEL requires bool-typed expression; bare
  //    identifiers are `dyn` and fail typing — use an explicit
  //    comparison to ground the type as bool) -------------------
  { slug: "bool-opted-in", expr: "ctx.user.opted_in == true" },
  { slug: "bool-experiment-active", expr: "ctx.experiment.active == true" },
  {
    slug: "bool-and-comparison",
    expr: "ctx.experiment.active && ctx.user.bucket < 50",
  },

  // -- device / browser / os --------------------------------------
  { slug: "device-mobile", expr: "ctx.device.type == 'mobile'" },
  { slug: "device-tablet", expr: "ctx.device.type == 'tablet'" },
  { slug: "browser-chrome", expr: "ctx.browser.name == 'chrome'" },
  { slug: "os-windows", expr: "ctx.os.name == 'Windows'" },

  // -- email / segment / subscription -----------------------------
  {
    slug: "internal-email",
    expr: "ctx.user.email == 'admin@acme.internal'",
  },
];

// Sanity check — we want exactly 50.
if (CASES.length !== 50) {
  throw new Error(`CEL corpus must be 50 fixtures, got ${CASES.length}`);
}

/** Wraps a CEL expression into a complete IR JSON object the
 *  dashboard accepts under {strategy:"cel", source: <this>}. */
function celIR(expr: string) {
  return {
    value_type: "boolean",
    default: false,
    rules: [
      {
        id: "r1",
        when: { kind: "cel", source: expr },
        value: true,
      },
    ],
  };
}

export const CEL_CORPUS: CorpusFixture[] = CASES.map((c) => {
  const ir = celIR(c.expr);
  const setupIR = celIR("false");
  return {
    id: `cel-${c.slug}`,
    strategy: "cel",
    valueType: "boolean",
    defaultValue: false,
    setupSource: JSON.stringify(setupIR, null, 2),
    setupIR,
    testSource: JSON.stringify(ir, null, 2),
    assertSubstring: c.expr,
  };
});
