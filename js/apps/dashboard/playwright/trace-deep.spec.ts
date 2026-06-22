import { expect, test } from "@playwright/test";

import { settle } from "./helpers";

// Deep coverage of the trace explorer against many flag/context
// combinations. Each spec is independent and shard-friendly.

const TRACE_CASES: ReadonlyArray<{
  slug: string;
  key: string;
  ctx: Record<string, unknown>;
  label: string;
}> = [
  {
    slug: "acme-web",
    key: "checkout-redesign",
    ctx: { user: { id: "u-1", plan: "pro" } },
    label: "checkout-redesign pro user",
  },
  {
    slug: "acme-web",
    key: "checkout-redesign",
    ctx: { user: { id: "u-2", plan: "free" } },
    label: "checkout-redesign free user",
  },
  {
    slug: "acme-web",
    key: "max-cart-items",
    ctx: { user: { plan: "enterprise" } },
    label: "max-cart-items enterprise",
  },
  {
    slug: "acme-web",
    key: "checkout-banner-text",
    ctx: { user: { country: "US" } },
    label: "checkout-banner-text US",
  },
  {
    slug: "acme-web",
    key: "proxy-readiness-bool",
    ctx: { user: { id: "u-rollout-1" } },
    label: "proxy-readiness-bool rollout-1",
  },
  {
    slug: "acme-web",
    key: "proxy-readiness-bool",
    ctx: { user: { id: "u-rollout-2" } },
    label: "proxy-readiness-bool rollout-2",
  },
  {
    slug: "acme-mobile",
    key: "force-update-required",
    ctx: { user: { app_version: "0.9.0", country: "DE" } },
    label: "force-update-required old version",
  },
  {
    slug: "acme-mobile",
    key: "force-update-required",
    ctx: { user: { app_version: "2.0.0", country: "US" } },
    label: "force-update-required new version",
  },
  {
    slug: "acme-mobile",
    key: "push-notification-cadence",
    ctx: { user: { engagement: "high" } },
    label: "push-notification-cadence high",
  },
  {
    slug: "acme-internal",
    key: "feature-x",
    ctx: { user: { team: "platform" } },
    label: "feature-x platform team",
  },
];

for (const tc of TRACE_CASES) {
  test(`trace: ${tc.label}`, async ({ page }) => {
    await page.goto(`/projects/${tc.slug}/flags/${tc.key}/trace`);
    await expect(page.getByTestId("eval-form")).toBeVisible();
    await settle(page, 400);
    await page.getByTestId("context-input").fill(JSON.stringify(tc.ctx));
    await settle(page, 300);
    await page.getByTestId("evaluate-btn").click();
    await expect(page.getByTestId("decision")).toBeVisible();
    await settle(page, 600);
    // Trace tree should also render.
    await expect(page.getByTestId("trace")).toBeVisible();
  });
}

test("trace explorer remembers its last input on submit", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign/trace");
  await page.getByTestId("context-input").fill(`{"user":{"plan":"trial"}}`);
  await settle(page, 400);
  await page.getByTestId("evaluate-btn").click();
  await expect(page.getByTestId("decision")).toBeVisible();
  await settle(page, 500);
  await expect(page.getByTestId("context-input")).toHaveValue(/trial/);
});

test("trace explorer re-evaluates after toggling context", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign/trace");
  for (const plan of ["free", "pro", "enterprise"]) {
    await page
      .getByTestId("context-input")
      .fill(JSON.stringify({ user: { plan } }));
    await settle(page, 300);
    await page.getByTestId("evaluate-btn").click();
    await expect(page.getByTestId("decision")).toBeVisible();
    await settle(page, 500);
  }
});
