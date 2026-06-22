interface CodeBlockProps {
  /** Pre-rendered Shiki HTML; takes precedence over fallbackJson. */
  html?: string;
  /** Pretty-printed JSON rendered as plain text when html is absent. */
  fallbackJson?: string;
  /** Optional muted caption under the code (e.g. provenance notes). */
  caption?: string;
}

// CodeBlock renders syntax-highlighted source from a server-side Shiki
// run, or falls back to a plain <pre> when the row predates the slice 8
// source_text column. Layout mirrors the existing card aesthetic so the
// flag detail page swap is visually neutral.
export function CodeBlock({ html, fallbackJson, caption }: CodeBlockProps) {
  return (
    <div className="overflow-hidden rounded-md border border-gray-200 bg-white text-xs">
      {html ? (
        <div
          className="overflow-x-auto p-4 [&_pre]:m-0 [&_pre]:bg-transparent [&_pre]:p-0"
          // biome-ignore lint/security/noDangerouslySetInnerHtml: server-rendered by Shiki
          dangerouslySetInnerHTML={{ __html: html }}
          data-testid="latest-version"
        />
      ) : (
        <pre className="overflow-x-auto p-4" data-testid="latest-version">
          <code>{fallbackJson}</code>
        </pre>
      )}
      {caption ? (
        <div
          className="border-t border-gray-100 px-4 py-2 text-[11px] text-gray-500"
          data-testid="latest-version-caption"
        >
          {caption}
        </div>
      ) : null}
    </div>
  );
}
