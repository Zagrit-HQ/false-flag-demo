// edit-corpus.spec.ts — 100 parallel-safe edit round-trip tests
// covering 50 CEL + 50 TS fixtures. Each test owns its own project
// (seeded by global-setup), navigates to its flag's /edit page,
// pastes the fixture's testSource into Monaco, saves, and asserts
// the publish-snapshot toast plus the rendered source contains a
// fixture-unique substring.

import { expect, test } from "@playwright/test";

import { CEL_CORPUS } from "./corpus/cel";
import { FLAG_KEY, projectFor } from "./corpus/setup";
import type { CorpusFixture } from "./corpus/types";
import { TS_CORPUS } from "./corpus/typescript";

const ALL: CorpusFixture[] = [...CEL_CORPUS, ...TS_CORPUS];

/** Drives Monaco via the global window.monaco API. Synthetic
 *  keystrokes hit auto-close/auto-indent and mangle the input; this
 *  function bypasses both and fires the React-side onChange that
 *  mirrors into the hidden <textarea name="source_text">. */
async function setMonacoValue(
  page: import("@playwright/test").Page,
  src: string,
): Promise<void> {
  await page.evaluate((s) => {
    // @ts-expect-error global injected by @monaco-editor/react
    const editors = window.monaco.editor.getEditors();
    if (!editors.length) throw new Error("no monaco editor mounted");
    editors[0].setValue(s);
  }, src);
}

for (const fixture of ALL) {
  test(`${fixture.strategy}: ${fixture.id}`, async ({ page }) => {
    const slug = projectFor(fixture);
    await page.goto(`/projects/${slug}/flags/${FLAG_KEY}/edit`);
    if (
      await page
        .getByTestId("error")
        .isVisible()
        .catch(() => false)
    ) {
      test.skip(true, `fixture not seeded for ${fixture.id}`);
    }

    const monaco = page.locator(".monaco-editor").first();
    await expect(monaco).toBeVisible({ timeout: 15_000 });

    await setMonacoValue(page, fixture.testSource);

    // Sanity: the hidden mirror has the new value before we submit.
    await expect(page.locator('textarea[name="source_text"]')).toHaveValue(
      fixture.testSource,
    );

    await page.getByTestId("save-cta").click();

    // Save lands on the view route with ?published=v{N}. The N
    // monotonically increments per re-run; match the shape only.
    await page.waitForURL(
      new RegExp(`/projects/${slug}/flags/${FLAG_KEY}\\?published=v\\d+`),
      { timeout: 15_000 },
    );

    await expect(page.getByTestId("publish-toast")).toBeVisible();
    await expect(page.getByTestId("publish-snapshot-cta")).toBeEnabled();
    await expect(page.getByTestId("source-code")).toContainText(
      fixture.assertSubstring,
    );
  });
}
