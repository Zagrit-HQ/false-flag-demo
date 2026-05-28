// Demo-quality CLI credentials store.
//
// Slice 5 deliberately does not implement OAuth. `falseflag auth login`
// writes a small JSON file at ~/.config/falseflag/credentials.json
// with the actor value the CLI sends in the X-Actor header (per the
// slice-3 audit convention). The file is overwritten on each `login`.

import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { homedir } from "node:os";
import { dirname, join } from "node:path";

export interface Credentials {
  /** Free-form actor string. Sent as the X-Actor request header. */
  actor: string;
  /** When the token was stored (ISO-8601). */
  saved_at: string;
}

export function credentialsPath(home: string = homedir()): string {
  return join(home, ".config", "falseflag", "credentials.json");
}

export function writeCredentials(creds: Credentials, home?: string): string {
  const path = credentialsPath(home);
  const dir = dirname(path);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true, mode: 0o700 });
  }
  writeFileSync(path, `${JSON.stringify(creds, null, 2)}\n`, {
    mode: 0o600,
  });
  return path;
}

export function readCredentials(home?: string): Credentials | null {
  const path = credentialsPath(home);
  if (!existsSync(path)) return null;
  try {
    return JSON.parse(readFileSync(path, "utf8")) as Credentials;
  } catch {
    return null;
  }
}
