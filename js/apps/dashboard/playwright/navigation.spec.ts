import { expect, test } from "@playwright/test";

import { settle } from "./helpers";

test("root / redirects to /projects", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveURL(/\/projects$/);
  await settle(page, 400);
});

test("breadcrumb back-nav from flag detail to project", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign");
  await settle(page, 400);
  await page
    .locator("nav")
    .first()
    .getByRole("link", { name: "acme-web" })
    .click();
  await expect(page.getByTestId("project-stats")).toBeVisible();
  await settle(page, 500);
});

test("nav from project to flags to flag detail to trace", async ({ page }) => {
  await page.goto("/projects/acme-web");
  await settle(page, 300);
  await page.goto("/projects/acme-web/flags");
  await settle(page, 300);
  await page.goto("/projects/acme-web/flags/checkout-redesign");
  await settle(page, 300);
  await page.getByTestId("trace-cta").click();
  await expect(page.getByTestId("eval-form")).toBeVisible();
  await settle(page, 600);
});

test("breadcrumb back-nav from trace to projects list", async ({ page }) => {
  await page.goto("/projects/acme-web/flags/checkout-redesign/trace");
  await settle(page, 400);
  await page
    .locator("nav")
    .first()
    .getByRole("link", { name: "Projects" })
    .click();
  await expect(page).toHaveURL(/\/projects$/);
  await settle(page, 500);
});
