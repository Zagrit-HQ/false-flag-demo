import { expect, test } from "@playwright/test";

import { SEEDED_FLAGS, settle } from "./helpers";

// One spec per seeded flag — each navigates to the flag detail page,
// asserts the version pane renders, and pauses for a realistic
// "wait for sparkline to draw" interval.

const ALL_FLAGS: Array<{ slug: keyof typeof SEEDED_FLAGS; key: string }> = [];
for (const [slug, keys] of Object.entries(SEEDED_FLAGS) as Array<
  [keyof typeof SEEDED_FLAGS, string[]]
>) {
  for (const key of keys) ALL_FLAGS.push({ slug, key });
}

for (const f of ALL_FLAGS) {
  test(`flag detail: ${f.slug}/${f.key} shows latest version`, async ({
    page,
  }) => {
    await page.goto(`/projects/${f.slug}/flags/${f.key}`);
    await expect(page.getByTestId("latest-version")).toBeVisible();
    await settle(page, 700);
  });

  test(`flag detail: ${f.slug}/${f.key} shows trace CTA`, async ({ page }) => {
    await page.goto(`/projects/${f.slug}/flags/${f.key}`);
    await expect(page.getByTestId("trace-cta")).toBeVisible();
    await settle(page, 600);
  });
}
