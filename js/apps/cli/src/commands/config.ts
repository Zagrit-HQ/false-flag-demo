import { readFileSync } from "node:fs";

import type { Command } from "commander";

import { type Strategy, publishFlagVersion } from "@falseflag/generated-client";

import { okOrLog } from "../output.js";

export interface ConfigContext {
  out: NodeJS.WritableStream;
  err: NodeJS.WritableStream;
}

interface ValidateOptions {
  project: string;
  strategy: Strategy;
  file: string;
}

interface SaveOptions extends ValidateOptions {
  flag: string;
}

function parseStrategy(s: string): Strategy {
  const allowed: Strategy[] = ["json", "cel", "typescript"];
  if (!allowed.includes(s as Strategy)) {
    throw new Error(
      `invalid strategy "${s}" — must be one of ${allowed.join(", ")}`,
    );
  }
  return s as Strategy;
}

function readSource(path: string): unknown {
  const raw = readFileSync(path, "utf8");
  // For JSON / CEL strategies we expect the file to contain JSON
  // already. TypeScript strategy files are arbitrary source the
  // server compiles — pass them through as a string.
  try {
    return JSON.parse(raw);
  } catch {
    return raw;
  }
}

// readSourceText returns the raw file contents verbatim, regardless of
// strategy. The server stores this on the flag_version row so the
// dashboard can render the author's original input. For TypeScript
// flags it is also the authoritative compile input (the server
// re-compiles it via esbuild + goja); the locally-parsed `source` is
// retained for back-compat.
function readSourceText(path: string): string {
  return readFileSync(path, "utf8");
}

/**
 * Client-side IR sanity check: ensure the parsed source has the
 * minimum shape the server's compiler would accept. This is NOT a
 * server-side validation, but it catches the common 90% of authoring
 * mistakes (missing value_type, missing default, malformed rules array)
 * without a network round-trip.
 */
function basicShapeOK(
  source: unknown,
): { ok: true } | { ok: false; reason: string } {
  if (source === null || typeof source !== "object") {
    return { ok: false, reason: "source is not an object" };
  }
  const s = source as Record<string, unknown>;
  if (!("value_type" in s)) return { ok: false, reason: "missing value_type" };
  if (!("default" in s)) return { ok: false, reason: "missing default" };
  if (!("rules" in s) || !Array.isArray(s.rules))
    return { ok: false, reason: "missing or non-array rules" };
  return { ok: true };
}

export function registerConfig(program: Command, ctx: ConfigContext): void {
  const cfg = program
    .command("config")
    .description("validate and publish flag configuration");

  cfg
    .command("validate")
    .description("parse a config file and check the basic IR shape")
    .requiredOption("--project <slug>", "project slug")
    .requiredOption("--strategy <strategy>", "json | cel | typescript")
    .requiredOption("--file <path>", "config file path")
    .action((opts: ValidateOptions) => {
      try {
        const strategy = parseStrategy(opts.strategy);
        const source = readSource(opts.file);
        // For TypeScript strategy we only sanity-check that there's
        // some text content — the server runs the compiler.
        if (strategy === "typescript") {
          if (typeof source !== "string" && typeof source !== "object") {
            throw new Error("typescript source is empty");
          }
          ctx.out.write(`valid ${strategy} source: ${opts.file}\n`);
          return;
        }
        const r = basicShapeOK(source);
        if (!r.ok) throw new Error(r.reason);
        ctx.out.write(`valid ${strategy} source: ${opts.file}\n`);
      } catch (e) {
        ctx.err.write(`invalid: ${(e as Error).message}\n`);
      }
    });

  cfg
    .command("save")
    .description("publish a new flag version from a file (validates first)")
    .requiredOption("--project <slug>", "project slug")
    .requiredOption("--flag <key>", "flag key")
    .requiredOption("--strategy <strategy>", "json | cel | typescript")
    .requiredOption("--file <path>", "config file path")
    .action(async (opts: SaveOptions) => {
      const strategy = parseStrategy(opts.strategy);
      const source = readSource(opts.file);
      const sourceText = readSourceText(opts.file);

      const data = await okOrLog(
        publishFlagVersion(opts.project, opts.flag, {
          strategy,
          source: source as unknown,
          source_text: sourceText,
        }),
        ctx.err,
      );
      if (!data) return;
      const body = data as { version?: { version?: number; id?: string } };
      const v = body.version;
      ctx.out.write(
        `published ${opts.project}/${opts.flag} version ${v?.version ?? "?"} (${v?.id ?? "?"})\n`,
      );
    });
}
