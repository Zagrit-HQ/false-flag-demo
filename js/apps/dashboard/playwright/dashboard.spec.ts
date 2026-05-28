import { expect, test } from "@playwright/test";

// dashboard.spec.ts is the slice-5 Playwright happy-path. It expects:
//   - the dashboard is up on baseURL (Playwright's webServer launches
//     `pnpm dev` automatically).
//   - the API is up and `make seed` has been run, providing at least
//     one project with at least one published flag.
//
// If the API isn't reachable, the dashboard renders an empty state
// banner instead of crashing — the test skips downstream assertions
// when that empty state is visible.

test("user navigates from projects to a flag's trace explorer", async ({
  page,
}) => {
  await page.goto("/");

  // The home page redirects to /projects.
  await expect(page).toHaveURL(/\/projects$/);
  await expect(page.getByRole("heading", { name: /^Projects$/ })).toBeVisible();

  // If the API is unreachable, an error banner appears. Bail gracefully.
  const errorBanner = page.getByTestId("error");
  if (await errorBanner.isVisible().catch(() => false)) {
    test.skip(true, `API unreachable: ${await errorBanner.textContent()}`);
  }

  // Pick the first project in the list, follow it.
  const firstProjectLink = page
    .getByTestId("projects")
    .getByRole("link")
    .first();
  await firstProjectLink.click();

  // Project detail page.
  await expect(page.getByTestId("project-stats")).toBeVisible();

  // Pick the first flag from the preview list.
  const previewLink = page
    .getByTestId("flag-preview")
    .getByRole("link")
    .first();
  if (!(await previewLink.isVisible().catch(() => false))) {
    test.skip(true, "no flags seeded in this project");
  }
  await previewLink.click();

  // Flag detail page.
  await expect(page.getByTestId("trace-cta")).toBeVisible();

  // Navigate to the trace explorer.
  await page.getByTestId("trace-cta").click();
  await expect(page.getByTestId("eval-form")).toBeVisible();

  // Submit the default context.
  await page.getByTestId("evaluate-btn").click();

  // The decision panel should appear.
  await expect(page.getByTestId("decision")).toBeVisible();
});
