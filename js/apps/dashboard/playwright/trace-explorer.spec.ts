import { expect, test } from "@playwright/test";

import { settle } from "./helpers";

test("trace explorer evaluates with the default context", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign/trace");
  await expect(page.getByTestId("eval-form")).toBeVisible();
  await settle(page, 400);
  await page.getByTestId("evaluate-btn").click();
  await expect(page.getByTestId("decision")).toBeVisible();
  await settle(page, 600);
});

test("trace explorer shows the trace tree after evaluating", async ({
  page,
}) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign/trace");
  await page.getByTestId("evaluate-btn").click();
  await expect(page.getByTestId("trace")).toBeVisible();
  await settle(page, 700);
});

test("trace explorer accepts custom JSON context", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign/trace");
  const ctx = JSON.stringify({ user: { plan: "free" } });
  await page.getByTestId("context-input").fill(ctx);
  await settle(page, 500);
  await page.getByTestId("evaluate-btn").click();
  await expect(page.getByTestId("decision")).toBeVisible();
});

test("trace explorer reports an invalid JSON context", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign/trace");
  await page.getByTestId("context-input").fill("{ not json");
  await settle(page, 400);
  await page.getByTestId("evaluate-btn").click();
  await expect(page.getByTestId("error")).toBeVisible();
  await settle(page, 600);
});

test("trace explorer round-trips several contexts", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign/trace");
  const inputs = [
    JSON.stringify({ user: { plan: "pro" } }),
    JSON.stringify({ user: { plan: "free" } }),
    JSON.stringify({ user: { plan: "enterprise" } }),
  ];
  for (const ctx of inputs) {
    await page.getByTestId("context-input").fill(ctx);
    await settle(page, 300);
    await page.getByTestId("evaluate-btn").click();
    await expect(page.getByTestId("decision")).toBeVisible();
  }
});

test("cel flag trace explorer evaluates", async ({ page }) => {
  await page.goto("/projects/acme-mobile/flags/force-update-required/trace");
  await page.getByTestId("context-input").fill(
    JSON.stringify({
      user: { app_version: "1.0.0", country: "US" },
    }),
  );
  await settle(page, 500);
  await page.getByTestId("evaluate-btn").click();
  await expect(page.getByTestId("decision")).toBeVisible();
  await settle(page, 800);
});
