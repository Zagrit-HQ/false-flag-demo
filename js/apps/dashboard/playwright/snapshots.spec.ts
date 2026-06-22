import { expect, test } from "@playwright/test";

import { settle } from "./helpers";

test("snapshots table is visible", async ({ page }) => {
  await page.goto("/projects/acme-web/snapshots");
  await expect(page.getByTestId("snapshots-table")).toBeVisible();
  await settle(page, 700);
});

test("snapshots table shows at least one version", async ({ page }) => {
  await page.goto("/projects/acme-web/snapshots");
  await settle(page, 600);
  // Each seeded project gets a published snapshot, so the table
  // should contain a numeric version.
  await expect(page.getByTestId("snapshots-table")).toContainText(/\d+/);
});

test("snapshots reachable for each seeded project", async ({ page }) => {
  for (const slug of ["acme-web", "acme-mobile", "acme-internal"] as const) {
    await page.goto(`/projects/${slug}/snapshots`);
    await settle(page, 500);
    await expect(page.getByTestId("snapshots-table")).toBeVisible();
  }
});

test("snapshots breadcrumb shows project slug", async ({ page }) => {
  await page.goto("/projects/acme-mobile/snapshots");
  await settle(page, 500);
  await expect(page.locator("nav").first()).toContainText("acme-mobile");
});
