import { expect, test } from "@playwright/test";

import { gotoProjects } from "./helpers";

// edit-flag.spec.ts — covers the slice 8 Monaco-based edit route and
// the slice 9 save → snapshot toast round-trip. Targets the seeded
// acme-internal/feature-x flag (typescript strategy with a real
// ff.flag(...) source so Monaco renders TS syntax highlighting), and
// gracefully skips when the API isn't reachable or the flag isn't
// seeded — matching the gotoProjects() convention used elsewhere.

test.describe("edit-flag", () => {
  test("opens a flag's edit page and surfaces Monaco", async ({ page }) => {
    await gotoProjects(page);

    await page.goto("/projects/acme-internal/flags/feature-x/edit");
    if (
      await page
        .getByTestId("error")
        .isVisible()
        .catch(() => false)
    ) {
      test.skip(true, "API/flag not seeded");
    }

    // Monaco lazy-loads — wait up to 15s for either the editor textarea
    // role or the visible monaco container.
    const editor = page.locator(".monaco-editor").first();
    await expect(editor).toBeVisible({ timeout: 15_000 });

    // Save button should be present and enabled before any edit.
    await expect(page.getByTestId("save-cta")).toBeEnabled();
  });

  test("renders the compile-error banner shape", async ({ page }) => {
    await gotoProjects(page);
    await page.goto("/projects/acme-internal/flags/feature-x/edit");
    if (
      await page
        .getByTestId("error")
        .isVisible()
        .catch(() => false)
    ) {
      test.skip(true, "API/flag not seeded");
    }
    // The compile-error-banner is conditional on a prior failed POST.
    // Just verify the banner test id is wired and absent on first load.
    await expect(page.getByTestId("compile-error-banner")).toHaveCount(0);
  });

  test("save round-trip rerenders source and shows publish-snapshot toast", async ({
    page,
  }) => {
    await gotoProjects(page);

    await page.goto("/projects/acme-internal/flags/feature-x/edit");
    if (
      await page
        .getByTestId("error")
        .isVisible()
        .catch(() => false)
    ) {
      test.skip(true, "API/flag not seeded");
    }

    // Wait for Monaco, then set its value via the global monaco API
    // rather than synthetic key events. Typing into Monaco from
    // Playwright runs into auto-close + auto-indent which mangle TS
    // source ("{" → "{}", typed "}" slides past the auto-close,
    // producing extra braces). monaco.editor.getEditors()[0].setValue
    // bypasses both and reliably fires the React-side onChange that
    // mirrors into the hidden <textarea name="source_text">.
    const monaco = page.locator(".monaco-editor").first();
    await expect(monaco).toBeVisible({ timeout: 15_000 });
    const newSource = [
      'import { FalseFlag as ff } from "@falseflag/config";',
      "",
      "export default ff.flag({",
      '  value_type: "boolean",',
      "  default: true,",
      "  rules: [],",
      "});",
      "",
    ].join("\n");
    await page.evaluate((src) => {
      // @ts-expect-error global injected by @monaco-editor/react
      const editors = window.monaco.editor.getEditors();
      if (!editors.length) throw new Error("no monaco editor mounted");
      editors[0].setValue(src);
    }, newSource);
    // Confirm the hidden mirror picked up the new value before submitting.
    await expect(page.locator('textarea[name="source_text"]')).toHaveValue(
      newSource,
    );

    await page.getByTestId("save-cta").click();

    // Redirect lands on the view route with ?published=v{N}; fixture
    // setup and prior local runs can advance the version.
    await page.waitForURL(
      /\/projects\/acme-internal\/flags\/feature-x\?published=v\d+/,
      { timeout: 15_000 },
    );

    // Toast + publish-snapshot CTA from slice 9 phase 3.
    await expect(page.getByTestId("publish-toast")).toBeVisible();
    await expect(page.getByTestId("publish-snapshot-cta")).toBeEnabled();

    // View route re-renders the edited source. The Shiki output wraps
    // the literal text in spans; an unkeyed substring check is robust
    // across whitespace and span boundaries.
    await expect(page.getByTestId("source-code")).toContainText(
      "default: true",
    );
  });
});
