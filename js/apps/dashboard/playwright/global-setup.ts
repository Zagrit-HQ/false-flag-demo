// Playwright globalSetup — runs once before any worker spawns.
// Seeds both the shared dashboard baseline used by navigation/read-only
// specs and the isolated 100-fixture edit corpus.

import type { FullConfig } from "@playwright/test";

import { CEL_CORPUS } from "./corpus/cel";
import { seedAll } from "./corpus/setup";
import { TS_CORPUS } from "./corpus/typescript";
import { seedDemoDashboard } from "./demo/setup";

export default async function globalSetup(_: FullConfig): Promise<void> {
  const all = [...CEL_CORPUS, ...TS_CORPUS];
  // Bail early with a clear message if the API isn't reachable —
  // matches the gotoProjects() graceful-skip convention.
  const apiBase = process.env.FALSEFLAG_API_BASE_URL ?? "http://localhost:8080";
  try {
    const r = await fetch(`${apiBase}/healthz`);
    if (!r.ok) throw new Error(`API ${apiBase} returned ${r.status}`);
  } catch (e) {
    console.warn(
      `[playwright globalSetup] API not reachable at ${apiBase}: ${(e as Error).message}. Fixture setup skipped; run \`docker compose up -d --build\` first.`,
    );
    return;
  }
  const started = Date.now();
  await seedDemoDashboard();
  await seedAll(all);
  console.log(
    `[playwright globalSetup] seeded dashboard baseline + ${all.length} corpus fixtures in ${
      Date.now() - started
    }ms`,
  );
}
