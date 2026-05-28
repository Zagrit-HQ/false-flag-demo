import type { Command } from "commander";

import {
  credentialsPath,
  readCredentials,
  writeCredentials,
} from "../credentials.js";

export interface AuthContext {
  out: NodeJS.WritableStream;
  err: NodeJS.WritableStream;
  /** Override HOME (test-only). */
  home?: string;
}

export function registerAuth(program: Command, ctx: AuthContext): void {
  const auth = program.command("auth").description("authentication commands");

  auth
    .command("login")
    .description("save a demo token to ~/.config/falseflag/credentials.json")
    .option(
      "--token <actor>",
      "the actor token (e.g. user@example.com)",
      "demo-user",
    )
    .action((opts: { token: string }) => {
      const path = writeCredentials(
        { actor: opts.token, saved_at: new Date().toISOString() },
        ctx.home,
      );
      ctx.out.write(`saved credentials to ${path}\n`);
    });

  auth
    .command("whoami")
    .description("print the stored actor token")
    .action(() => {
      const creds = readCredentials(ctx.home);
      if (!creds) {
        ctx.err.write(
          `no credentials found at ${credentialsPath(ctx.home)}; run "falseflag auth login" first\n`,
        );
        return;
      }
      ctx.out.write(`${creds.actor}\n`);
    });
}
