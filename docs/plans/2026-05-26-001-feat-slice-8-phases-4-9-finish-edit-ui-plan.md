---
title: Slice 8 Phases 4-9 — finish server-side TS compile + view/edit source UX
type: feat
status: active
date: 2026-05-26
predecessor: docs/plans/2026-05-22-001-feat-server-ts-compile-and-edit-ui-plan.md
---

# Slice 8 Phases 4-9 — finish server-side TS compile + view/edit source UX

## Overview

Phases 1-3 of slice 8 shipped to `main` on 2026-05-22 (commits `b3735de`, `7912d92`, `65d8f97`): the `source_text` column exists on `flag_versions`, the real esbuild+goja compiler is wired in `internal/config/typescript.go` (285 lines, fully replaces the stub), and both REST + Connect handlers accept `source_text` and surface 422s on compile failure.

What remains:

- **Phase 4** — fix the nested transaction in `Store.WithAudit` + `Store.PublishFlagVersion`. They currently open two independent Postgres transactions; the inner one is serializable, the outer audit txn is default isolation, and a failure/panic in the inner code path after its commit does **not** roll back the audit row. The handlers already receive a txn-scoped `*db.Queries` from `WithAudit` and discard it with `func(_ *db.Queries)`. Phase 4 plumbs that `q` through.
- **Phase 5** — CLI populates `source_text` for `typescript`-strategy flags. orval-generated client and Zod schema already carry the field.
- **Phase 6** — two new corpus fixtures with real TS `source_text` (rollout + nested CEL), a new `tests/hurl/14-typescript-publish.hurl` (10 is missing, 11/12/13 already taken), and a Playwright spec skeleton for edit-flag.
- **Phase 7** — replace the `<pre>{JSON.stringify(latest.compiled, null, 2)}</pre>` block on the flag view route with Shiki SSR using the JavaScript regex engine (no WASM, sidesteps Vite SSR asset handling). Add an "Edit" link to the new edit route.
- **Phase 8** — new edit route `projects.$slug.flags.$key.edit.tsx` with `@monaco-editor/react` lazy-imported via the Remix 2.x `.client.tsx` convention. Remix `action` POSTs to `publishFlagVersion`; 422 compile errors surface as `setModelMarkers` annotations + a banner.
- **Phase 9** — update `internal/config/README.md` + `js/packages/config-ts/README.md` to describe the new pipeline. (`infra/Dockerfile` and `.github/workflows/ci.yml` do **not** change — both new Go deps are pure Go, and CI auto-triggers are intentionally off ahead of slice 7b.)

PR grouping: Phases 4-6 ship as one PR (backend correctness + tests). Phases 7-8 ship together (view+edit is one user-visible delta). CI workflow auto-triggers stay off (slice 7b owns re-enabling them).

## Problem Statement / Motivation

**1. Nested-transaction bug is now load-bearing.** Phase 3 routed all 422-eligible writes through `WithAudit(... func(_ *db.Queries) error { Store.PublishFlagVersion(...) })`. The outer audit txn is default isolation; the inner publish txn is `Serializable`. Today, if `Store.PublishFlagVersion` commits its inner txn and then the audit append fails (or the request context cancels), the flag_version row is committed and orphaned — `audit_events` has no matching row. The reverse is also true under retry. This worked accidentally for the demo so far because nothing in normal happy-path flow exercises the failure mode, but the slice 8 edit UX will exercise it (every save is a publish + audit pair under user-visible latency), and a half-committed write would surface as a flag with no audit trace on the next page load. Fix it before Monaco can write.

**2. CLI is still IR-only for `typescript`.** Phase 3 made the server authoritative for TS compilation, but the CLI's `falseflag config publish` still sends `{strategy, source: rawTSstring}` and never populates `source_text`. The server happily compiles `source` as TS when `source_text` is absent (legacy path), but no row will have `source_text` populated unless the dashboard writes it — which means the view route will show "compiled IR — original source not stored" for every CLI-published flag.

**3. Dashboard renders JSON for TS flags.** The whole point of slice 8: the flag detail page at line 92-96 of `projects.$slug.flags.$key._index.tsx` shows `<pre><code>{JSON.stringify(latest.compiled, null, 2)}</code></pre>`. A TS-authored flag shows compiled IR JSON. That UX is what triggered this slice.

**4. There's no edit affordance.** Every value change requires `falseflag config publish`. Demo audience expects a button.

## Proposed Solution

### Phase 4: Real single-transaction publish

`internal/store/flags.go`: rename the existing `PublishFlagVersion(ctx, params)` to `PublishFlagVersionTx(ctx, q *db.Queries, params)` (no `pool.BeginTx`; uses caller's `q`). Add a thin `Store.PublishFlagVersionStandalone(ctx, params)` wrapper that opens its own serializable txn — for callers (e.g. tests) that want isolation without an audit row.

`internal/store/audit.go`: `WithAudit` keeps its signature but upgrades the isolation level it requests to `Serializable` (matching what `PublishFlagVersion` used to do internally). The closure already receives `*db.Queries` — handlers now actually use it.

`internal/server/handlers/flags.go:182-201` and `internal/server/rpc/flags.go:138-157`: replace `func(_ *db.Queries) error { Store.PublishFlagVersion(...) }` with `func(q *db.Queries) error { Store.PublishFlagVersionTx(ctx, q, params) }`.

Add `TestWithAuditRollsBackOnPanic` in `internal/store/integration_test.go` after the existing `TestWithAuditRollsBackOnMutationFailure` (line ~278). Asserts: a panic inside the audit closure (after `PublishFlagVersionTx` returns nil) causes the flag_version insert to roll back — `q.GetLatestFlagVersion` after recovery returns `ErrNoRows`.

### Phase 5: CLI sends `source_text`

`js/apps/cli/src/commands/config.ts:34-44, 106-123`:

- `readSource` keeps returning `string | object`. Add a sibling `readSourceText(filePath)` that returns `string | null`: returns the raw file contents when the filename ends `.ts`, `.json`, or `.cel`; returns `null` otherwise (defensive — no current path hits null).
- In `save`: before calling `publishFlagVersion`, capture `const sourceText = readSourceText(opts.file)`. Send `{ strategy, source, source_text: sourceText ?? undefined }`. The orval-generated `PublishFlagVersionRequest` type already accepts the optional field; no codegen run needed.

`js/apps/cli/tests/config.test.ts` line ~132: update the captured-body assertion to expect `source_text` matching the temp file contents byte-for-byte for the existing `.json` fixture (since `readSourceText` returns raw text for `.json` too — keeps the assertion uniform). Add a second test case using a `.ts` fixture and assert the same.

### Phase 6: Tests + corpus + Playwright skeleton

Two new fixtures in `tests/eval-corpus/`:

- `26-typescript-rollout.json` — `strategy: "typescript"`, `source_text` is real TS exercising `rollout(...)` with a 25% bucket and a salt; `ir` is the expected compiled output. Pick one `context` that lands inside the 25% and one that doesn't (extend `expected` accordingly per existing fixture shape).
- `27-typescript-nested-cel.json` — `strategy: "typescript"`, `source_text` is real TS using `all(...)` wrapping a `cel("user.plan == 'pro' && user.region in ['us','eu']")` predicate alongside an `eq` predicate; `ir` is the expected compiled output.

Both fixtures only get loaded by tests that look at `ir`. Per Phase 5 research, neither `internal/sdkgo/conformance_test.go` nor `js/packages/sdk-js/tests/conformance.test.ts` decode `source_text` — they read `ir` directly. No conformance-test changes required for Phase 6.

New `tests/hurl/14-typescript-publish.hurl` (numbering: 10 missing, 11/12/13 already taken; 14 is next free) covers:

1. PUT TS via `source_text` only → 201; assert the response includes `source_text` and `compiled` with the expected IR shape.
2. PUT malformed TS via `source_text` (e.g. `const x: = 1;`) → 422; assert `error.details[0].line == 1` and `error.details[0].text` is non-empty.
3. PUT with legacy `source` only (no `source_text`) → 201; assert `source_text` is `null` in the response.
4. PUT with both fields where the IRs would differ → 201; server's compile wins. (Verifying the warning log requires out-of-band capture; the Hurl assertion is limited to "201, server-compiled IR in response".)

`scripts/smoke.sh` doesn't need changes — it already globs `tests/hurl/*.hurl`. New file is picked up automatically.

New `js/apps/dashboard/playwright/edit-flag.spec.ts` (skeleton — full assertion runs in Phase 8 once the edit route exists):

```ts
import { test, expect } from "@playwright/test";
import { gotoProjects } from "./helpers";

test("edit a typescript flag round-trips through Monaco", async ({ page }) => {
  await gotoProjects(page);
  // resolved fully in Phase 8 — for Phase 6 just exit cleanly so CI is green
  test.skip(true, "Phase 8 not landed yet");
});
```

Phase 8 swaps the `test.skip` for the real assertion.

### Phase 7: Shiki SSR view route

`pnpm --filter @falseflag/dashboard add shiki@^3` (current major; the [Shiki 3.x docs](https://shiki.style/) document `createJavaScriptRegexEngine` as the WASM-free path).

New `js/apps/dashboard/app/lib/highlighter.server.ts`:

```ts
import { createHighlighter, type Highlighter } from "shiki";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";

let cached: Promise<Highlighter> | null = null;

export function getHighlighter(): Promise<Highlighter> {
  if (!cached) {
    cached = createHighlighter({
      langs: ["typescript", "javascript", "json"],
      themes: ["github-light"],
      engine: createJavaScriptRegexEngine(),
    });
  }
  return cached;
}

const langFor: Record<string, "typescript" | "javascript" | "json"> = {
  typescript: "typescript",
  cel: "javascript", // no CEL grammar — JS regex highlighting is close enough
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
```

The `.server.ts` suffix keeps it server-side only (Remix 2.x convention). Module-scoped singleton — `createHighlighter` is async and expensive (~50ms cold), so reuse across requests.

New `js/apps/dashboard/app/components/CodeBlock.tsx`:

```tsx
type CodeBlockProps = {
  html?: string;          // pre-rendered Shiki HTML
  fallbackJson?: string;  // pretty-printed IR fallback
  caption?: string;
};

export function CodeBlock({ html, fallbackJson, caption }: CodeBlockProps) {
  return (
    <div className="rounded-md border border-gray-200 bg-white text-xs">
      {html ? (
        <div
          className="overflow-x-auto p-4 [&_pre]:m-0 [&_pre]:bg-transparent"
          // biome-ignore lint/security/noDangerouslySetInnerHtml: Shiki SSR
          dangerouslySetInnerHTML={{ __html: html }}
          data-testid="latest-version"
        />
      ) : (
        <pre
          className="overflow-x-auto p-4"
          data-testid="latest-version"
        >
          <code>{fallbackJson}</code>
        </pre>
      )}
      {caption ? (
        <div className="border-t border-gray-100 px-4 py-2 text-[11px] text-gray-500">
          {caption}
        </div>
      ) : null}
    </div>
  );
}
```

The `data-testid="latest-version"` matches the existing pre block so the Playwright spec at `dashboard.spec.ts` (currently asserting on the JSON pretty-print) continues to work — it'll find the new wrapper.

`js/apps/dashboard/app/routes/projects.$slug.flags.$key._index.tsx`:

- Loader (line 31-55): add `source_text` to the fields read off `latest`. The orval `FlagVersion` type already includes `source_text?: string | null` from Phase 3. When `latest.source_text` is present, call `highlightSource(latest.source_text, latest.strategy)` and pass `{ html, caption: undefined }` to the component. When absent, pass `{ fallbackJson: JSON.stringify(latest.compiled, null, 2), caption: "compiled IR — original source not stored" }`.
- Component (line 92-96): swap the `<pre>` block for `<CodeBlock {...sourceProps} />`.
- Header (line 73 area): add an "Edit" link next to `StrategyBadge` pointing to `/projects/${slug}/flags/${key}/edit`. Style as a small button matching the existing trace CTA link. New component if helpful, inline `<Link>` otherwise (keep it simple).

Snapshot test for `CodeBlock`: `js/apps/dashboard/app/components/CodeBlock.test.tsx` — Vitest + `@testing-library/react`. Snapshot one render with `html="<pre>x</pre>"` and one with `fallbackJson` set. Asserts the wrapper structure is stable. (Don't snapshot Shiki's actual output — its output across versions isn't stable; rely on the integration via the Playwright spec for that.)

### Phase 8: Monaco lazy edit route

`pnpm --filter @falseflag/dashboard add @monaco-editor/react@^4 monaco-editor@^0.52`.

`js/apps/dashboard/vite.config.ts` — add:

```ts
optimizeDeps: {
  include: ["monaco-editor"],
},
```

A `MonacoEnvironment.getWorker` shim is **not** required because `@monaco-editor/react` ships an internal loader that fetches from CDN by default. For demo-quality, the CDN loader is fine; if it turns out to be blocked in some browsers, the fallback (workers via `?worker` imports + `loader.config({ paths: { vs: ... } })`) can be added without breaking the route shape. Document this trade in the route file's top-of-file comment.

New `js/apps/dashboard/app/components/editor.client.tsx`:

```tsx
import Editor, { type OnMount } from "@monaco-editor/react";
import { useRef } from "react";

export type CompileErrorDetail = {
  line: number;
  column: number;
  text: string;
};

type Props = {
  value: string;
  language: "typescript" | "javascript" | "json";
  onChange: (next: string) => void;
  errors?: CompileErrorDetail[];
};

export default function CodeEditor({ value, language, onChange, errors }: Props) {
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null);
  const monacoRef = useRef<Parameters<OnMount>[1] | null>(null);

  const handleMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;
    monacoRef.current = monaco;
    applyMarkers(editor, monaco, errors);
  };

  // re-apply markers when errors change
  if (editorRef.current && monacoRef.current) {
    applyMarkers(editorRef.current, monacoRef.current, errors);
  }

  return (
    <div className="h-[480px] overflow-hidden rounded-md border border-gray-200">
      <Editor
        height="100%"
        language={language}
        value={value}
        onChange={(v) => onChange(v ?? "")}
        onMount={handleMount}
        theme="vs"
        options={{ minimap: { enabled: false }, fontSize: 13 }}
      />
    </div>
  );
}

function applyMarkers(
  editor: Parameters<OnMount>[0],
  monaco: Parameters<OnMount>[1],
  errors: CompileErrorDetail[] = [],
) {
  const model = editor.getModel();
  if (!model) return;
  monaco.editor.setModelMarkers(
    model,
    "falseflag-compile",
    errors.map((e) => ({
      severity: monaco.MarkerSeverity.Error,
      startLineNumber: e.line,
      endLineNumber: e.line,
      startColumn: e.column,
      endColumn: e.column + 1,
      message: e.text,
    })),
  );
}
```

The `.client.tsx` suffix is the Remix 2.x browser-only file convention — Remix replaces it with an empty module during SSR.

New `js/apps/dashboard/app/components/EditorSkeleton.tsx`:

```tsx
export function EditorSkeleton() {
  return (
    <div
      className="h-[480px] animate-pulse rounded-md border border-gray-200 bg-gray-50"
      aria-busy="true"
      data-testid="editor-skeleton"
    />
  );
}
```

New `js/apps/dashboard/app/routes/projects.$slug.flags.$key.edit.tsx` — mirrors the `.trace.tsx` sibling pattern that already exists.

Loader: same `getFlag(slug, key)` + `listFlagVersions(slug, key)` calls as the view route. Pass `{ slug, key, flag, latest, strategy: latest.strategy, initialSource: latest.source_text ?? JSON.stringify(latest.compiled, null, 2) }`.

Action: read form data `{ source_text, strategy }`. Call `publishFlagVersion(slug, key, { strategy, source: undefined, source_text })`. The orval client expects `source` but it's the legacy path now — pass an empty object (or omit if the type allows). If the response is 422 with structured details, return `{ errors: details }` via `useActionData`. If 201, `redirect` back to `/projects/${slug}/flags/${key}`.

Component: `const Editor = React.lazy(() => import("~/components/editor.client"))`. Render inside `<Suspense fallback={<EditorSkeleton/>}>`. Track `source` in `useState` initialized from loader data. On submit, POST as a Remix `<Form method="post">` carrying the current value. Cancel link goes back to the view route.

Bundle verification: after `pnpm --filter @falseflag/dashboard build`, inspect the chunk output; the view route's chunk must not contain `monaco-editor`. Document the expected chunk shape in a one-line comment at the top of `editor.client.tsx`.

Edit-flag Playwright spec (replaces the Phase 6 skeleton):

1. Seed a `typescript`-strategy flag (the existing `cmd/falseflag-seed` already creates `acme-web`'s seven flags including at least one TS flag — verify by reading the seed binary; if none is TS, add one in this slice).
2. Navigate to `/projects/acme-web/flags/<ts-flag-key>/edit`.
3. Wait for the editor (`[role="textbox"]` or Monaco's `.monaco-editor` selector) to be visible.
4. Use `page.keyboard` to modify a string literal in the source.
5. Click Save.
6. Assert redirect to the view route; assert the rendered source HTML now contains the edited string.

The spec gracefully `test.skip`s on `API unreachable` per the existing `gotoProjects` helper pattern.

### Phase 9: Docs

`internal/config/README.md`: update the `typescript` section to describe the new pipeline (esbuild → goja → IR validator → CEL program compile) and replace any stale "Slice 2 does NOT execute user-submitted TypeScript" caveats. If the README doesn't exist yet (likely — none of the internal/* packages have READMEs per slice 1-5 audits), don't create one for the demo; instead, append a top-of-file doc comment to `internal/config/typescript.go` summarizing the pipeline (3-4 lines).

`js/packages/config-ts/README.md`: same posture — if it doesn't exist, append a top-of-file comment to `src/index.ts`. The CLI now sends `source_text`; the server is authoritative.

## Technical Approach

### Architecture (delta from Phase 3)

```
Phase 4 only — interaction graph change:

PUT /v1/projects/{slug}/flags/{key}
  ├── REST handler (internal/server/handlers/flags.go:109)
  │     ↓ publishCoordinator (already in place from Phase 3)
  ├── config.Compile(strategy, source_text || source)  // unchanged
  ├── Store.WithAudit(ctx, ev, func(q *db.Queries) error {
  │     // q is NOW USED (was: discarded with `_`)
  │     v, err := Store.PublishFlagVersionTx(ctx, q, params)
  │     ...
  │   })
  └── single Postgres txn (Serializable) covers both
      flag_versions INSERT and audit_events INSERT
```

Phase 5-8 don't change the API contract; they change what populates `source_text` (CLI) and what renders (dashboard).

### Data model

No schema changes in this PR. The `source_text` column already exists from Phase 1.

### Generated artifacts

No proto changes. No OpenAPI changes. No sqlc changes. orval-generated `PublishFlagVersionRequest` already carries `source_text` from Phase 3. `make generate-check` must still pass on a clean tree — no drift expected.

### Bundling discipline (Phases 7-8)

- **Shiki** is server-only. `highlighter.server.ts` enforces that. The `.server.ts` suffix is Remix's convention for server-only modules; Vite's Remix plugin tree-shakes any accidental client import.
- **Monaco** is client-only. `.client.tsx` suffix enforces that. `React.lazy(() => import("..."))` plus the `<Suspense>` boundary plus the `.client.tsx` suffix together ensure Monaco never lands in SSR output and never lands in the view route's chunk.
- After `pnpm build`, verify two things: (a) the view route chunk does not import `monaco-editor`; (b) `dist/server/` does not contain `monaco-editor`. Manual check is sufficient for the demo — no CI gate needed.

## Implementation Phases

### Phase 4 — Nested-transaction fix (~2 hrs)

Tasks:
- `internal/store/flags.go`: rename `PublishFlagVersion` → `PublishFlagVersionTx(ctx, q *db.Queries, params)`. Remove the `pool.BeginTx`; use `q` for `NextFlagVersion` + `CreateFlagVersion`. Add `Store.PublishFlagVersionStandalone(ctx, params)` that opens `pool.BeginTx` with serializable isolation, calls `PublishFlagVersionTx`, commits.
- `internal/store/audit.go`: bump `WithAudit`'s `BeginTx` to `pgx.TxOptions{IsoLevel: pgx.Serializable}`.
- `internal/server/handlers/flags.go:182-201`: change the closure to `func(q *db.Queries) error` and call `Store.PublishFlagVersionTx(ctx, q, params)`.
- `internal/server/rpc/flags.go:138-157`: same.
- Grep `PublishFlagVersion(` across the repo for any other callers (snapshots may compile via a separate path; verify they don't publish versions inline). Update or migrate to `PublishFlagVersionStandalone` as appropriate.
- `internal/store/integration_test.go` after line 278: add `TestWithAuditRollsBackOnPanic`. Uses `newTestStore(t)`, creates a project + flag, calls `WithAudit` with a closure that runs `PublishFlagVersionTx` then `panic("forced")`. Wraps in `recover()`. Asserts: `q.ListFlagVersions(ctx, ...)` returns 0 rows after recovery.

Success criteria:
- `go build ./...` clean.
- `go vet ./...` clean.
- `go test ./...` clean (no regressions in `internal/server/`, `internal/store/`, `internal/config/`).
- `FALSEFLAG_TEST_DATABASE_URL=… go test ./internal/store/... ./internal/server/...` clean (live Postgres path).
- `make contract-test` clean (REST↔Connect parity).
- New `TestWithAuditRollsBackOnPanic` asserts no orphan row exists.

### Phase 5 — CLI sends `source_text` (~1.5 hrs)

Tasks:
- `js/apps/cli/src/commands/config.ts`: add `readSourceText(filePath)` helper returning `string | null`. Update `save()` to capture it before the API call and add `source_text` to the request body.
- `js/apps/cli/tests/config.test.ts`: update existing test (line ~132) to expect `source_text` in the captured body. Add a new test using a `.ts` fixture (temp file with `import * as F from "@falseflag/config"; export default F.FalseFlag.flag({...})`) and assert `source_text` matches the file contents.

Success criteria:
- `pnpm --filter @falseflag/cli typecheck` clean.
- `pnpm --filter @falseflag/cli test` clean (existing + new tests).
- `pnpm --filter @falseflag/cli build` clean.

### Phase 6 — Corpus + Hurl + Playwright skeleton (~4 hrs)

Tasks:
- `tests/eval-corpus/26-typescript-rollout.json` — new fixture with TS `source_text` exercising `rollout`.
- `tests/eval-corpus/27-typescript-nested-cel.json` — new fixture with TS `source_text` exercising `all(...)` + `cel(...)` + `eq`.
- `tests/hurl/14-typescript-publish.hurl` — four cases per the spec above.
- `js/apps/dashboard/playwright/edit-flag.spec.ts` — skeleton with `test.skip` (Phase 8 replaces).

Success criteria:
- `go test ./internal/sdkgo/... ./internal/config/...` clean (new fixtures load fine — both packages just need the `ir` field).
- `pnpm --filter @falseflag/shared-eval-corpus test` clean.
- `pnpm --filter @falseflag/sdk-js test` clean.
- `make conformance` clean (Go SDK 27/27, TS SDK 27/27).
- `make smoke` clean (13 Hurl files: `00` through `09`, `11`, `12`, `13`, `14`; ~78 requests).
- `pnpm --filter @falseflag/dashboard exec playwright test edit-flag.spec.ts` runs (test.skip is fine).

### Phase 7 — Shiki SSR view (~3 hrs)

Tasks:
- Install shiki@^3 via pnpm.
- New `app/lib/highlighter.server.ts`.
- New `app/components/CodeBlock.tsx`.
- New `app/components/CodeBlock.test.tsx`.
- Update `app/routes/projects.$slug.flags.$key._index.tsx` loader + render path.
- Add Edit link next to StrategyBadge.

Success criteria:
- `pnpm --filter @falseflag/dashboard typecheck` clean.
- `pnpm --filter @falseflag/dashboard test` clean.
- `pnpm --filter @falseflag/dashboard build` clean (SSR bundle does NOT contain `monaco-editor` since it's not imported yet — sanity check before Phase 8).
- `pnpm --filter @falseflag/dashboard lint` clean (Biome).
- Manual: open `/projects/acme-web/flags/<a TS flag>` in the seeded compose stack; verify highlighted TS source renders.

### Phase 8 — Monaco lazy edit (~5 hrs)

Tasks:
- Install `@monaco-editor/react@^4` + `monaco-editor@^0.52`.
- Update `vite.config.ts` with `optimizeDeps.include`.
- New `app/components/editor.client.tsx`.
- New `app/components/EditorSkeleton.tsx`.
- New `app/routes/projects.$slug.flags.$key.edit.tsx`.
- Replace the Phase 6 Playwright skeleton with the real spec.
- Manually verify bundle separation (Monaco only in edit-route chunk).

Success criteria:
- `pnpm --filter @falseflag/dashboard typecheck` clean.
- `pnpm --filter @falseflag/dashboard test` clean.
- `pnpm --filter @falseflag/dashboard build` clean.
- `pnpm --filter @falseflag/dashboard lint` clean.
- `make dashboard-e2e` runs the new edit-flag spec to completion (or `test.skip`s on API-unreachable per existing pattern).
- Manual: open `/projects/acme-web/flags/<a TS flag>/edit`, edit a string, save, see the change reflected on the view route.

### Phase 9 — Docs + status (~1.5 hrs)

Tasks:
- Append doc comment to `internal/config/typescript.go` describing the pipeline (skip README creation — no precedent for `internal/*/README.md`).
- Append doc comment to `js/packages/config-ts/src/index.ts` noting server-side authoritative compile.

Success criteria:
- No README files created.

**Total estimated effort: ~17 hours / 2-2.5 working days.**

## Alternative Approaches Considered

**A1: Skip Phase 4 entirely; ship Phases 5-9 first.** Tempting (no DB refactor risk on a busy day), but Monaco edits *will* exercise the audit-rollback path under user-facing latency. Better to fix the foundation before piling traffic on it.

**A2: Make `WithAudit` accept a `func() error` instead of `func(q *db.Queries) error`.** Then `WithAudit` opens the txn, runs the inner function, and that's it — no `q` plumbing. Rejected: the inner code (`PublishFlagVersionTx`) needs `q` to do its own writes. Hiding it makes the caller go fetch `q` somehow else, which is worse.

**A3: Drop `@monaco-editor/react`; write a custom Monaco wrapper.** Rejected — `@monaco-editor/react` is the canonical wrapper, well-maintained, and its CDN loader handles the worker setup for us. Building one ourselves trades 30 minutes saved on bundle size for several hours of worker config bugs.

**A4: Use CodeMirror 6 instead of Monaco.** Smaller bundle (~150KB vs Monaco's ~5MB), modern API. Rejected for the demo: Monaco is what the audience expects to see when they think "code editor in a web app" (VS Code is Monaco). Demo signal > bundle byte count.

**A5: Use Highlight.js or Prism instead of Shiki.** Smaller; client-side; works without SSR plumbing. Rejected: Shiki SSR produces zero client JS for highlighting (already-rendered HTML), which is the right posture for an Always-SSR Remix route. Prism's grammar quality on TypeScript is also notably worse than Shiki's.

## System-Wide Impact

### Interaction graph

Phase 4: every `PublishFlagVersion` call site now runs inside one Postgres txn that also writes the audit row. If anything in the closure fails or panics, both rolls back. No other consumer of `WithAudit` exists today (grep confirms it's only called from the publish handlers).

Phase 5: CLI publish requests now carry `source_text`. Server reads it and re-compiles. CLI's local TS compile is now a smoke-check (verifies the file parses locally before sending), not the source of truth.

Phases 7-8: dashboard view + edit routes hit the same `/v1/projects/{slug}/flags/{key}` GET/PUT endpoints. No new server-side endpoints. Snapshot republish is *not* triggered on edit — user must hit `POST /v1/projects/{slug}/snapshots` manually (out of scope per slice 8 plan §A5).

### Error & failure propagation

- **Phase 4**: serialization conflicts now surface as a single txn error to the handler. The existing 409-on-conflict mapping in handlers (if present) still applies.
- **Phase 5**: CLI ignores `source_text`-related server errors gracefully if the field is absent in the type (it's not — Phase 3 already added it). 422 from the server is surfaced via the existing error-formatting helper.
- **Phase 7**: Shiki failures (e.g., a language grammar load error) shouldn't happen at runtime once the highlighter is initialized. Wrap `highlightSource()` in a try/catch in the loader and fall back to the JSON-pretty-print path on error. Log the error to `slog` (server-side).
- **Phase 8**: Monaco lazy-load failure (CDN unreachable) shows the `EditorSkeleton` forever. Acceptable demo behavior; if a kiosk environment matters later, ship a self-hosted worker setup.

### State lifecycle risks

After Phase 4, the only persistent writes from a publish are `flag_versions` + `audit_events`, in one txn. Phase 5 doesn't add new writes. Phases 7-8 are read-mostly + the existing publish write. No new state lifecycle surface.

### API surface parity

Phase 4 must keep REST↔Connect parity intact. `make contract-test` covers this — it must stay green.

Phase 5 doesn't touch the server.

Phases 7-8 use the existing REST endpoints; Connect clients unaffected.

### Integration test scenarios

1. **Phase 4**: panic in audit closure → no orphan flag_version row (covered by new `TestWithAuditRollsBackOnPanic`).
2. **Phase 4**: serialization conflict during a publish → both inserts roll back; handler returns 409.
3. **Phase 5**: CLI publishes a TS flag → response carries `source_text` matching the file byte-for-byte → dashboard view route shows highlighted source.
4. **Phase 6**: Hurl PUT with malformed TS → 422 with line/column in details.
5. **Phase 8**: edit a TS flag in Monaco → save → reload view route → new source appears highlighted.

## Acceptance Criteria

### Functional

- [ ] `Store.PublishFlagVersionTx(ctx, q, params)` exists; `Store.PublishFlagVersionStandalone(ctx, params)` exists; old `Store.PublishFlagVersion` no longer exists (or is removed).
- [ ] REST + Connect handlers call `PublishFlagVersionTx` inside `WithAudit`'s closure with `q` plumbed through.
- [ ] `WithAudit` opens a serializable transaction.
- [ ] `TestWithAuditRollsBackOnPanic` exists and asserts no orphan row.
- [ ] CLI `falseflag config publish` sends `source_text` for `.ts`, `.json`, and `.cel` files.
- [ ] CLI tests cover both `.ts` and `.json` fixtures with `source_text` assertions.
- [ ] `tests/eval-corpus/26-typescript-rollout.json` and `27-typescript-nested-cel.json` exist with real TS `source_text`.
- [ ] `tests/hurl/14-typescript-publish.hurl` covers 201/422/back-compat/divergent-IR.
- [ ] Dashboard view route renders Shiki SSR HTML for TS/CEL/JSON sources when `source_text` is present; falls back to IR pretty-print with caption when absent.
- [ ] Edit link visible on view route header next to StrategyBadge.
- [ ] Dashboard edit route loads Monaco lazily; view route bundle does not include Monaco.
- [ ] Save submits to `PUT /v1/projects/{slug}/flags/{key}` with `source_text`; redirect to view route on 201.
- [ ] 422 responses surface as Monaco `setModelMarkers` annotations + a banner with the error text.

### Non-functional

- [ ] API binary remains CGO_ENABLED=0 (no goja/esbuild ABI surprises; both already pure Go).
- [ ] No new infra services. No base image changes.
- [ ] CI workflow auto-triggers remain off (slice 7b owns re-enabling them).
- [ ] Shiki highlighter is a process-singleton (~50ms cold start amortized).
- [ ] Monaco loads in its own chunk; visible in `vite build` output.

### Quality gates

- [ ] `go build ./...`, `go vet ./...`, `go test ./...` green.
- [ ] `FALSEFLAG_TEST_DATABASE_URL=… go test ./internal/store/... ./internal/server/...` green (live Postgres).
- [ ] `make generate-check` green (no drift).
- [ ] `make contract-test` green (REST↔Connect parity).
- [ ] `make conformance` green (27/27 fixtures, both runtimes).
- [ ] `make smoke` green (13 Hurl files).
- [ ] `pnpm -r typecheck`, `pnpm -r test`, `pnpm -r build`, `pnpm --dir js lint` all green.
- [ ] `make dashboard-e2e` runs the new edit-flag spec (or `test.skip`s gracefully on API-unreachable).
- [ ] `make bake-print` green.

## Success Metrics

- A user clicking on a `typescript`-strategy flag in the dashboard sees readable, highlighted TS source — not IR JSON. This is the conference-audience metric.
- A user clicking "Edit", changing a default value, and clicking Save sees the new value on the view route within ~1 second. This is the second conference-audience metric.
- `flag_versions` row + `audit_events` row are atomic — either both exist or neither does. Asserted by the new panic-rollback test.

## Dependencies & Prerequisites

Already present:
- Phase 1-3 of slice 8 (source_text column, compiler, API wiring).
- pnpm workspaces + Turborepo, Remix 2.15, React 18, Vite 5.4, Tailwind 3.4.
- `@playwright/test` 1.49 with Chromium binary (assumed installed locally — slice 5 noted it requires a one-time `playwright install chromium`).

New JS dependencies:
- `shiki@^3` (server-only).
- `@monaco-editor/react@^4` + `monaco-editor@^0.52` (client-only).

No new Go dependencies.

## Risk Analysis & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| `WithAudit` isolation level bump causes higher serialization-failure rate in normal traffic | low | publishes occasionally 409 | Acceptable for the demo. If observed, downgrade to `RepeatableRead`. |
| Hidden third caller of `PublishFlagVersion` breaks after rename | low | compile error | grep before refactor, update or migrate to `PublishFlagVersionStandalone`. |
| Monaco's CDN loader blocked in some browsers/networks | medium | edit page shows skeleton forever | Acceptable demo behavior. Followup: self-hosted workers via `loader.config({ paths: { vs } })`. |
| Monaco lands in the view route's chunk (regression) | low | bundle bloat on the SSR-mostly view path | Manual verification of `vite build` output in Phase 8. |
| Shiki SSR fails on cold start for a route render | low | view route 500 | Try/catch in the loader; fall back to IR JSON path. |
| `tests/hurl/14-typescript-publish.hurl` collides with future hurl file numbering | low | rename later | 14 is correct per current numbering (10 missing, 11/12/13 taken). |
| Playwright spec can't find Monaco selector reliably | medium | edit-flag spec flaky | Use Monaco's `[role="textbox"]` aria role; widen the `waitFor` timeout to 10s for the lazy chunk. |
| Snapshot republish not triggered on edit confuses demo viewers | medium | published edit doesn't appear in proxy `/v1/evaluate` | Out of scope per slice 8 plan. Add a toast on the view route after a save: "Run `falseflag snapshot publish` to deploy this change." |

## Resource Requirements

1 implementer, 2-2.5 working days. Postgres + compose stack running locally. Docker for `make smoke` / `make dashboard-e2e`. Playwright Chromium installed.

## Future Considerations

- Optimistic locking on flag_version edits (`If-Match` carrying current version).
- Snapshot republish on save (opt-in `?republish=true`).
- CodeMirror as a smaller-bundle alternative once the demo settles.
- Self-hosted Monaco workers for offline / kiosk environments.
- Form-based predicate builder for non-power-users.

## Documentation Plan

- `internal/config/typescript.go` top-of-file doc comment describing the pipeline.
- `js/packages/config-ts/src/index.ts` top-of-file doc comment noting server-side authoritative compile.
- No new READMEs (no precedent under `internal/` or `js/packages/`; slice 1-6 confirmed).

## Sources & References

### Predecessor plan

- **`docs/plans/2026-05-22-001-feat-server-ts-compile-and-edit-ui-plan.md`** — the original slice 8 plan covering all 9 phases. Phases 1-3 landed in commits `b3735de`, `7912d92`, `65d8f97`. This plan picks up at Phase 4.

### Backend references (file_path:line_number)

- `internal/store/flags.go:69-112` — current `PublishFlagVersion` opening its own serializable txn.
- `internal/store/audit.go:86-107` — current `WithAudit` opening default-isolation txn.
- `internal/store/types.go:35-47` — `store.FlagVersion` with `SourceText` (added Phase 1).
- `internal/store/integration_test.go:250-278` — `TestWithAuditRollsBackOnMutationFailure` pattern; new panic test goes after line 278.
- `internal/server/handlers/flags.go:109-206` — REST publish handler (line 182-201 is the `WithAudit` call).
- `internal/server/rpc/flags.go:92-162` — Connect publish handler (line 138-157 is the `WithAudit` call).
- `internal/config/typescript.go` — 285-line real compiler (Phase 2 landed).
- `internal/server/contract_test.go` — REST↔Connect parity suite (must stay green).

### CLI references

- `js/apps/cli/src/commands/config.ts:34-44` — `readSource` returning `string | object`.
- `js/apps/cli/src/commands/config.ts:106-123` — `save` action calling `publishFlagVersion`.
- `js/apps/cli/tests/config.test.ts:82-136` — `fakeFetch` test pattern, line ~132 has the body-assertion to extend.
- `js/packages/generated-client-ts/src/generated/api.ts:206-221` — `PublishFlagVersionRequest` type with `source_text?: string`.
- `js/packages/generated-client-ts/src/generated/api.zod.ts:135-139` — matching Zod schema.

### Test references

- `tests/eval-corpus/15-typescript-dsl-output.json` — existing TS fixture shape with `source_text`.
- `tests/eval-corpus/` — 25 existing fixtures; new ones become `26-typescript-rollout.json`, `27-typescript-nested-cel.json`.
- `tests/hurl/` — `00`–`09`, `11`, `12`, `13` taken; `10` missing; `14` is next free.
- `scripts/smoke.sh` — globs `tests/hurl/*.hurl`; no change needed.
- `internal/sdkgo/conformance_test.go:33-43` — `corpusFixture` doesn't decode `source_text`; new fixtures Just Work.
- `js/packages/sdk-js/tests/conformance.test.ts:15-26` — same.

### Dashboard references

- `js/apps/dashboard/app/routes/projects.$slug.flags.$key._index.tsx:31-55` — loader.
- `js/apps/dashboard/app/routes/projects.$slug.flags.$key._index.tsx:92-96` — `<pre>` target for replacement.
- `js/apps/dashboard/app/routes/projects.$slug.flags.$key._index.tsx:73` — StrategyBadge anchor for Edit link.
- `js/apps/dashboard/app/routes/projects.$slug.flags.$key.trace.tsx` — flat-routes sibling precedent for `.edit.tsx`.
- `js/apps/dashboard/app/lib/api.server.ts` — `withApiFetch` helper.
- `js/apps/dashboard/app/components/` — `ErrorBanner.tsx`, `Nav.tsx`, `StrategyBadge.tsx`, `TraceTree.tsx`.
- `js/apps/dashboard/tailwind.config.ts` — `strategy.{json,cel,typescript}` colors defined.
- `js/apps/dashboard/vite.config.ts` — current plugin set; add `optimizeDeps.include` here.
- `js/apps/dashboard/package.json` — Remix 2.15.2, React 18.3.1, Vite 5.4.11; neither shiki nor monaco present.
- `js/apps/dashboard/tsconfig.json` — `~/*` → `./app/*`.
- `js/apps/dashboard/playwright/helpers.ts:22-28` — graceful skip pattern.

### External references

- Shiki 3.x docs: `createHighlighter`, `createJavaScriptRegexEngine` (WASM-free path).
- `@monaco-editor/react` docs: `OnMount` callback, `editor.setModelMarkers`.
- Remix 2.x docs: `.client.tsx` and `.server.ts` file naming conventions.
- goja `Runtime.Interrupt()` — already used in `internal/config/typescript.go` (Phase 2).
- esbuild Go API `api.Build()` with `Stdin` + `External` — already used in `internal/config/typescript.go` (Phase 2).

### Related work

- Slice 7a (slow CI baseline) — shipped to `main` as `9a625f7`; CI auto-triggers off. Do **not** re-enable in this plan.
- Slice 7b (Depot acceleration) — deferred, maintainer-owned.
- Slice 9 (polish + demo script) — the next planned slice after slice 8 lands.
