import { defineConfig } from "@playwright/test";

const PORT = Number(process.env.FALSEFLAG_DASHBOARD_PORT ?? 3030);

export default defineConfig({
  testDir: "./playwright",
  globalSetup: "./playwright/global-setup.ts",
  // globalSetup owns the shared dashboard baseline plus per-fixture
  // corpus projects. Specs either read that baseline or mutate their
  // own isolated project, so fullyParallel is safe.
  fullyParallel: true,
  workers: process.env.CI ? 1 : undefined,
  reporter: process.env.CI ? "list" : "line",
  timeout: 60_000,
  use: {
    baseURL: process.env.FALSEFLAG_DASHBOARD_URL ?? `http://127.0.0.1:${PORT}`,
    headless: true,
    trace: "on-first-retry",
  },
  webServer: process.env.FALSEFLAG_DASHBOARD_URL
    ? undefined
    : {
        command: `pnpm dev --port ${PORT}`,
        url: `http://127.0.0.1:${PORT}`,
        reuseExistingServer: !process.env.CI,
        timeout: 120_000,
        env: {
          FALSEFLAG_API_BASE_URL:
            process.env.FALSEFLAG_API_BASE_URL ?? "http://127.0.0.1:8080",
        },
      },
});
