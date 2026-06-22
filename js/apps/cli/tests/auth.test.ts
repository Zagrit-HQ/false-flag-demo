import { existsSync, readFileSync, rmSync, statSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { PassThrough } from "node:stream";

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { createProgram } from "../src/index.js";

let tempHome: string;

beforeEach(() => {
  tempHome = join(
    tmpdir(),
    `falseflag-cli-test-${Math.random().toString(36).slice(2)}`,
  );
});

afterEach(() => {
  if (existsSync(tempHome)) {
    rmSync(tempHome, { recursive: true, force: true });
  }
});

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

describe("falseflag auth login + whoami", () => {
  it("writes credentials with mode 0600 and reads them back", async () => {
    const cap = capture();
    const program = createProgram({
      out: cap.out,
      err: cap.err,
      home: tempHome,
    });
    await program.parseAsync(
      ["auth", "login", "--token", "alice@example.com"],
      { from: "user" },
    );
    expect(cap.stdout()).toContain("saved credentials to");

    const path = join(tempHome, ".config", "falseflag", "credentials.json");
    expect(existsSync(path)).toBe(true);
    const stat = statSync(path);
    // Permissions check is best-effort: macOS/Linux only.
    if (process.platform !== "win32") {
      expect(stat.mode & 0o777).toBe(0o600);
    }
    const raw = JSON.parse(readFileSync(path, "utf8"));
    expect(raw.actor).toBe("alice@example.com");

    const cap2 = capture();
    const program2 = createProgram({
      out: cap2.out,
      err: cap2.err,
      home: tempHome,
    });
    await program2.parseAsync(["auth", "whoami"], { from: "user" });
    expect(cap2.stdout().trim()).toBe("alice@example.com");
  });

  it("whoami reports an error when no credentials exist", async () => {
    const cap = capture();
    const program = createProgram({
      out: cap.out,
      err: cap.err,
      home: tempHome,
    });
    await program.parseAsync(["auth", "whoami"], { from: "user" });
    expect(cap.stderr()).toContain("no credentials found");
  });
});
