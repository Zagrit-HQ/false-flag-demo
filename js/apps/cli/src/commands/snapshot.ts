import { writeFileSync } from "node:fs";

import type { Command } from "commander";

import { getLatestSnapshot } from "@falseflag/generated-client";

import { okOrLog } from "../output.js";

export interface SnapshotContext {
  out: NodeJS.WritableStream;
  err: NodeJS.WritableStream;
}

/**
 * canonicalize sorts the top-level `flags` map for stable diffs.
 * See docs/snapshot-format.md.
 */
function canonicalize(snapshot: unknown): unknown {
  if (snapshot === null || typeof snapshot !== "object") return snapshot;
  const s = snapshot as Record<string, unknown>;
  const compiled = s.compiled as
    | { flags?: Record<string, unknown> }
    | undefined;
  if (compiled?.flags) {
    const sorted: Record<string, unknown> = {};
    for (const key of Object.keys(compiled.flags).sort()) {
      sorted[key] = compiled.flags[key];
    }
    s.compiled = { ...compiled, flags: sorted };
  }
  return s;
}

function toYAML(value: unknown, indent = 0): string {
  const pad = "  ".repeat(indent);
  if (value === null) return "null";
  if (typeof value === "string") return JSON.stringify(value);
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  if (Array.isArray(value)) {
    if (value.length === 0) return "[]";
    return value
      .map((item) => `${pad}- ${toYAML(item, indent + 1).replace(/^\s+/, "")}`)
      .join("\n");
  }
  if (typeof value === "object") {
    const keys = Object.keys(value as Record<string, unknown>).sort();
    if (keys.length === 0) return "{}";
    return keys
      .map((k) => {
        const v = (value as Record<string, unknown>)[k];
        if (typeof v === "object" && v !== null) {
          return `${pad}${k}:\n${toYAML(v, indent + 1)}`;
        }
        return `${pad}${k}: ${toYAML(v, indent + 1)}`;
      })
      .join("\n");
  }
  return String(value);
}

export function registerSnapshot(program: Command, ctx: SnapshotContext): void {
  const snapshot = program.command("snapshot").description("snapshot commands");

  snapshot
    .command("latest")
    .description("print the latest snapshot for a project")
    .requiredOption("--project <slug>", "project slug")
    .action(async (opts: { project: string }) => {
      const data = await okOrLog(getLatestSnapshot(opts.project), ctx.err);
      if (data) ctx.out.write(`${JSON.stringify(data, null, 2)}\n`);
    });

  snapshot
    .command("export")
    .description("export the latest snapshot in canonical JSON or YAML")
    .requiredOption("--project <slug>", "project slug")
    .option("--format <fmt>", "json | yaml", "json")
    .option("--out <path>", "output file (defaults to stdout)")
    .action(async (opts: { project: string; format: string; out?: string }) => {
      const data = await okOrLog(getLatestSnapshot(opts.project), ctx.err);
      if (!data) return;
      const canonical = canonicalize(data);
      let serialized: string;
      if (opts.format === "yaml") {
        serialized = `${toYAML(canonical)}\n`;
      } else if (opts.format === "json") {
        serialized = `${JSON.stringify(canonical, null, 2)}\n`;
      } else {
        ctx.err.write(`unknown format ${opts.format}\n`);
        return;
      }
      if (opts.out) {
        writeFileSync(opts.out, serialized);
        ctx.out.write(`wrote ${opts.out}\n`);
      } else {
        ctx.out.write(serialized);
      }
    });
}
