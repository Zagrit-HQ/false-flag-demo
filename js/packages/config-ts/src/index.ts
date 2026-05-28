// @falseflag/config — the TypeScript config-as-code DSL. Authors
// write a flag in TypeScript using the builders below; the default
// export is a JSON-serializable plain-object IR.
//
// Since slice 8, the server is the authoritative compiler: the CLI
// sends both the raw .ts file contents (as `source_text`) and the
// locally-compiled IR (as `source`) on publishFlagVersion. The API
// re-runs esbuild + goja against `source_text` and persists the
// produced IR alongside the original text, so the dashboard can
// render the author's input verbatim with Shiki syntax highlighting.
// The CLI's local compile remains a fast pre-flight check; on a
// divergence between the locally-compiled IR and the server's, the
// server wins.
//
// The Go-side shim that re-implements these builders lives at
// internal/config/typescript_shim.js. Keep the two in sync — the
// cross-runtime corpus exercises both compilers against the same TS
// fixtures.

export type ValueType = "boolean" | "string" | "number" | "object";

export interface PredicateNode {
  kind: string;
  [k: string]: unknown;
}

export interface RuleNode {
  id: string;
  when: PredicateNode;
  value: unknown;
}

export interface FlagDocument {
  value_type: ValueType;
  default: unknown;
  rules: RuleNode[];
}

export interface FlagInput {
  /** OpenFeature value type. */
  value_type: ValueType;
  /** Default value served when no rule matches. */
  default: unknown;
  /** Rules evaluated in order; first match wins. */
  rules: RuleNode[];
}

// Predicate builders ------------------------------------------------------

export function eq(attr: string, value: unknown): PredicateNode {
  return { kind: "eq", attr, value };
}

export function neq(attr: string, value: unknown): PredicateNode {
  return { kind: "neq", attr, value };
}

export function isIn(attr: string, values: unknown[]): PredicateNode {
  return { kind: "in", attr, values };
}

export function gt(attr: string, value: number): PredicateNode {
  return { kind: "gt", attr, value };
}

export function gte(attr: string, value: number): PredicateNode {
  return { kind: "gte", attr, value };
}

export function lt(attr: string, value: number): PredicateNode {
  return { kind: "lt", attr, value };
}

export function lte(attr: string, value: number): PredicateNode {
  return { kind: "lte", attr, value };
}

export function matches(attr: string, pattern: string): PredicateNode {
  return { kind: "matches", attr, pattern };
}

export function rollout(
  attr: string,
  salt: string,
  percent: number,
): PredicateNode {
  return { kind: "rollout", attr, salt, percent };
}

export function all(...predicates: PredicateNode[]): PredicateNode {
  return { kind: "all", of: predicates };
}

export function any(...predicates: PredicateNode[]): PredicateNode {
  return { kind: "any", of: predicates };
}

export function not(predicate: PredicateNode): PredicateNode {
  return { kind: "not", of_one: predicate };
}

/** CEL expression as a predicate; evaluated against `ctx`. */
export function cel(source: string): PredicateNode {
  return { kind: "cel", source };
}

export function always(): PredicateNode {
  return { kind: "always" };
}

// Rule + flag builders ----------------------------------------------------

export function rule(
  id: string,
  when: PredicateNode,
  value: unknown,
): RuleNode {
  return { id, when, value };
}

/**
 * Produce the wire shape the FalseFlag API expects under
 * `{strategy: "typescript", source: <this>}`.
 */
export function flag(input: FlagInput): FlagDocument {
  return {
    value_type: input.value_type,
    default: input.default,
    rules: input.rules,
  };
}

/**
 * FalseFlag is a convenience namespace so authors can write
 * `FalseFlag.flag(...)` instead of importing every helper.
 */
export const FalseFlag = {
  flag,
  rule,
  eq,
  neq,
  in: isIn,
  gt,
  gte,
  lt,
  lte,
  matches,
  rollout,
  all,
  any,
  not,
  cel,
  always,
};
