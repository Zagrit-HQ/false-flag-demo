import { existsSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { PassThrough } from "node:stream";

import { afterEach, describe, expect, it } from "vitest";

import { createProgram } from "../src/index.js";

function capture() {
  const out = new PassThrough();
  const err = new PassThrough();
  const outChunks: Buffer[] = [];
  const errChunks: Buffer[] = [];
  out.on("data", (c: Buffer) => outChunks.push(c));
  err.on("data", (c: Buffer) => errChunks.push(c));
  return {
    out,
    err,
    stdout: () => Buffer.concat(outChunks).toString("utf8"),
    stderr: () => Buffer.concat(errChunks).toString("utf8"),
  };
}

const snapshotBody = {
  id: "snap-1",
  project_id: "proj-1",
  version: 4,
  created_at: "2026-05-20T12:00:00Z",
  compiled: {
    flags: {
      "z-flag": { value_type: "boolean", default: false, rules: [] },
      "a-flag": { value_type: "string", default: "x", rules: [] },
    },
  },
};

const fakeFetch: typeof fetch = async () =>
  new Response(JSON.stringify(snapshotBody), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });

const tempPaths: string[] = [];
afterEach(() => {
  for (const p of tempPaths) {
    if (existsSync(p)) rmSync(p, { force: true });
  }
});

describe("falseflag snapshot export", () => {
  it("emits canonical JSON with flags sorted alphabetically", async () => {
    const cap = capture();
    const program = createProgram({
      out: cap.out,
      err: cap.err,
      baseUrl: "http://api",
      fetch: fakeFetch,
    });
    await program.parseAsync(
      ["snapshot", "export", "--project", "demo", "--format", "json"],
      { from: "user" },
    );
    const out = cap.stdout();
    // Verify flags were sorted (a-flag should appear before z-flag in JSON).
    expect(out.indexOf("a-flag")).toBeLessThan(out.indexOf("z-flag"));
  });

  it("writes to --out when supplied", async () => {
    const cap = capture();
    const out = join(tmpdir(), `snap-${Date.now()}.json`);
    tempPaths.push(out);
    const program = createProgram({
      out: cap.out,
      err: cap.err,
      baseUrl: "http://api",
      fetch: fakeFetch,
    });
    await program.parseAsync(
      ["snapshot", "export", "--project", "demo", "--out", out],
      { from: "user" },
    );
    expect(existsSync(out)).toBe(true);
    const contents = readFileSync(out, "utf8");
    expect(contents).toContain("a-flag");
  });

  it("supports yaml format", async () => {
    const cap = capture();
    const program = createProgram({
      out: cap.out,
      err: cap.err,
      baseUrl: "http://api",
      fetch: fakeFetch,
    });
    await program.parseAsync(
      ["snapshot", "export", "--project", "demo", "--format", "yaml"],
      { from: "user" },
    );
    const out = cap.stdout();
    // YAML uses `key: value` with no quotes around keys.
    expect(out).toMatch(/created_at:/);
    expect(out).toMatch(/version: 4/);
  });
});
