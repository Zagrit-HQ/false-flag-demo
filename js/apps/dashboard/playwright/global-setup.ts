// Playwright globalSetup — runs once before any worker spawns.
// Seeds the 100-fixture edit-corpus (50 CEL + 50 TS) via the API.
// Each fixture owns its own project, so subsequent tests run fully
// parallel without cross-contamination.

import type { FullConfig } from "@playwright/test";

import { CEL_CORPUS } from "./corpus/cel";
import { seedAll } from "./corpus/setup";
import { TS_CORPUS } from "./corpus/typescript";

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
      `[playwright globalSetup] API not reachable at ${apiBase}: ${(e as Error).message}. Edit-corpus tests will fail-fast; run \`docker compose up -d --build\` first.`,
    );
    return;
  }
  const started = Date.now();
  await seedAll(all);
  console.log(
    `[playwright globalSetup] seeded ${all.length} corpus fixtures in ${
      Date.now() - started
    }ms`,
  );
}
