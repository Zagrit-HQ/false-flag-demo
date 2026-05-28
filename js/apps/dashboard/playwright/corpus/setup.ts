// Per-fixture API setup: creates a unique project + flag and
// publishes the fixture's setupSource so the edit page loads with
// the right strategy. Idempotent on 409.

import type { CorpusFixture } from "./types";

const API_BASE = process.env.FALSEFLAG_API_BASE_URL ?? "http://localhost:8080";

async function api(
  method: string,
  path: string,
  body?: unknown,
): Promise<{ status: number; data: unknown }> {
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers: {
      "content-type": "application/json",
      "x-actor": "playwright/corpus",
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text();
  const data = text ? safeJSON(text) : null;
  return { status: res.status, data };
}

function safeJSON(s: string): unknown {
  try {
    return JSON.parse(s);
  } catch {
    return s;
  }
}

/** Project slug derived from fixture id. Kept stable across runs so
 *  re-runs are idempotent (409 on the create). */
export function projectFor(f: CorpusFixture): string {
  return `e2e-corpus-${f.id}`;
}

/** Flag key within the per-fixture project. Constant since each
 *  fixture owns its own project. */
export const FLAG_KEY = "corpus";

/** Provision one fixture: project → flag → setup publish. Idempotent. */
export async function seedFixture(f: CorpusFixture): Promise<void> {
  const slug = projectFor(f);

  const projRes = await api("POST", "/v1/projects", {
    slug,
    display_name: slug,
    config_strategy: f.strategy,
  });
  if (
    projRes.status !== 201 &&
    projRes.status !== 200 &&
    projRes.status !== 409
  ) {
    throw new Error(
      `seedFixture(${f.id}): create project HTTP ${projRes.status}: ${JSON.stringify(projRes.data)}`,
    );
  }

  const flagRes = await api("POST", `/v1/projects/${slug}/flags`, {
    key: FLAG_KEY,
    name: f.id,
    value_type: f.valueType,
    default_value: f.defaultValue,
  });
  if (
    flagRes.status !== 201 &&
    flagRes.status !== 200 &&
    flagRes.status !== 409
  ) {
    throw new Error(
      `seedFixture(${f.id}): create flag HTTP ${flagRes.status}: ${JSON.stringify(flagRes.data)}`,
    );
  }

  const versionRes = await api(
    "PUT",
    `/v1/projects/${slug}/flags/${FLAG_KEY}`,
    {
      strategy: f.strategy,
      source: f.setupIR,
      source_text: f.setupSource,
    },
  );
  if (versionRes.status !== 200 && versionRes.status !== 201) {
    throw new Error(
      `seedFixture(${f.id}): publish setup HTTP ${versionRes.status}: ${JSON.stringify(versionRes.data)}`,
    );
  }
}

/** Seed all fixtures sequentially. Each fixture is 3 cheap API calls
 *  (~3ms total round-trip locally) so 100 fixtures finish in <500ms
 *  — well under any reasonable globalSetup budget. Parallel batches
 *  triggered undici keepAlive pool exhaustion (HeadersTimeoutError)
 *  even at batch sizes of 10; the work isn't worth the complexity. */
export async function seedAll(fixtures: CorpusFixture[]): Promise<void> {
  for (const f of fixtures) {
    await seedFixture(f);
  }
}
