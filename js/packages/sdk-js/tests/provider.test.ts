import { describe, expect, it, vi } from "vitest";

import { createClient } from "../src/client.js";
import type { RulesTree } from "../src/ir.js";
import { createProvider } from "../src/provider.js";

const ir: Record<string, RulesTree> = {
  "checkout-redesign": {
    value_type: "boolean",
    default: false,
    rules: [
      {
        id: "r1",
        when: { kind: "eq", attr: "user.plan", value: "pro" },
        value: true,
      },
    ],
  },
  greeting: {
    value_type: "string",
    default: "hello",
    rules: [],
  },
  "max-items": {
    value_type: "number",
    default: 10,
    rules: [],
  },
  badges: {
    value_type: "object",
    default: { enabled: false },
    rules: [],
  },
};

function jsonResponse(body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "content-type": "application/json" },
  });
}

function makeProvider() {
  const fetchMock = vi.fn().mockResolvedValue(
    jsonResponse({
      id: "11111111-1111-1111-1111-111111111111",
      version: 1,
      created_at: "2026-05-20T12:00:00Z",
      compiled: { flags: ir },
    }),
  );
  const client = createClient({
    baseUrl: "http://api",
    projectSlug: "demo",
    fetch: fetchMock,
    pollIntervalMs: 0,
  });
  const provider = createProvider({
    baseUrl: "http://api",
    projectSlug: "demo",
    client,
  });
  return { provider, client };
}

describe("FalseFlagProvider", () => {
  it("reports metadata", () => {
    const { provider } = makeProvider();
    expect(provider.metadata.name).toBe("falseflag");
  });

  it("resolves boolean evaluations against the snapshot", async () => {
    const { provider, client } = makeProvider();
    await client.start();
    const d = await provider.resolveBooleanEvaluation(
      "checkout-redesign",
      false,
      { user: { plan: "pro" } },
    );
    expect(d.value).toBe(true);
    expect(d.reason).toBe("rule_matched");
  });

  it("resolves string evaluations and falls back to default on type mismatch", async () => {
    const { provider, client } = makeProvider();
    await client.start();
    const ok = await provider.resolveStringEvaluation("greeting", "x", {});
    expect(ok.value).toBe("hello");
    expect(ok.reason).toBe("default");
    const mismatch = await provider.resolveStringEvaluation(
      "max-items",
      "fallback",
      {},
    );
    expect(mismatch.value).toBe("fallback");
    expect(mismatch.reason).toBe("type_mismatch");
  });

  it("resolves number evaluations", async () => {
    const { provider, client } = makeProvider();
    await client.start();
    const d = await provider.resolveNumberEvaluation("max-items", -1, {});
    expect(d.value).toBe(10);
  });

  it("resolves object evaluations", async () => {
    const { provider, client } = makeProvider();
    await client.start();
    const d = await provider.resolveObjectEvaluation("badges", {}, {});
    expect(d.value).toEqual({ enabled: false });
  });
});
