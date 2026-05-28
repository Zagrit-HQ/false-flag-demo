import { expect, test } from "@playwright/test";

import { settle } from "./helpers";

// One spec per (flag, context) permutation. Each is an independent
// e2e turn against the live trace endpoint, which makes them
// shard-friendly: 20 specs over 4 shards is ~5 specs per worker.

interface Case {
  slug: string;
  key: string;
  ctx: Record<string, unknown>;
}

const CASES: ReadonlyArray<Case> = [
  {
    slug: "acme-web",
    key: "checkout-redesign",
    ctx: { user: { id: "u-001", plan: "pro" } },
  },
  {
    slug: "acme-web",
    key: "checkout-redesign",
    ctx: { user: { id: "u-002", plan: "pro" } },
  },
  {
    slug: "acme-web",
    key: "checkout-redesign",
    ctx: { user: { id: "u-003", plan: "free" } },
  },
  {
    slug: "acme-web",
    key: "checkout-redesign",
    ctx: { user: { id: "u-004", plan: "enterprise" } },
  },
  {
    slug: "acme-web",
    key: "max-cart-items",
    ctx: { user: { plan: "pro", country: "US" } },
  },
  {
    slug: "acme-web",
    key: "max-cart-items",
    ctx: { user: { plan: "free", country: "CA" } },
  },
  {
    slug: "acme-web",
    key: "max-cart-items",
    ctx: { user: { plan: "enterprise", country: "DE" } },
  },
  { slug: "acme-web", key: "max-cart-items", ctx: { user: { plan: "trial" } } },
  {
    slug: "acme-web",
    key: "checkout-banner-text",
    ctx: { user: { country: "US" } },
  },
  {
    slug: "acme-web",
    key: "checkout-banner-text",
    ctx: { user: { country: "GB" } },
  },
  {
    slug: "acme-web",
    key: "checkout-banner-text",
    ctx: { user: { country: "JP" } },
  },
  {
    slug: "acme-web",
    key: "proxy-smoke-bool",
    ctx: { user: { id: "rollout-a" } },
  },
  {
    slug: "acme-web",
    key: "proxy-smoke-bool",
    ctx: { user: { id: "rollout-b" } },
  },
  {
    slug: "acme-web",
    key: "proxy-smoke-bool",
    ctx: { user: { id: "rollout-c" } },
  },
  {
    slug: "acme-web",
    key: "proxy-smoke-bool",
    ctx: { user: { id: "rollout-d" } },
  },
  {
    slug: "acme-mobile",
    key: "force-update-required",
    ctx: { user: { app_version: "1.0.0" } },
  },
  {
    slug: "acme-mobile",
    key: "force-update-required",
    ctx: { user: { app_version: "1.5.3" } },
  },
  {
    slug: "acme-mobile",
    key: "force-update-required",
    ctx: { user: { app_version: "2.0.0-beta.1" } },
  },
  {
    slug: "acme-mobile",
    key: "push-notification-cadence",
    ctx: { user: { engagement: "low" } },
  },
  {
    slug: "acme-mobile",
    key: "push-notification-cadence",
    ctx: { user: { engagement: "medium" } },
  },
];

for (const [i, c] of CASES.entries()) {
  test(`bulk trace #${i.toString().padStart(2, "0")}: ${c.slug}/${c.key}`, async ({
    page,
  }) => {
    await page.goto(`/projects/${c.slug}/flags/${c.key}/trace`);
    await expect(page.getByTestId("eval-form")).toBeVisible();
    await settle(page, 400);
    await page.getByTestId("context-input").fill(JSON.stringify(c.ctx));
    await settle(page, 300);
    await page.getByTestId("evaluate-btn").click();
    await expect(page.getByTestId("decision")).toBeVisible();
    await settle(page, 700);
  });
}
