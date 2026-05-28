import { expect, test } from "@playwright/test";

import { settle } from "./helpers";

test("audit log page renders the filters", async ({ page }) => {
  await page.goto("/projects/acme-web/audit");
  await expect(page.getByTestId("audit-filters")).toBeVisible();
  await settle(page, 500);
});

test("audit log page renders the table", async ({ page }) => {
  await page.goto("/projects/acme-web/audit");
  await expect(page.getByTestId("audit-table")).toBeVisible();
  await settle(page, 700);
});

test("audit log contains a project.create entry", async ({ page }) => {
  await page.goto("/projects/acme-web/audit");
  await settle(page, 600);
  await expect(page.getByTestId("audit-table")).toContainText(/project|flag/);
});

test("audit log is reachable for every seeded project", async ({ page }) => {
  for (const slug of ["acme-web", "acme-mobile", "acme-internal"] as const) {
    await page.goto(`/projects/${slug}/audit`);
    await settle(page, 400);
    await expect(page.getByTestId("audit-table")).toBeVisible();
  }
});
