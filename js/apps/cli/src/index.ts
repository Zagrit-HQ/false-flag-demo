import { Command } from "commander";

import {
  type Flag,
  type Project,
  listFlags,
  listProjects,
} from "@falseflag/generated-client";

import { registerAuth } from "./commands/auth.js";
import { registerConfig } from "./commands/config.js";
import { registerSmoke } from "./commands/smoke.js";
import { registerSnapshot } from "./commands/snapshot.js";
import { readCredentials } from "./credentials.js";
import { okOrLog, printItems } from "./output.js";

const VERSION = "0.1.0";

export interface ProgramOptions {
  /** Override stdout. Useful in tests. */
  out?: NodeJS.WritableStream;
  /** Override stderr. */
  err?: NodeJS.WritableStream;
  /** Override the API base URL. Defaults to FALSEFLAG_API_BASE_URL or localhost. */
  baseUrl?: string;
  /** Custom fetch implementation. The generated client uses bare fetch(); we
   *  pass it via options.fetch when present, otherwise fall back to globalThis.fetch. */
  fetch?: typeof fetch;
  /** Override HOME (test-only) so credentials live in a temp dir. */
  home?: string;
}

/**
 * createProgram returns a configured Commander program. The
 * Orval-generated client uses relative paths like `/v1/projects`; we
 * prefix them with the configured baseUrl in a wrapping fetch.
 *
 * The wrapping fetch also injects the `X-Actor` request header from
 * `~/.config/falseflag/credentials.json` if present — this is how
 * `falseflag config save` ends up attributing audit events to the
 * configured demo user.
 */
export function createProgram(opts: ProgramOptions = {}): Command {
  const out = opts.out ?? process.stdout;
  const err = opts.err ?? process.stderr;
  const baseUrl =
    opts.baseUrl ??
    process.env.FALSEFLAG_API_BASE_URL ??
    "http://localhost:8080";
  const realFetch = opts.fetch ?? globalThis.fetch;
  const creds = readCredentials(opts.home);
  const apiFetch: typeof fetch = (input, init) => {
    const initHeaders = new Headers(init?.headers);
    if (creds?.actor && !initHeaders.has("X-Actor")) {
      initHeaders.set("X-Actor", `cli/${creds.actor}`);
    }
    const finalInit = { ...init, headers: initHeaders };
    if (typeof input === "string" && input.startsWith("/")) {
      return realFetch(`${baseUrl}${input}`, finalInit);
    }
    return realFetch(input, finalInit);
  };
  // The generated client calls bare fetch(); patch globalThis just
  // for the lifetime of this program object.
  const prior = globalThis.fetch;
  globalThis.fetch = apiFetch;
  const restoreFetch = () => {
    globalThis.fetch = prior;
  };

  const program = new Command();

  program
    .name("falseflag")
    .description("FalseFlag command-line client")
    .version(VERSION, "-v, --version", "print CLI version")
    .hook("postAction", restoreFetch);

  program
    .command("health")
    .description("print local CLI health (no network call)")
    .action(() => {
      out.write(
        `${JSON.stringify({ status: "ok", cli: "falseflag", version: VERSION })}\n`,
      );
    });

  registerAuth(program, { out, err, home: opts.home });

  const project = program.command("project").description("project commands");
  project
    .command("list")
    .description("list projects from the FalseFlag API")
    .action(async () => {
      const data = await okOrLog(listProjects(), err);
      if (!data) return;
      const list = data as { items: Project[] };
      printItems(out, list.items, (p) => `${p.slug}\t${p.display_name}`);
    });

  const flag = program.command("flag").description("flag commands");
  flag
    .command("list")
    .description("list flags in a project")
    .requiredOption("--project <slug>", "project slug")
    .action(async (cmdOpts: { project: string }) => {
      const data = await okOrLog(listFlags(cmdOpts.project), err);
      if (!data) return;
      const list = data as { items: Flag[] };
      printItems(
        out,
        list.items,
        (f) => `${f.key}\t${f.value_type}\t${f.name}`,
      );
    });

  registerConfig(program, { out, err });
  registerSnapshot(program, { out, err });
  registerSmoke(program, { out, err });

  return program;
}
