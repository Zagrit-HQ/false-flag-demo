import { expect, test } from "@playwright/test";

import { SEEDED_FLAGS, settle } from "./helpers";

// Cross-project sweeps: every spec visits a different mix of projects
// and flags so the suite gives broad surface coverage without piling
// onto a single project.

test("project hop: acme-web → acme-mobile → acme-internal", async ({
  page,
}) => {
  for (const slug of ["acme-web", "acme-mobile", "acme-internal"] as const) {
    await page.goto(`/projects/${slug}`);
    await expect(page.getByTestId("project-stats")).toBeVisible();
    await settle(page, 500);
  }
});

test("flags-list sweep across every seeded project", async ({ page }) => {
  for (const slug of Object.keys(SEEDED_FLAGS) as Array<
    keyof typeof SEEDED_FLAGS
  >) {
    await page.goto(`/projects/${slug}/flags`);
    await expect(page.getByTestId("flags-table")).toBeVisible();
    await settle(page, 400);
  }
});

test("audit sweep across every seeded project", async ({ page }) => {
  for (const slug of Object.keys(SEEDED_FLAGS) as Array<
    keyof typeof SEEDED_FLAGS
  >) {
    await page.goto(`/projects/${slug}/audit`);
    await expect(page.getByTestId("audit-table")).toBeVisible();
    await settle(page, 400);
  }
});

test("snapshots sweep across every seeded project", async ({ page }) => {
  for (const slug of Object.keys(SEEDED_FLAGS) as Array<
    keyof typeof SEEDED_FLAGS
  >) {
    await page.goto(`/projects/${slug}/snapshots`);
    await expect(page.getByTestId("snapshots-table")).toBeVisible();
    await settle(page, 400);
  }
});

test("flag-detail sweep across every seeded acme-web flag", async ({
  page,
}) => {
  for (const key of SEEDED_FLAGS["acme-web"]) {
    await page.goto(`/projects/acme-web/flags/${key}`);
    await expect(page.getByTestId("latest-version")).toBeVisible();
    await settle(page, 400);
  }
});

test("trace-cta sweep across every seeded acme-web flag", async ({ page }) => {
  for (const key of SEEDED_FLAGS["acme-web"]) {
    await page.goto(`/projects/acme-web/flags/${key}`);
    await expect(page.getByTestId("trace-cta")).toBeVisible();
    await settle(page, 350);
  }
});
