import { writeFileSync } from "node:fs";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { PassThrough } from "node:stream";

import { describe, expect, it } from "vitest";

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

function tempFile(name: string, contents: string): string {
  const dir = mkdtempSync(join(tmpdir(), "falseflag-cfg-"));
  const path = join(dir, name);
  writeFileSync(path, contents);
  return path;
}

describe("falseflag config validate", () => {
  it("accepts a well-formed JSON IR", async () => {
    const cap = capture();
    const path = tempFile(
      "spec.json",
      JSON.stringify({ value_type: "boolean", default: false, rules: [] }),
    );
    const program = createProgram({ out: cap.out, err: cap.err });
    await program.parseAsync(
      [
        "config",
        "validate",
        "--project",
        "demo",
        "--strategy",
        "json",
        "--file",
        path,
      ],
      { from: "user" },
    );
    expect(cap.stdout()).toMatch(/valid json source/);
  });

  it("rejects a malformed IR (missing rules)", async () => {
    const cap = capture();
    const path = tempFile(
      "bad.json",
      JSON.stringify({ value_type: "boolean", default: false }),
    );
    const program = createProgram({ out: cap.out, err: cap.err });
    await program.parseAsync(
      [
        "config",
        "validate",
        "--project",
        "demo",
        "--strategy",
        "json",
        "--file",
        path,
      ],
      { from: "user" },
    );
    expect(cap.stderr()).toMatch(/invalid:/);
  });
});

describe("falseflag config save", () => {
  it("POSTs the source to publishFlagVersion and prints the new version", async () => {
    const cap = capture();
    const ir = { value_type: "boolean", default: false, rules: [] };
    const fileContents = JSON.stringify(ir);
    const path = tempFile("spec.json", fileContents);

    type Captured = { url: string; body: unknown };
    let captured: Captured | null = null;
    const fakeFetch: typeof fetch = async (url, init) => {
      const body = init?.body ? JSON.parse(String(init.body)) : undefined;
      captured = { url: String(url), body } as Captured;
      return new Response(
        JSON.stringify({
          version: {
            id: "v-uuid",
            flag_id: "f-uuid",
            version: 3,
            strategy: "json",
            compiled: ir,
            created_at: "2026-05-20T00:00:00Z",
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    };

    const program = createProgram({
      out: cap.out,
      err: cap.err,
      baseUrl: "http://api",
      fetch: fakeFetch,
    });
    await program.parseAsync(
      [
        "config",
        "save",
        "--project",
        "demo",
        "--flag",
        "check",
        "--strategy",
        "json",
        "--file",
        path,
      ],
      { from: "user" },
    );

    expect((captured as { url: string } | null)?.url).toBe(
      "http://api/v1/projects/demo/flags/check",
    );
    expect((captured as { body: unknown } | null)?.body).toEqual({
      strategy: "json",
      source: ir,
      source_text: fileContents,
    });
    expect(cap.stdout()).toMatch(/published demo\/check version 3/);
  });

  it("sends raw TypeScript source_text byte-for-byte", async () => {
    const cap = capture();
    const tsSource = [
      'import * as F from "@falseflag/config";',
      "",
      "export default F.flag({",
      '  value_type: "string",',
      '  default: "light",',
      "  rules: [],",
      "});",
      "",
    ].join("\n");
    const path = tempFile("flag.ts", tsSource);

    type Captured = { url: string; body: unknown };
    let captured: Captured | null = null;
    const fakeFetch: typeof fetch = async (_url, init) => {
      const body = init?.body ? JSON.parse(String(init.body)) : undefined;
      captured = { url: String(_url), body } as Captured;
      return new Response(
        JSON.stringify({
          version: {
            id: "v-uuid",
            flag_id: "f-uuid",
            version: 5,
            strategy: "typescript",
            compiled: { value_type: "string", default: "light", rules: [] },
            source_text: tsSource,
            created_at: "2026-05-26T00:00:00Z",
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    };

    const program = createProgram({
      out: cap.out,
      err: cap.err,
      baseUrl: "http://api",
      fetch: fakeFetch,
    });
    await program.parseAsync(
      [
        "config",
        "save",
        "--project",
        "demo",
        "--flag",
        "theme",
        "--strategy",
        "typescript",
        "--file",
        path,
      ],
      { from: "user" },
    );

    const body = (captured as { body: { source_text: string } } | null)?.body;
    expect(body?.source_text).toBe(tsSource);
    expect(cap.stdout()).toMatch(/published demo\/theme version 5/);
  });
});
