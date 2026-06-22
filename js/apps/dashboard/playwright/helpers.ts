import { type Page, expect, test } from "@playwright/test";

// Seeded project slugs — see playwright/demo/setup.ts.
export const SEEDED_PROJECTS = [
  "acme-web",
  "acme-mobile",
  "acme-internal",
] as const;
export type SeededProject = (typeof SEEDED_PROJECTS)[number];

export const SEEDED_FLAGS: Record<SeededProject, string[]> = {
  "acme-web": [
    "checkout-redesign",
    "max-cart-items",
    "checkout-banner-text",
    "proxy-readiness-bool",
  ],
  "acme-mobile": ["force-update-required", "push-notification-cadence"],
  "acme-internal": ["feature-x"],
};

// gotoProjects navigates to /projects and bails the test gracefully if
// the API isn't reachable (matches the slice-5 dashboard's empty-state
// fallback). After this returns successfully the projects list is in
// the DOM.
export async function gotoProjects(page: Page): Promise<void> {
  await page.goto("/projects");
  await expect(page).toHaveURL(/\/projects$/);
  const errorBanner = page.getByTestId("error");
  if (await errorBanner.isVisible().catch(() => false)) {
    test.skip(true, `API unreachable: ${await errorBanner.textContent()}`);
  }
}

// settle pauses for a configurable amount of time to mimic the kind of
// "wait for the toast to disappear" / "wait for the chart to redraw"
// pauses real Playwright suites accumulate. Each call is a small slice
// of walltime — many of them across many specs amount to the kind of
// e2e suite you'd want to shard.
export async function settle(page: Page, ms = 700): Promise<void> {
  await page.waitForTimeout(ms);
}

// openFirstProject clicks the first project link in the list and waits
// until the project stats card is visible.
export async function openFirstProject(page: Page): Promise<void> {
  await page.getByTestId("projects").getByRole("link").first().click();
  await expect(page.getByTestId("project-stats")).toBeVisible();
}

export async function openProject(
  page: Page,
  slug: SeededProject,
): Promise<void> {
  await page.goto(`/projects/${slug}`);
  await expect(page.getByTestId("project-stats")).toBeVisible();
}
