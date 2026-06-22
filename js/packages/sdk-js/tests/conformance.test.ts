import { readFileSync, readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import { createClient } from "../src/client.js";
import type {
  Decision,
  DecisionReason,
  RulesTree,
  Strategy,
} from "../src/index.js";

interface Fixture {
  id: string;
  description: string;
  strategy: Strategy;
  ir: RulesTree;
  context: Record<string, unknown>;
  expected: {
    value: unknown;
    reason: DecisionReason;
    rule_id?: string;
  };
}

function corpusDir(): string {
  const here = dirname(fileURLToPath(import.meta.url));
  // js/packages/sdk-js/tests/ → repo root is 5 levels up.
  return join(here, "..", "..", "..", "..", "tests", "eval-corpus");
}

function loadCorpus(): Fixture[] {
  const dir = corpusDir();
  return readdirSync(dir)
    .filter((f) => f.endsWith(".json"))
    .sort()
    .map((f) => JSON.parse(readFileSync(join(dir, f), "utf8")) as Fixture);
}

describe("SDK conformance against tests/eval-corpus", () => {
  const fixtures = loadCorpus();

  it("loads at least 25 fixtures", () => {
    expect(fixtures.length).toBeGreaterThanOrEqual(25);
  });

  it("evaluates every fixture identically to the expected Decision", async () => {
    // Build a synthetic snapshot containing every fixture keyed by id.
    const flagsMap: Record<string, RulesTree> = {};
    for (const fx of fixtures) flagsMap[fx.id] = fx.ir;

    const snapshotBody = {
      id: "conformance-snapshot",
      project_id: "conformance-project",
      version: 1,
      created_at: "2026-05-20T00:00:00Z",
      compiled: { flags: flagsMap },
    };

    const fakeFetch: typeof fetch = async () =>
      new Response(JSON.stringify(snapshotBody), {
        status: 200,
        headers: { "content-type": "application/json" },
      });

    const client = createClient({
      baseUrl: "http://api",
      projectSlug: "conformance",
      fetch: fakeFetch,
      pollIntervalMs: 0,
      warn: () => {},
    });
    await client.start();
    expect(client.getSnapshot()).not.toBeNull();

    for (const fx of fixtures) {
      const got: Decision = await client.evaluate(fx.id, fx.context);
      expect(JSON.stringify(got.value), `${fx.id} value`).toBe(
        JSON.stringify(fx.expected.value),
      );
      expect(got.reason, `${fx.id} reason`).toBe(fx.expected.reason);
      if (fx.expected.rule_id) {
        expect(got.rule_id, `${fx.id} rule_id`).toBe(fx.expected.rule_id);
      }
    }
    client.stop();
  });
});
