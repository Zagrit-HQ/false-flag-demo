import { expect, test } from "@playwright/test";

import { settle } from "./helpers";

// Error-path coverage. These exercise the 404 / empty-state branches
// the dashboard ships for resources that don't exist.

test("unknown project shows an error banner or empty state", async ({
  page,
}) => {
  await page.goto("/projects/does-not-exist");
  await settle(page, 800);
  const error = page.getByTestId("error");
  const empty = page.getByTestId("empty");
  const errVisible = await error.isVisible().catch(() => false);
  const emptyVisible = await empty.isVisible().catch(() => false);
  expect(errVisible || emptyVisible).toBeTruthy();
});

test("unknown flag does not crash the route", async ({ page }) => {
  // The dashboard renders a Remix error boundary or empty state for
  // a missing flag; either way the page must finish loading without
  // unhandled exceptions.
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto("/projects/acme-web/flags/does-not-exist");
  await settle(page, 800);
  expect(errors, errors.join("\n")).toHaveLength(0);
});

test("unknown flag's trace page renders the form anyway", async ({ page }) => {
  // The trace page renders the form regardless of whether the flag
  // exists — evaluation just fails on submit.
  await page.goto("/projects/acme-web/flags/does-not-exist/trace");
  await expect(page.getByTestId("eval-form")).toBeVisible();
  await settle(page, 500);
  await page.getByTestId("evaluate-btn").click();
  await settle(page, 800);
  await expect(page.getByTestId("error")).toBeVisible();
});

test("audit for unknown project does not crash the route", async ({ page }) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto("/projects/missing-proj/audit");
  await settle(page, 800);
  expect(errors, errors.join("\n")).toHaveLength(0);
});
