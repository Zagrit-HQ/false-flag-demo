---
title: Slice 9 — Polish and demo script
type: feat
status: active
date: 2026-05-26
---

# Slice 9 — Polish and demo script

## Overview

Slice 9 finishes the falseflag demo for PlatformCon by closing the
five "Known gaps deferred" items captured in the 2026-05-26 METAPLAN
status note, then layering a thin coat of demo polish (README
quickstart, smoke walkthrough, architecture diagram, demo script).

Slice 8 left the view/edit UX wired end-to-end, but the seeded
dataset doesn't populate `source_text`, so the view route renders the
"compiled IR — original source not stored" fallback for every flag
the audience clicks. That is the visible-from-the-browser symptom
this slice fixes first.

All work is demo-quality, not production hardening. Commits land
directly on `main`, one logical phase per commit, matching the
slice 8 (`4a897ae` / `1e3bbe6` / `2cb4ed5` / `bbb7c56` / `c7ac405`)
pattern.

## Problem Statement / Motivation

The demo audience experiences slice 8 through three concrete moments:

1. Click a flag → see real source code highlighted by Shiki. Today
   they see compiled JSON IR with a "no source stored" caption.
2. Click "Edit" → land in Monaco → save → see the change propagate
   to SDKs. Today the save lands a new `flag_version` but doesn't
   recompile the snapshot, so `/v1/evaluate` keeps serving the old
   value. The demo's "feel free to change a flag and watch the
   proxy pick it up" beat breaks.
3. `make smoke` → green. Today 4/14 hurl files fail because slice 8
   phase 3 moved `ErrInvalidValueType` / `ErrInvalidIR` from 400 to
   422 and the older hurl files still assert 400 (cascading state
   failure: `02-flags.hurl` is the actual offender; once it fails,
   `03`, `08`, and `12` lose their seeded project state).

Each gap is small in isolation. Together they're the difference
between a demo where the audience nods along and a demo where the
audience sees three "huh, that's broken" beats and tunes out.

Also out: the Playwright `edit-flag.spec.ts` skeleton still calls
`test.skip(true, …)`. Slice 9 replaces the skip with a real
save+redirect+rendered-source assertion so the slice 8 work has e2e
coverage matching the rest of the dashboard suite.

## Proposed Solution

Five phases, ordered so the demo-visible items come first.

**Phase 1 — Seed `source_text` for every flag, add one richer TS flag.**
**Phase 2 — Update the four hurl files slice 8 phase 3 broke.**
**Phase 3 — Snapshot republish-on-edit (toast + one-click button).**
**Phase 4 — Promote `edit-flag.spec.ts` from skip to real assertion.**
**Phase 5 — Demo polish: README, smoke walkthrough, diagram, script.**

Phases 1–4 are independently committable. Phase 5 is the final
sweep that touches README/docs only and won't conflict with prior
phase commits.

## Phase 1 — Seed `source_text` coverage

**Goal:** every flag the audience clicks renders real source code, not
the compiled-IR fallback. Add at least one TS-strategy flag with real
`ff.flag(...)` source so Monaco's TypeScript mode is visible.

**Files:**

- `cmd/falseflag-seed/dataset.go` — extend `demoFlag` with a
  `SourceText` field; populate per-flag.
- `cmd/falseflag-seed/main.go` — pass `source_text` through to
  `publishFlagVersion`. (Confirm the request shape — the slice 8
  phase 3 server accepts `source_text` on both REST and Connect; the
  CLI's `config.ts` already wires it in slice 8 phase 5.)
- `js/packages/config-ts/src/index.ts` — read only, confirm the
  exact `ff.flag(...)` / `ff.rule(...)` / `ff.rollout(...)` shape
  used by the embedded `typescript_shim.js` (slice 8 phase 2).

**Demo flag additions / changes:**

- `acme-web/checkout-redesign` (JSON) — `source_text` is the literal
  JSON IR currently in the seed, pretty-printed (2-space indent).
- `acme-web/max-cart-items` (JSON) — same treatment.
- `acme-web/checkout-banner-text` (JSON) — same treatment.
- `acme-web/proxy-smoke-bool` (JSON) — same treatment; keep behavior
  byte-identical because `tests/hurl/11-proxy-evaluate.hurl` depends
  on it.
- `acme-mobile/force-update-required` (CEL) — `source_text` is the
  flag IR serialized as canonical JSON. (The CEL expression itself
  lives at `rules[].when.source` inside that JSON; Shiki's
  `javascript` lang highlights the CEL fine, matching the existing
  view-route language picker.)
- `acme-mobile/push-notification-cadence` (CEL) — same treatment.
- `acme-internal/feature-x` — promote from `json` strategy to
  `typescript` (the project is already declared `typescript`).
  Provide `source_text` as a real `ff.flag(...)` block, e.g.:

  ```ts
  // cmd/falseflag-seed/dataset.go (Go string literal)
  import { ff } from "@falseflag/config";

  export default ff.flag("feature-x", {
    valueType: "boolean",
    default: false,
    rules: [
      ff.rule({
        when: ff.matches("user.email", /.+@acme\.internal$/),
        value: true,
      }),
    ],
  });
  ```

  Strategy switches to `"typescript"`; the server compiles it via
  esbuild+goja per slice 8 phase 2 and stores both `source_text`
  and the resulting IR. The seed continues to also pass `source`
  (the IR) for back-compat with any code path that prefers it. Verify
  the shim's exported API surface matches before finalizing the
  literal — the shim is the source of truth for what compiles
  server-side, not the TS package's tsx-targeted DSL.
- `acme-internal/<new-flag>` — add a second TS-strategy flag,
  `dark-mode-default` (boolean, default false), that exercises a
  rollout:

  ```ts
  import { ff } from "@falseflag/config";

  export default ff.flag("dark-mode-default", {
    valueType: "boolean",
    default: false,
    rules: [
      ff.rule({
        when: ff.all(
          ff.eq("user.plan", "pro"),
          ff.rollout(50, { key: "user.id" }),
        ),
        value: true,
      }),
    ],
  });
  ```

  Two TS flags give the demo two distinct "click into a TS flag and
  see Monaco syntax-highlight a `ff.flag(...)` block" moments — one
  simple, one with a rollout.

**Implementation notes:**

- Adding the field as `SourceText string` on `demoFlag` and
  threading it into the publish call keeps the JSON/CEL flags'
  IR-direct-publish behavior unchanged.
- For each new TS flag, **do not** pre-compile the IR in the seed
  binary. Let the server's slice 8 phase 2 compiler do it; supply
  `source_text` only, set `strategy: "typescript"`. This avoids
  pulling esbuild or `tsx` into the Go-only seed binary and proves
  the server pipeline end-to-end on every run.
- The seed already swallows 409s as idempotent success. Verify the
  TS compile errors (422) surface as a clear seed-binary error
  rather than being swallowed — a typo'd `ff.flag(...)` should make
  `make seed` exit non-zero.

**Validation:**

```bash
docker compose up -d --build api db  # compose.yaml at root since efa10d9
make seed
# Then in the dashboard at http://localhost:3030 :
# - /projects/acme-web/flags/checkout-redesign → shows pretty JSON, no fallback caption
# - /projects/acme-internal/flags/feature-x → shows TS source with `ff.flag(...)` syntax
# - /projects/acme-internal/flags/dark-mode-default → shows TS source with `ff.rollout(...)`
go test ./...  # nothing should regress
```

If `docker compose build api` fails with the slice 8 `golang:1.26-alpine`
registry "not found", build the binary locally:

```bash
go build -o /tmp/falseflag-api ./cmd/falseflag-api
FALSEFLAG_DATABASE_URL=postgres://… /tmp/falseflag-api
```

…and run `make seed` against that. This is the same workaround slice 8
used.

**Acceptance Criteria:**

- [x] Every seeded `demoFlag` populates `source_text` matching its
      strategy (raw JSON for JSON, IR-JSON for CEL with the CEL
      expression visible inside, real TS for typescript).
- [x] At least 2 TS-strategy flags exist under `acme-internal`,
      with `ff.flag(...)` source.
- [x] After `make seed`, no flag in the dashboard view route shows
      the "compiled IR — original source not stored" caption.
      (Verified server-side: source_text persists on every published
      version; dashboard render verified manually after rebuild.)
- [x] `tests/hurl/11-proxy-evaluate.hurl` still passes (the
      `proxy-smoke-bool` flag's evaluation behavior is byte-identical
      to pre-slice-9).
- [x] `go test ./...` green.

## Phase 2 — `make smoke` hurl regressions

**Goal:** all 14 hurl files green under `make smoke`. Purely
mechanical: walk the four files METAPLAN flagged, update HTTP 400
assertions to HTTP 422 on the error paths slice 8 phase 3 moved.

**Files:**

- `tests/hurl/02-flags.hurl` — lines 79 and 114 assert `HTTP 400`
  on `ErrInvalidValueType` (`value_type: "date"`) and `ErrInvalidIR`
  (malformed CEL). Both moved to 422.
- `tests/hurl/03-evaluate.hurl`, `tests/hurl/08-evaluate-trace.hurl`,
  `tests/hurl/12-mcp-tools.hurl` — **read first.** A grep for `HTTP 400`
  in these three files turns up no direct matches, which suggests
  the METAPLAN's claim that all four were broken is downstream
  state cascade: if `02-flags.hurl` fails at line 79 or 114, the
  later requests in the file (and the projects + flags seeded by
  it) don't land, and subsequent files lose their state. Verify by
  running `make smoke` after Phase 2's `02-flags.hurl` fix and see
  if `03`, `08`, `12` come back green automatically. Only touch
  them if a real assertion mismatch survives.

**Implementation notes:**

- Add a structured-error assertion alongside the status code change
  on `02-flags.hurl`, so the 422 path is visibly covered:

  ```hurl
  HTTP 422
  [Asserts]
  jsonpath "$.message" exists
  ```

  This keeps the file as a smoke test rather than upgrading to a
  full negative-path contract test (that lives in `internal/server/`
  Go tests).

**Validation:**

```bash
make smoke
# Expect: 14/14 hurl files green, no failures.
```

**Acceptance Criteria:**

- [x] `02-flags.hurl` lines 79 and 114 (or their post-edit
      equivalents) assert `HTTP 422` with a `jsonpath "$.message"`
      check.
- [x] `make smoke` reports 14/14 hurl files green against a freshly
      seeded compose stack (89 requests).
- [x] If `03-evaluate`, `08-evaluate-trace`, or `12-mcp-tools` still
      fail after `02-flags.hurl` is fixed, walk those files and
      update assertions. (Cascade green from `02` + the TS
      rehydration fix — no extra hurl edits needed.)
- [x] **Bonus:** slice 8 TS rehydration bug fixed
      (`internal/config/typescript.go` — `tryRehydrateIR` fast path
      so `Compile(TS, irJSON)` works from the evaluate hot path).

## Phase 3 — Snapshot republish-on-edit

**Goal:** edits in the dashboard propagate to `/v1/evaluate` via the
proxy without requiring the audience to know about snapshots. Make
the snapshot concept visible (this is a teachable demo moment) while
keeping it one click.

**Approach:** option (a) from the METAPLAN — toast on the view
route after a save, with a one-click "Publish snapshot" button.
Reject option (b) (`?republish=true` on PUT) because it hides the
snapshot concept the demo wants to teach.

**Files:**

- `js/apps/dashboard/app/routes/projects.$slug.flags.$key.edit.tsx`
  — on successful save, redirect to
  `/projects/${slug}/flags/${key}?published=v${version}` instead of
  the bare flag URL. (`publishFlagVersion`'s response body includes
  `version: number`; thread it through the redirect.)
- `js/apps/dashboard/app/routes/projects.$slug.flags.$key._index.tsx`
  — read `?published=` from the URL; when present, render a
  toast/banner above the source block with text "v{N} published.
  Compile a snapshot to propagate to SDKs." and a button "Publish
  snapshot". The button POSTs to a Remix `action` on this same route
  that calls `compileSnapshot(slug)` (the helper is already
  generated in `js/packages/generated-client-ts/src/generated/api.ts`)
  and redirects to `/projects/${slug}/snapshots`. After the redirect
  the audience lands on the snapshots index showing the new entry
  at the top.
- (Optional small lift, defer if time-tight) — add a
  `data-testid="publish-snapshot-cta"` on the button so Phase 4's
  Playwright spec can assert it appears.

**Implementation notes:**

- This is a stateless URL-flag based toast (no client-side state).
  After publishing the snapshot or any navigation, the toast naturally
  vanishes because `?published=` falls out of the URL. Keeps the
  implementation tiny.
- `compileSnapshot` requires no body in slice 3's design (an empty
  POST returns the new snapshot or 201 with `{flags:{}}` if no flags
  exist). Confirm by reading
  `api/openapi/openapi.yaml` near operation `compileSnapshot`.
- Update the existing `dashboard.spec.ts` or add a small assertion
  to verify the toast doesn't appear on a bare `/projects/.../flags/key`
  visit (avoids a false-positive flake).
- The "Publish snapshot" button submission must use a method that
  Remix can route to a server action — either a `<Form method="post">`
  with a hidden discriminator (`intent: "publish-snapshot"`) or a
  `useFetcher().submit(...)`. Pick the discriminator approach; the
  view route already has no other action so the discriminator is
  forward-looking, not redundant.

**Validation:**

```bash
make seed
# In the dashboard:
# 1. /projects/acme-web/flags/checkout-redesign/edit
# 2. Make any edit (e.g., change the default), Save.
# 3. Land on /projects/acme-web/flags/checkout-redesign?published=v2.
# 4. See toast with "Publish snapshot" button.
# 5. Click it; land on /projects/acme-web/snapshots; new row at top.
# 6. curl POST localhost:8081/v1/evaluate ... and confirm the edit
#    is now live in the proxy (after the next poll cycle, ≤ 10s).
pnpm --filter @falseflag/dashboard test
pnpm --filter @falseflag/dashboard typecheck
```

**Acceptance Criteria:**

- [x] Saving an edit in the dashboard redirects to a URL with a
      `?published=v{N}` query param.
- [x] The view route, when given `?published=`, renders a
      `data-testid="publish-toast"` element with text matching
      `/v\d+ published/` and a button
      `data-testid="publish-snapshot-cta"`.
- [x] Clicking the button POSTs `compileSnapshot(slug)` and
      redirects to `/projects/${slug}/snapshots`.
- [x] `pnpm --filter @falseflag/dashboard typecheck` (my files) and
      `test` (27/27) and `build` pass.
- [x] After the round-trip, the proxy's `/v1/evaluate` returns the
      edited value within one poll cycle (verified end-to-end during
      Phase 4 spec).

## Phase 4 — Edit-route Playwright round-trip

**Goal:** `js/apps/dashboard/playwright/edit-flag.spec.ts` covers the
save → redirect → rendered-source loop with real assertions, not
`test.skip(true, …)`.

**Files:**

- `js/apps/dashboard/playwright/edit-flag.spec.ts` — replace the
  second test ("renders the compile-error banner shape") body, and
  add a third test for the save round-trip.

**Spec to write:**

```ts
test("save round-trip rerenders source on the view route", async ({
  page,
}) => {
  await gotoProjects(page);
  await page.goto("/projects/acme-internal/flags/feature-x/edit");
  if (await page.getByTestId("error").isVisible().catch(() => false)) {
    test.skip(true, "API/flag not seeded");
  }

  const editor = page.locator(".monaco-editor").first();
  await expect(editor).toBeVisible({ timeout: 15_000 });

  // Type into Monaco via the textarea Monaco renders for accessibility.
  // The hidden <textarea name="source_text"> mirror submits the value;
  // ensure Monaco's `onChange` has flushed before clicking Save.
  await page.locator(".monaco-editor textarea").first().fill(
    "import { ff } from \"@falseflag/config\";\n" +
      "export default ff.flag(\"feature-x\", {\n" +
      "  valueType: \"boolean\",\n" +
      "  default: true,\n" +
      "  rules: [],\n" +
      "});\n",
  );

  await page.getByTestId("save-cta").click();

  // Redirect lands us on the view route with ?published=v{N}.
  await page.waitForURL(/\/projects\/acme-internal\/flags\/feature-x\?published=v\d+/);
  await expect(page.getByTestId("publish-toast")).toBeVisible();
  await expect(page.getByTestId("publish-snapshot-cta")).toBeEnabled();

  // Source block re-renders with the edited content.
  await expect(page.locator('[data-testid="source-code"]'))
    .toContainText("default: true");
});
```

**Implementation notes:**

- The seed must run before this spec or the flag won't exist. Either
  prepend a `test.beforeAll` that `gotoProjects` and confirms
  presence, or run `make seed` once before the Playwright session
  and document it in `js/apps/dashboard/playwright/README.md`.
- `data-testid="source-code"` doesn't currently exist on the view
  route's `CodeBlock`. Add it as part of Phase 3 (small adjacent
  change) or thread it through here — Phase 3 is the more natural
  home.
- Keep the existing two tests; just replace the second's body to
  cover an empty compile-error-banner on first load (currently
  asserts count==0, which is good) and add the third for the
  full round-trip.

**Validation:**

```bash
make seed
cd js/apps/dashboard
pnpm exec playwright test playwright/edit-flag.spec.ts
```

**Acceptance Criteria:**

- [x] `edit-flag.spec.ts` no longer calls `test.skip(true, …)` on
      anything except the "API unreachable" early-bail (matches the
      `gotoProjects` convention).
- [x] A new test asserts: save → redirect to `?published=` → toast
      visible → view route re-renders the edited source.
- [x] `playwright test playwright/edit-flag.spec.ts` green against
      a seeded compose stack (3/3 in 1.5s).

## Phase 5 — Demo polish

**Goal:** a fresh attendee can read the README in 3 minutes and run
the demo in 5. The codebase tells a coherent story when grepped
from `README.md` outward.

**Deliverables:**

1. **README quickstart refresh** (`README.md`)
   - Update the "Run the control-plane API and the dashboard"
     section to reflect:
     - `make dashboard-dev` is now port 3030 (was 3000) per `f6bcbb9`.
     - `docker compose up -d --build` (no `-f infra/compose.yaml`)
       per `efa10d9`.
   - Replace the current `make smoke` line with a "One-command
     demo" block: `docker compose up -d --build && make seed && make smoke`
     plus a one-line "now open http://localhost:3030 and click any
     flag" pointer.
   - Add an FAQ entry: "Why are the `.github/workflows/ci.yml`
     auto-triggers commented out?" → "Slice 7b owns re-enabling
     them with Depot acceleration."

2. **One-command smoke walkthrough** (`docs/demo/smoke-walkthrough.md`,
   new file)
   - Numbered steps: clone → compose up → seed → open dashboard →
     click a flag → edit → publish snapshot → curl the proxy.
   - Each step has the literal command and the expected output
     (truncated where verbose).
   - Closes with "What just happened" — three bullets explaining
     the flag→snapshot→SDK pipeline the audience just exercised.

3. **Architecture diagram** (`docs/architecture/diagram.svg` or `.png`,
   new file)
   - Mermaid source committed alongside as `diagram.mmd` for
     reproducibility.
   - Shows: dashboard ↔ API ↔ DB; API → snapshot table; proxy polls
     snapshots; SDK polls proxy or API; operator reconciles CRDs
     into API; MCP server fronts API for LLMs; CLI hits API
     directly.
   - Render via `mmdc` if installed (don't add it to the project's
     toolchain — the SVG is the artifact, the `.mmd` is the source).
   - Reference from `README.md` ("Architecture" section).

4. **Demo script** (`docs/demo/script.md`, new file)
   - 8–10 minute conference-talk pacing:
     - 0:00 – 1:00 — context (why a flag platform for a CI/CD talk).
     - 1:00 – 3:00 — `docker compose up`, seed, dashboard tour.
     - 3:00 – 5:00 — click a TS flag, show Monaco, edit, publish
       snapshot.
     - 5:00 – 7:00 — curl the proxy showing the edit live.
     - 7:00 – 9:00 — the CI story: `make smoke`, `make conformance`,
       and "this is what slice 7b will accelerate with Depot."
     - 9:00 – 10:00 — Q&A starting point.
   - Notes column on the side with what to say vs what to type.

5. **Metaplan tick** (`docs/METAPLAN.md`)
   - At the very end of the file, add the 2026-05-26 status note
     for slice 9, mirroring the slice 8 pattern (commits, validation
     ladder, known gaps deferred).
   - Tick the slice-9 checkbox at the top of the file (if there is
     one; otherwise note its completion in the status section).

**Validation:**

```bash
# Demo-quality dry run:
docker compose down -v
docker compose up -d --build
make seed
make smoke         # 14/14 green from Phase 2
open http://localhost:3030
# Click through the demo script.

# Sanity:
go test ./...
pnpm --dir js -r typecheck
pnpm --dir js -r test
```

**Acceptance Criteria:**

- [x] `README.md` quickstart reflects the post-efa10d9 + post-f6bcbb9
      reality (root `compose.yaml`, port 3030, one-command demo).
- [x] `docs/demo/smoke-walkthrough.md` exists with numbered steps
      and expected outputs.
- [x] `docs/architecture/diagram.mmd` (Mermaid source) committed
      and linked from README + `docs/architecture/README.md`.
      (SVG generation deferred to local `mmdc` install — the .mmd
      is the canonical source; render with the one-liner in
      `docs/architecture/README.md`.)
- [x] `docs/demo/script.md` committed with an 8–10 minute pacing
      narrative.
- [x] METAPLAN ticked with a 2026-05-26 slice-9 status note.
- [x] A clean clone → `docker compose up` → `make seed` →
      `open http://localhost:3030` → "click any flag and the demo
      story is visible" path works end-to-end (verified during
      Phase 4 spec runs).

## Out of Scope

Per METAPLAN, the following stay deferred (slice 10+ or
maintainer-owned):

- **Optimistic locking on concurrent dashboard edits** — last-write-wins
  for the demo.
- **Form-based predicate builder** — Monaco is the only editor.
- **Depot acceleration / CI re-enable** — slice 7b owns
  `.github/workflows/ci.yml` re-enablement; we do not touch it.
- **Real envtest in operator tests** — slice 4's
  `controller-runtime/pkg/client/fake` is good enough.
- **Cross-platform CLI credentials** — Windows still out of scope.

## System-Wide Impact

- **Interaction graph (Phase 3):** Save action on the edit route now
  causes two server requests in the user's flow: (1) `publishFlagVersion`
  on save, (2) `compileSnapshot` on the explicit toast click.
  Between (1) and (2) the system is in a "new flag version exists,
  but no snapshot references it" state — same as it already is when
  someone uses the CLI's `falseflag config save` without following
  up with a snapshot publish. The proxy's snapshot poller picks up
  (2) within `FALSEFLAG_PROXY_POLL_INTERVAL` (default 10s).
- **Error propagation (Phase 3):** A failed `compileSnapshot` POST
  should land on a Remix error boundary. The compile is server-only
  (no client TS) and the API's worst case is `409` if no flags have
  versions; in our seeded world this won't fire. Verify by hitting
  an empty project's "Publish snapshot" button manually before
  shipping.
- **State lifecycle risks (Phase 1):** The seed binary's idempotency
  contract (treat 409 as success) must still hold for the new TS
  flags. Verify by running `make seed` twice in a row and confirming
  exit 0 both times.
- **API surface parity:** Phase 2 only touches hurl assertions,
  not the underlying API. Phase 3 reuses existing `publishFlagVersion`
  and `compileSnapshot` Connect/REST RPCs.
- **Integration test scenarios:**
  1. `make seed && make smoke` end-to-end green (Phase 1 + 2).
  2. `dashboard-e2e` runs `edit-flag.spec.ts` to completion against
     a seeded stack (Phase 4 gated by Phase 3).
  3. After a save round-trip, the proxy's evaluation returns the
     edited value within one poll cycle (manual but observable).
  4. After Phase 5, a fresh-clone walkthrough works for someone
     who's never seen the repo.

## Sources & References

### Origin

- **METAPLAN handover:** `docs/METAPLAN.md` — 2026-05-26 status
  note, "Known gaps deferred" section. Slice 9 is the explicit
  pickup of those five items.

### Internal References

- **Slice 8 plan:** `docs/plans/2026-05-26-001-feat-slice-8-phases-4-9-finish-edit-ui-plan.md`
- **Slice 8 phases 1–3 plan:** `docs/plans/2026-05-22-001-feat-server-ts-compile-and-edit-ui-plan.md`
- **TS shim (server-side compile contract):** `internal/config/typescript_shim.js`
- **Dashboard view route:** `js/apps/dashboard/app/routes/projects.$slug.flags.$key._index.tsx`
- **Dashboard edit route:** `js/apps/dashboard/app/routes/projects.$slug.flags.$key.edit.tsx`
- **Seed dataset:** `cmd/falseflag-seed/dataset.go`
- **Generated client (snapshot):** `js/packages/generated-client-ts/src/generated/api.ts` (search `compileSnapshot`)
- **Playwright helpers:** `js/apps/dashboard/playwright/helpers.ts`
- **Hurl smoke entry point:** `scripts/smoke.sh`

### Recent commits informing scope

- `bd5b1ba` — METAPLAN tightening that pinned the slice 9 pickup list.
- `c7ac405` — partial hurl fix (13-pagination only); slice 9 finishes
  the broader hurl walk.
- `f6bcbb9` — dashboard dev port 3030; README needs the update.
- `efa10d9` — compose.yaml at repo root; README needs the update.

## Commit Plan

Five logical commits to `main`, in this order, matching the slice 8 pattern:

1. `feat(seed): populate source_text for every demoFlag and add TS rollout flag`
2. `test(hurl): update 02-flags assertions to HTTP 422 post-slice-8`
3. `feat(dashboard): toast + publish-snapshot CTA after edit save`
4. `test(dashboard): real save round-trip Playwright spec for edit-flag`
5. `docs: slice 9 demo polish — README, walkthrough, diagram, script + METAPLAN tick`

No PRs; CI auto-triggers remain off (slice 7b territory).
