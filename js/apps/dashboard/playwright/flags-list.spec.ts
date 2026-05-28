import { expect, test } from "@playwright/test";

import { SEEDED_FLAGS, openProject, settle } from "./helpers";

test("flags table is visible on acme-web", async ({ page }) => {
  await page.goto("/projects/acme-web/flags");
  await expect(page.getByTestId("flags-table")).toBeVisible();
  await settle(page, 800);
});

test("flags table lists the seeded checkout-redesign flag", async ({
  page,
}) => {
  await page.goto("/projects/acme-web/flags");
  await settle(page, 500);
  await expect(page.getByTestId("flags-table")).toContainText(
    "checkout-redesign",
  );
});

test("flags table lists every seeded acme-web flag", async ({ page }) => {
  await page.goto("/projects/acme-web/flags");
  await settle(page, 700);
  const table = page.getByTestId("flags-table");
  for (const key of SEEDED_FLAGS["acme-web"]) {
    await expect(table).toContainText(key);
  }
});

test("flags list is reachable from project detail", async ({ page }) => {
  await openProject(page, "acme-web");
  await settle(page, 400);
  await page.goto("/projects/acme-web/flags");
  await expect(page.getByTestId("flags-table")).toBeVisible();
});

test("acme-mobile flags list shows cel flags", async ({ page }) => {
  await page.goto("/projects/acme-mobile/flags");
  await settle(page, 700);
  const table = page.getByTestId("flags-table");
  await expect(table).toContainText("force-update-required");
  await expect(table).toContainText("push-notification-cadence");
});

test("acme-internal flags list shows feature-x", async ({ page }) => {
  await page.goto("/projects/acme-internal/flags");
  await settle(page, 500);
  await expect(page.getByTestId("flags-table")).toContainText("feature-x");
});
