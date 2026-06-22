import { type Highlighter, createHighlighter } from "shiki";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";

// Module-scoped singleton. createHighlighter is async and pulls
// grammars + themes; reusing the instance across requests keeps the
// per-render cost negligible. The .server.ts suffix keeps this out of
// any client bundle.
let cached: Promise<Highlighter> | null = null;

export function getHighlighter(): Promise<Highlighter> {
  if (!cached) {
    cached = createHighlighter({
      langs: ["typescript", "javascript", "json"],
      themes: ["github-light"],
      // JavaScript regex engine — avoids shipping the Oniguruma WASM
      // asset, which complicates the Vite SSR build for no benefit on
      // our three small languages.
      engine: createJavaScriptRegexEngine(),
    });
  }
  return cached;
}

const langFor: Record<string, "typescript" | "javascript" | "json"> = {
  typescript: "typescript",
  // No CEL grammar in Shiki; JavaScript-flavoured highlighting renders
  // user.plan == "pro" the way you'd expect.
  cel: "javascript",
  json: "json",
};

export async function highlightSource(
  source: string,
  strategy: string,
): Promise<string> {
  const hl = await getHighlighter();
  return hl.codeToHtml(source, {
    lang: langFor[strategy] ?? "json",
    theme: "github-light",
  });
}
