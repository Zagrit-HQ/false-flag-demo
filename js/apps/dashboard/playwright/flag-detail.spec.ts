import { expect, test } from "@playwright/test";

import { settle } from "./helpers";

test("flag detail shows the latest version", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign");
  await expect(page.getByTestId("latest-version")).toBeVisible();
  await settle(page, 600);
});

test("flag detail shows the trace CTA", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign");
  await expect(page.getByTestId("trace-cta")).toBeVisible();
  await settle(page, 500);
});

test("flag detail shows the versions list", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/max-cart-items");
  await settle(page, 600);
  await expect(page.getByTestId("versions")).toBeVisible();
});

test("flag detail breadcrumb shows project and key", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-banner-text");
  await settle(page, 500);
  const nav = page.locator("nav").first();
  await expect(nav).toContainText("acme-web");
  await expect(nav).toContainText("checkout-banner-text");
});

test("cel flag detail loads on acme-mobile", async ({ page }) => {
  await page.goto("/projects/acme-mobile/flags/force-update-required");
  await expect(page.getByTestId("latest-version")).toBeVisible();
  await settle(page, 800);
});
