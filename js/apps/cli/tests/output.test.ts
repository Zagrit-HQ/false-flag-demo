import { PassThrough } from "node:stream";

import { describe, expect, it } from "vitest";

import { okOrLog, printItems, writeJSON } from "../src/output.js";

function captureStream() {
  const s = new PassThrough();
  const chunks: Buffer[] = [];
  s.on("data", (c: Buffer) => chunks.push(c));
  return {
    stream: s,
    text: () => Buffer.concat(chunks).toString("utf8"),
  };
}

describe("printItems", () => {
  it.each([undefined, []])("writes (no items) for %p", async (items) => {
    const c = captureStream();
    printItems(c.stream, items as unknown[] | undefined, (x) => String(x));
    expect(c.text()).toBe("(no items)\n");
  });

  it("writes each item on its own line", () => {
    const c = captureStream();
    printItems(c.stream, ["a", "b", "c"], (x) => x);
    expect(c.text()).toBe("a\nb\nc\n");
  });

  it.each([1, 2, 5, 10, 50, 100])("handles %i items", (n) => {
    const c = captureStream();
    const items = Array.from({ length: n }, (_, i) => i);
    printItems(c.stream, items, (x) => `item-${x}`);
    const lines = c.text().trim().split("\n");
    expect(lines).toHaveLength(n);
    expect(lines[0]).toBe("item-0");
    expect(lines[n - 1]).toBe(`item-${n - 1}`);
  });

  it("uses the formatter for each item", () => {
    const c = captureStream();
    printItems(c.stream, [1, 2, 3], (x) => `[${x}]`);
    expect(c.text()).toBe("[1]\n[2]\n[3]\n");
  });
});

describe("writeJSON", () => {
  it("writes pretty JSON by default", () => {
    const c = captureStream();
    writeJSON(c.stream, { a: 1, b: "x" });
    expect(c.text()).toBe(`{\n  "a": 1,\n  "b": "x"\n}\n`);
  });

  it("supports compact when pretty=false", () => {
    const c = captureStream();
    writeJSON(c.stream, { a: 1, b: "x" }, false);
    expect(c.text()).toBe(`{"a":1,"b":"x"}\n`);
  });

  it.each([
    null,
    true,
    false,
    0,
    1,
    -3.14,
    "",
    "x",
    [],
    [1, 2, 3],
    {},
    { k: "v" },
  ])("round-trips %p", (v) => {
    const c = captureStream();
    writeJSON(c.stream, v);
    // Trailing newline.
    expect(c.text().endsWith("\n")).toBe(true);
    const parsed = JSON.parse(c.text().trim());
    expect(parsed).toEqual(v);
  });
});

describe("okOrLog", () => {
  it.each([200, 201, 202, 204, 299])(
    "returns data on status %i",
    async (status) => {
      const c = captureStream();
      const got = await okOrLog(
        Promise.resolve({ status, data: { ok: true } }),
        c.stream,
      );
      expect(got).toEqual({ ok: true });
      expect(c.text()).toBe("");
    },
  );

  it.each([300, 301, 400, 401, 403, 404, 409, 422, 500, 503])(
    "logs and returns undefined on status %i",
    async (status) => {
      const c = captureStream();
      const got = await okOrLog(
        Promise.resolve({ status, data: { error: "x" } }),
        c.stream,
      );
      expect(got).toBeUndefined();
      expect(c.text()).toMatch(/error: HTTP/);
      expect(c.text()).toContain(String(status));
    },
  );

  it("logs thrown errors", async () => {
    const c = captureStream();
    const got = await okOrLog(
      Promise.reject(new Error("connection refused")),
      c.stream,
    );
    expect(got).toBeUndefined();
    expect(c.text()).toContain("connection refused");
  });
});
