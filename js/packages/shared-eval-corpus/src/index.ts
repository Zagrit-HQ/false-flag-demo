// @falseflag/shared-eval-corpus loads the canonical cross-runtime
// fixtures from tests/eval-corpus/ at the repo root and exposes them
// to JS test runners. The same files are read by Go in
// internal/eval/cross_runtime_test.go — if you change the format,
// update both sides.

import { readFileSync, readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import type {
  Decision,
  DecisionReason,
  RulesTree,
  Strategy,
} from "@falseflag/sdk";

export interface TestCase {
  id: string;
  description: string;
  strategy: Strategy;
  ir: RulesTree;
  context: Record<string, unknown>;
  expected: {
    value: unknown;
    reason: DecisionReason;
    rule_id?: string;
  };
}

const here = dirname(fileURLToPath(import.meta.url));

// Repo root is four levels up from this source file:
//   js/packages/shared-eval-corpus/src/index.ts
//                     ^             ^   ^
//                     |             |   here
//                     |             dirname()
//                     dirname() x 3 = packages → shared-eval-corpus's parent
//
// We resolve relative to the file rather than process.cwd() so the
// loader works in any test runner regardless of working directory.
function corpusDir(): string {
  return join(here, "..", "..", "..", "..", "tests", "eval-corpus");
}

export function loadCorpus(): TestCase[] {
  const dir = corpusDir();
  const files = readdirSync(dir)
    .filter((f) => f.endsWith(".json"))
    .sort();
  return files.map((file) => {
    const raw = readFileSync(join(dir, file), "utf8");
    return JSON.parse(raw) as TestCase;
  });
}

// Helper for assertions: round-trip values through JSON for stable
// comparison across nested objects.
export function jsonEqualish(a: unknown, b: unknown): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

// Re-export for ergonomics so test files can `import { Decision }`
// without depending on @falseflag/sdk directly.
export type { Decision, DecisionReason, RulesTree, Strategy };
