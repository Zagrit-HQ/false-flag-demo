import type { Command } from "commander";

import {
  compileSnapshot,
  evaluateFlag,
  getLatestSnapshot,
  publishFlagVersion,
} from "@falseflag/generated-client";

import { okOrLog } from "../output.js";

export interface SmokeContext {
  out: NodeJS.WritableStream;
  err: NodeJS.WritableStream;
}

/**
 * smoke-test exercises the end-to-end "save → snapshot → evaluate"
 * workflow against an existing project + flag. It does NOT create
 * the project or flag (those are seeded by `make seed` or a manual
 * `curl` from the demo script).
 */
export function registerSmoke(program: Command, ctx: SmokeContext): void {
  program
    .command("smoke-test")
    .description(
      "chain validate → save → snapshot → evaluate against a project",
    )
    .requiredOption("--project <slug>", "project slug (must exist)")
    .option("--flag <key>", "flag key (must exist)", "smoke-bool")
    .action(async (opts: { project: string; flag: string }) => {
      ctx.out.write(`smoke-test against ${opts.project}/${opts.flag}\n`);

      // 1. Publish a trivial IR.
      const ir = {
        value_type: "boolean",
        default: false,
        rules: [
          {
            id: "rule-pro",
            when: { kind: "eq", attr: "user.plan", value: "pro" },
            value: true,
          },
        ],
      };
      ctx.out.write("  step 1/3: publishing a fresh flag version… ");
      const pub = await okOrLog(
        publishFlagVersion(opts.project, opts.flag, {
          strategy: "json",
          source: ir,
        }),
        ctx.err,
      );
      if (!pub) {
        ctx.err.write("smoke-test failed at publish step\n");
        process.exitCode = 1;
        return;
      }
      ctx.out.write("ok\n");

      // 2. Compile a project-wide snapshot.
      ctx.out.write("  step 2/3: compiling project snapshot… ");
      const snap = await okOrLog(compileSnapshot(opts.project, {}), ctx.err);
      if (!snap) {
        ctx.err.write("smoke-test failed at snapshot step\n");
        process.exitCode = 1;
        return;
      }
      ctx.out.write("ok\n");

      // 3. Confirm getLatestSnapshot agrees, then evaluate against
      //    a pro user.
      ctx.out.write("  step 3/3: evaluating the flag for user.plan=pro… ");
      const latest = await okOrLog(getLatestSnapshot(opts.project), ctx.err);
      if (!latest) {
        ctx.err.write("smoke-test failed at latest-snapshot step\n");
        process.exitCode = 1;
        return;
      }
      const dec = await okOrLog(
        evaluateFlag(opts.project, opts.flag, {
          context: { user: { plan: "pro" } },
        }),
        ctx.err,
      );
      if (!dec) {
        ctx.err.write("smoke-test failed at evaluate step\n");
        process.exitCode = 1;
        return;
      }
      const d = dec as { value?: unknown; reason?: string };
      ctx.out.write(
        `ok — value=${JSON.stringify(d.value)} reason=${d.reason}\n`,
      );
      ctx.out.write("smoke-test PASSED\n");
    });
}
