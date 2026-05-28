import { expect, test } from "@playwright/test";

import { gotoProjects, openProject, settle } from "./helpers";

test("project detail renders the stats card", async ({ page }) => {
  await openProject(page, "acme-web");
  await expect(page.getByTestId("project-stats")).toBeVisible();
  await settle(page, 600);
});

test("project detail links to the flags list", async ({ page }) => {
  await openProject(page, "acme-web");
  await settle(page, 500);
  await expect(page.getByTestId("flag-preview")).toBeVisible();
});

test("project detail breadcrumb shows the project name", async ({ page }) => {
  await openProject(page, "acme-mobile");
  await settle(page, 600);
  // The breadcrumb renders the project's display_name, not the slug.
  await expect(page.locator("nav").first()).toContainText(/Acme Mobile/i);
});

test("project detail can be reached via the projects list", async ({
  page,
}) => {
  await gotoProjects(page);
  await settle(page, 400);
  await page.getByTestId("projects").getByRole("link").first().click();
  await expect(page).toHaveURL(/\/projects\/[^/]+$/);
  await settle(page, 600);
  await expect(page.getByTestId("project-stats")).toBeVisible();
});

test("each seeded project detail page loads", async ({ page }) => {
  for (const slug of ["acme-web", "acme-mobile", "acme-internal"] as const) {
    await openProject(page, slug);
    await settle(page, 400);
    await expect(page.getByTestId("project-stats")).toBeVisible();
  }
});
