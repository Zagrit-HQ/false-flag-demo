import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { createClient } from "../src/client.js";
import type { RulesTree } from "../src/ir.js";

const SNAPSHOT_URL = /\/v1\/projects\/demo\/snapshots\/latest$/;

const flagsIR: Record<string, RulesTree> = {
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
  "max-items": {
    value_type: "number",
    default: 10,
    rules: [],
  },
};

function snapshotBody(version = 1) {
  return {
    id: "11111111-1111-1111-1111-111111111111",
    version,
    created_at: "2026-05-20T12:00:00Z",
    compiled: { flags: flagsIR },
    project_id: "22222222-2222-2222-2222-222222222222",
  };
}

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

describe("snapshot polling client", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("populates the snapshot cache on start()", async () => {
    const fetchMock = vi.fn().mockImplementation((input: string | URL) => {
      if (String(input).match(SNAPSHOT_URL)) {
        return Promise.resolve(jsonResponse(snapshotBody(7)));
      }
      return Promise.resolve(new Response(null, { status: 404 }));
    });

    const c = createClient({
      baseUrl: "http://api",
      projectSlug: "demo",
      fetch: fetchMock,
      pollIntervalMs: 1000,
    });
    await c.start();
    const snap = c.getSnapshot();
    expect(snap).not.toBeNull();
    expect(snap?.version).toBe(7);
    expect(Object.keys(snap?.flags ?? {})).toContain("checkout-redesign");
    c.stop();
  });

  it("serves evaluations from the snapshot cache without further fetches", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(snapshotBody(2)));
    const c = createClient({
      baseUrl: "http://api",
      projectSlug: "demo",
      fetch: fetchMock,
      pollIntervalMs: 0, // no background poll
    });
    await c.start();
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const d = await c.evaluate("checkout-redesign", { user: { plan: "pro" } });
    expect(d.value).toBe(true);
    expect(d.reason).toBe("rule_matched");

    const d2 = await c.evaluate("max-items", { user: { plan: "free" } });
    expect(d2.value).toBe(10);
    expect(d2.reason).toBe("default");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    c.stop();
  });

  it("keeps last-good snapshot on subsequent poll failures", async () => {
    let call = 0;
    const fetchMock = vi.fn().mockImplementation(() => {
      call++;
      if (call === 1) return Promise.resolve(jsonResponse(snapshotBody(3)));
      return Promise.resolve(new Response("boom", { status: 500 }));
    });
    const warn = vi.fn();
    const c = createClient({
      baseUrl: "http://api",
      projectSlug: "demo",
      fetch: fetchMock,
      pollIntervalMs: 1000,
      warn,
    });
    await c.start();
    expect(c.getSnapshot()?.version).toBe(3);

    await vi.advanceTimersByTimeAsync(1500);
    expect(warn).toHaveBeenCalled();
    expect(c.getSnapshot()?.version).toBe(3); // last good preserved
    c.stop();
  });

  it("falls back to per-flag fetch when start() has not run", async () => {
    const fetchMock = vi.fn().mockImplementation((input: string | URL) => {
      const url = String(input);
      if (url.endsWith("/flags/checkout-redesign")) {
        return Promise.resolve(
          jsonResponse({
            flag: { value_type: "boolean" },
            latest_version: {
              version: 9,
              compiled: flagsIR["checkout-redesign"],
            },
          }),
        );
      }
      return Promise.resolve(new Response(null, { status: 404 }));
    });

    const c = createClient({
      baseUrl: "http://api",
      projectSlug: "demo",
      fetch: fetchMock,
    });
    const d = await c.evaluate("checkout-redesign", { user: { plan: "pro" } });
    expect(d.value).toBe(true);
    expect(d.reason).toBe("rule_matched");
    expect(d.version).toBe(9);
  });
});
