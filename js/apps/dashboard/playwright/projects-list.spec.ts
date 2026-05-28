import { expect, test } from "@playwright/test";

import { gotoProjects, settle } from "./helpers";

test("projects list renders heading + breadcrumb", async ({ page }) => {
  await gotoProjects(page);
  await expect(page.getByRole("heading", { name: /^Projects$/ })).toBeVisible();
  await settle(page);
  await expect(page.locator("nav").first()).toContainText("Projects");
});

test("projects list shows at least one seeded project", async ({ page }) => {
  await gotoProjects(page);
  const links = page.getByTestId("projects").getByRole("link");
  await expect(links.first()).toBeVisible();
  await settle(page, 500);
  const count = await links.count();
  expect(count).toBeGreaterThan(0);
});

test("projects list shows the seeded acme-web slug", async ({ page }) => {
  await gotoProjects(page);
  await settle(page, 600);
  await expect(page.getByTestId("projects")).toContainText("acme-web");
});

test("projects list shows all three seeded projects", async ({ page }) => {
  await gotoProjects(page);
  const panel = page.getByTestId("projects");
  await settle(page, 800);
  await expect(panel).toContainText("acme-web");
  await expect(panel).toContainText("acme-mobile");
  await expect(panel).toContainText("acme-internal");
});

test("strategy badge renders on the projects list", async ({ page }) => {
  await gotoProjects(page);
  await settle(page, 500);
  // The seed creates a json project (acme-web) and a cel project
  // (acme-mobile).
  await expect(
    page.locator('[data-testid^="strategy-"]').first(),
  ).toBeVisible();
});
