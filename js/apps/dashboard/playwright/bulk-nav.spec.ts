import { expect, test } from "@playwright/test";

import { SEEDED_FLAGS, settle } from "./helpers";

// Bulk navigation specs: every (project, sub-route) pair gets a small
// visit-and-assert test. Many small specs is the whole point —
// they parallelise per shard cleanly.

const PROJECTS = ["acme-web", "acme-mobile", "acme-internal"] as const;
const SUBROUTES = [
  { path: "", testid: "project-stats" },
  { path: "/flags", testid: "flags-table" },
  { path: "/audit", testid: "audit-table" },
  { path: "/snapshots", testid: "snapshots-table" },
] as const;

for (const slug of PROJECTS) {
  for (const r of SUBROUTES) {
    test(`nav: ${slug}${r.path || "/"}`, async ({ page }) => {
      await page.goto(`/projects/${slug}${r.path}`);
      await settle(page, 500);
      await expect(page.getByTestId(r.testid)).toBeVisible();
      await settle(page, 400);
    });
  }
}

// Per-flag trace landing pages: 7 flags = 7 specs.
const ALL_FLAGS: Array<{ slug: keyof typeof SEEDED_FLAGS; key: string }> = [];
for (const [slug, keys] of Object.entries(SEEDED_FLAGS) as Array<
  [keyof typeof SEEDED_FLAGS, string[]]
>) {
  for (const key of keys) ALL_FLAGS.push({ slug, key });
}

for (const f of ALL_FLAGS) {
  test(`nav: trace landing for ${f.slug}/${f.key}`, async ({ page }) => {
    await page.goto(`/projects/${f.slug}/flags/${f.key}/trace`);
    await expect(page.getByTestId("eval-form")).toBeVisible();
    await settle(page, 500);
  });
}

// Click-through specs that exercise the breadcrumb back-nav from
// each leaf page.
for (const slug of PROJECTS) {
  test(`breadcrumb back-nav: ${slug} flags → project`, async ({ page }) => {
    await page.goto(`/projects/${slug}/flags`);
    await settle(page, 400);
    await page.locator("nav").first().getByRole("link", { name: slug }).click();
    await expect(page.getByTestId("project-stats")).toBeVisible();
    await settle(page, 500);
  });
}
