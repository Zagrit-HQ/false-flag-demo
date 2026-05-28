# Dashboard end-to-end tests

Playwright drives a real Chromium against a real Remix dev server
(which itself talks to the FalseFlag API). Slice 5 has one spec:
`dashboard.spec.ts` covers the "projects → flag → trace" happy path.

## Run it locally

```bash
# 1. compose: bring up the API + DB
make up
make seed         # populate the demo dataset

# 2. install Playwright's bundled Chromium (one-time)
cd js && pnpm --filter @falseflag/dashboard exec playwright install chromium

# 3. run the test
make dashboard-e2e
# or, equivalently:
cd js && pnpm --filter @falseflag/dashboard test:e2e
```

The Playwright config in `playwright.config.ts` spawns the dashboard's
`pnpm dev` for you via the `webServer` option. If you already have the
dashboard running, set `FALSEFLAG_DASHBOARD_URL=http://localhost:3000`
to skip the webServer launch.

## What the spec asserts

1. `/` redirects to `/projects`.
2. The Projects heading renders.
3. Clicking the first project surfaces the project stats grid.
4. Clicking the first flag in the preview list opens the flag detail
   page with the "Evaluate / trace →" CTA.
5. Submitting the default evaluation context renders a Decision panel.

If the API is unreachable, the test logs the error banner text and
skips downstream steps — the dashboard still renders a graceful empty
state, so the test won't false-fail in environments without compose.

## CI

Slice 7 owns the CI wiring. The expensive bits (Chromium download,
Remix SSR build) are exactly the surfaces Depot Cache accelerates;
slice 5 just makes them exist.
