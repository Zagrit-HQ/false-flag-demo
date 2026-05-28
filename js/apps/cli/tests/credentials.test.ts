import { existsSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, beforeEach, describe, expect, it } from "vitest";

import {
  credentialsPath,
  readCredentials,
  writeCredentials,
} from "../src/credentials.js";

let tempHome: string;

beforeEach(() => {
  tempHome = join(
    tmpdir(),
    `ff-cli-creds-${Math.random().toString(36).slice(2)}`,
  );
});

afterEach(() => {
  if (existsSync(tempHome)) {
    rmSync(tempHome, { recursive: true, force: true });
  }
});

describe("credentialsPath", () => {
  it.each(["/home/alice", "/tmp/x", "/var/empty", "/Users/wito"])(
    "places file under %s/.config/falseflag",
    (home) => {
      expect(credentialsPath(home)).toBe(
        `${home}/.config/falseflag/credentials.json`,
      );
    },
  );

  it("returns the file under $HOME/.config/falseflag", () => {
    const p = credentialsPath();
    expect(p).toMatch(/\.config\/falseflag\/credentials\.json$/);
  });
});

describe("writeCredentials", () => {
  it.each([
    "alice",
    "bot",
    "alice@example.com",
    "ci-runner-42",
    "operator",
    "with spaces",
  ])("round-trips actor=%s", (actor) => {
    const saved_at = new Date().toISOString();
    writeCredentials({ actor, saved_at }, tempHome);
    const got = readCredentials(tempHome);
    expect(got).toEqual({ actor, saved_at });
  });

  it("creates the directory if missing", () => {
    writeCredentials({ actor: "x", saved_at: "now" }, tempHome);
    expect(existsSync(`${tempHome}/.config/falseflag`)).toBe(true);
  });

  it("overwrites an existing file", () => {
    writeCredentials({ actor: "alpha", saved_at: "1" }, tempHome);
    writeCredentials({ actor: "beta", saved_at: "2" }, tempHome);
    expect(readCredentials(tempHome)?.actor).toBe("beta");
  });

  it("returns the file path that was written", () => {
    const path = writeCredentials({ actor: "x", saved_at: "1" }, tempHome);
    expect(path).toBe(`${tempHome}/.config/falseflag/credentials.json`);
  });
});

describe("readCredentials", () => {
  it("returns null when the file does not exist", () => {
    expect(readCredentials(tempHome)).toBeNull();
  });

  it("returns null for malformed JSON", () => {
    mkdirSync(`${tempHome}/.config/falseflag`, { recursive: true });
    writeFileSync(`${tempHome}/.config/falseflag/credentials.json`, "{bad");
    expect(readCredentials(tempHome)).toBeNull();
  });

  it.each([
    { actor: "alice", saved_at: "2026-01-01T00:00:00.000Z" },
    { actor: "bot", saved_at: "2026-02-01T00:00:00.000Z" },
    { actor: "", saved_at: "" },
  ])("reads back %j verbatim", (creds) => {
    writeCredentials(creds, tempHome);
    expect(readCredentials(tempHome)).toEqual(creds);
  });
});
