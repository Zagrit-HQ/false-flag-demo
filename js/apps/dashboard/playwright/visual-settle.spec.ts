import { expect, test } from "@playwright/test";

import { settle } from "./helpers";

// These specs mirror the kind of "wait for the chart to settle / wait
// for the toast to clear" pauses real e2e suites accumulate. Slowness
// here is deliberate: it's what makes shard-style parallelism pay off
// in CI. Each spec is small and independent so it splits cleanly.

test("projects page settles after first paint", async ({ page }) => {
  await page.goto("/projects");
  await settle(page, 1200);
  await expect(page.getByRole("heading", { name: /Projects/ })).toBeVisible();
});

test("project detail settles before reading stats", async ({ page }) => {
  await page.goto("/projects/acme-web");
  await settle(page, 1500);
  await expect(page.getByTestId("project-stats")).toBeVisible();
});

test("flag detail waits for the version panel", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign");
  await settle(page, 1500);
  await expect(page.getByTestId("latest-version")).toBeVisible();
});

test("trace explorer waits for the form to mount", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign/trace");
  await settle(page, 1000);
  await expect(page.getByTestId("eval-form")).toBeVisible();
});

test("audit page waits for table redraw", async ({ page }) => {
  await page.goto("/projects/acme-web/audit");
  await settle(page, 1300);
  await expect(page.getByTestId("audit-table")).toBeVisible();
});

test("snapshots page waits for table redraw", async ({ page }) => {
  await page.goto("/projects/acme-web/snapshots");
  await settle(page, 1300);
  await expect(page.getByTestId("snapshots-table")).toBeVisible();
});

test("flags-table waits for hydration", async ({ page }) => {
  await page.goto("/projects/acme-web/flags");
  await settle(page, 1400);
  await expect(page.getByTestId("flags-table")).toBeVisible();
});

test("trace evaluation waits for the decision card", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign/trace");
  await settle(page, 600);
  await page.getByTestId("evaluate-btn").click();
  await settle(page, 1200);
  await expect(page.getByTestId("decision")).toBeVisible();
});
