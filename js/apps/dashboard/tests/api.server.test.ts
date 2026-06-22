import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { apiBaseUrl, withApiFetch } from "../app/lib/api.server";

const ORIG_FETCH = globalThis.fetch;

beforeEach(() => {
  // biome-ignore lint/performance/noDelete: assigning undefined coerces to the string "undefined" and breaks the ?? check in api.server.ts.
  delete (process.env as Record<string, string | undefined>)
    .FALSEFLAG_API_BASE_URL;
});

afterEach(() => {
  globalThis.fetch = ORIG_FETCH;
});

describe("apiBaseUrl", () => {
  it("defaults to http://localhost:8080", () => {
    expect(apiBaseUrl()).toBe("http://localhost:8080");
  });

  it.each([
    "http://localhost:8080",
    "http://localhost:9090",
    "http://api.example.com",
    "https://api.production.example.com:8443",
    "http://10.0.0.1:8080",
  ])("returns %s when env is set", (url) => {
    process.env.FALSEFLAG_API_BASE_URL = url;
    expect(apiBaseUrl()).toBe(url);
  });
});

describe("withApiFetch", () => {
  it.each([
    "/v1/projects",
    "/v1/projects/demo",
    "/v1/projects/demo/flags",
    "/healthz",
    "/openapi.yaml",
  ])("rewrites relative URL %s", async (path) => {
    const fakeFetch = vi.fn(
      async () => new Response("ok", { status: 200 }),
    ) as unknown as typeof fetch;
    globalThis.fetch = fakeFetch;
    await withApiFetch(async () => {
      await globalThis.fetch(path);
    });
    expect((fakeFetch as ReturnType<typeof vi.fn>).mock.calls[0]?.[0]).toBe(
      `http://localhost:8080${path}`,
    );
  });

  it("respects opts.baseUrl override", async () => {
    const fakeFetch = vi.fn(
      async () => new Response("ok"),
    ) as unknown as typeof fetch;
    globalThis.fetch = fakeFetch;
    await withApiFetch(
      async () => {
        await globalThis.fetch("/x");
      },
      { baseUrl: "http://api.local:9090" },
    );
    expect((fakeFetch as ReturnType<typeof vi.fn>).mock.calls[0]?.[0]).toBe(
      "http://api.local:9090/x",
    );
  });

  it("passes through absolute URLs unmodified", async () => {
    const fakeFetch = vi.fn(
      async () => new Response("ok"),
    ) as unknown as typeof fetch;
    globalThis.fetch = fakeFetch;
    await withApiFetch(async () => {
      await globalThis.fetch("https://example.com/x");
    });
    expect((fakeFetch as ReturnType<typeof vi.fn>).mock.calls[0]?.[0]).toBe(
      "https://example.com/x",
    );
  });

  it("restores fetch after the body", async () => {
    const sentinel = vi.fn(
      async () => new Response("orig"),
    ) as unknown as typeof fetch;
    globalThis.fetch = sentinel;
    await withApiFetch(async () => undefined);
    expect(globalThis.fetch).toBe(sentinel);
  });

  it("restores fetch on thrown error too", async () => {
    const sentinel = vi.fn(
      async () => new Response("orig"),
    ) as unknown as typeof fetch;
    globalThis.fetch = sentinel;
    await expect(
      withApiFetch(async () => {
        throw new Error("boom");
      }),
    ).rejects.toThrow("boom");
    expect(globalThis.fetch).toBe(sentinel);
  });

  it("returns the body's return value", async () => {
    const got = await withApiFetch(async () => 42);
    expect(got).toBe(42);
  });

  it.each([1, 2, 3, 4, 5])(
    "nests %i deep without leaking fetch",
    async (depth) => {
      const sentinel = vi.fn(
        async () => new Response("orig"),
      ) as unknown as typeof fetch;
      globalThis.fetch = sentinel;
      async function nest(n: number): Promise<number> {
        if (n === 0) return 1;
        return withApiFetch(() => nest(n - 1));
      }
      await nest(depth);
      expect(globalThis.fetch).toBe(sentinel);
    },
  );
});
