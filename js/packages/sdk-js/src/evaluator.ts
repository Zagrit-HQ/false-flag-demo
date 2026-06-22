// JS mirror of internal/eval. Behavior must match the Go evaluator
// for every fixture in tests/eval-corpus/.

import { type CelProgram, compileCEL, evalCEL } from "./cel-lite.js";
import type { Decision, DecisionReason, Predicate, RulesTree } from "./ir.js";
import { inBucket } from "./rollout.js";

export interface Compiled {
  ir: RulesTree;
  celPrograms: Map<string, CelProgram>;
}

export function compile(ir: RulesTree): Compiled {
  const programs = new Map<string, CelProgram>();
  walkAndCompileCEL(
    ir.rules.map((r) => r.when),
    programs,
  );
  return { ir, celPrograms: programs };
}

function walkAndCompileCEL(
  preds: (Predicate | undefined)[],
  out: Map<string, CelProgram>,
): void {
  for (const p of preds) {
    if (!p) continue;
    if (p.kind === "cel" && p.source !== undefined && !out.has(p.source)) {
      out.set(p.source, compileCEL(p.source));
    }
    if (p.of) walkAndCompileCEL(p.of, out);
    if (p.of_one) walkAndCompileCEL([p.of_one], out);
  }
}

export function evaluate(
  compiled: Compiled,
  ctx: Record<string, unknown>,
  version = 1,
): Decision {
  const tree = compiled.ir;
  for (const rule of tree.rules) {
    let matched: boolean;
    try {
      matched = match(rule.when, ctx, compiled.celPrograms);
    } catch {
      return {
        value: tree.default,
        reason: "error",
        rule_id: rule.id,
        version,
      };
    }
    if (!matched) continue;
    if (!valueMatchesType(rule.value, tree.value_type)) {
      return {
        value: tree.default,
        reason: "type_mismatch",
        rule_id: rule.id,
        version,
      };
    }
    return {
      value: rule.value,
      reason: reasonForRule(rule.when),
      rule_id: rule.id,
      version,
    };
  }
  return { value: tree.default, reason: "default", version };
}

function reasonForRule(p: Predicate): DecisionReason {
  if (p.kind === "rollout") return "rollout_in_bucket";
  return "rule_matched";
}

function match(
  p: Predicate,
  ctx: Record<string, unknown>,
  programs: Map<string, CelProgram>,
): boolean {
  switch (p.kind) {
    case "always":
      return true;
    case "eq":
      return cmpEq(p, ctx);
    case "neq":
      return !cmpEq(p, ctx);
    case "in":
      return cmpIn(p, ctx);
    case "gt":
    case "gte":
    case "lt":
    case "lte":
      return cmpOrd(p, ctx);
    case "matches":
      return cmpMatches(p, ctx);
    case "rollout": {
      const v = lookupString(ctx, p.attr ?? "");
      if (v === undefined) return false;
      return inBucket(p.salt ?? "", v, p.percent ?? 0);
    }
    case "all":
      return (p.of ?? []).every((c) => match(c, ctx, programs));
    case "any":
      return (p.of ?? []).some((c) => match(c, ctx, programs));
    case "not":
      return !match(p.of_one as Predicate, ctx, programs);
    case "cel": {
      const prog = programs.get(p.source ?? "");
      if (!prog)
        throw new Error(
          `cel program missing for source ${JSON.stringify(p.source)}`,
        );
      return evalCEL(prog, ctx);
    }
  }
}

function cmpEq(p: Predicate, ctx: Record<string, unknown>): boolean {
  const actual = lookup(ctx, p.attr ?? "");
  if (actual === undefined) return false;
  return jsonEqual(actual, p.value);
}

function cmpIn(p: Predicate, ctx: Record<string, unknown>): boolean {
  const actual = lookup(ctx, p.attr ?? "");
  if (actual === undefined) return false;
  return (p.values ?? []).some((v) => jsonEqual(actual, v));
}

function cmpOrd(p: Predicate, ctx: Record<string, unknown>): boolean {
  const actual = lookup(ctx, p.attr ?? "");
  const a = toNumber(actual);
  const b = toNumber(p.value);
  if (a === undefined || b === undefined) return false;
  switch (p.kind) {
    case "gt":
      return a > b;
    case "gte":
      return a >= b;
    case "lt":
      return a < b;
    case "lte":
      return a <= b;
    default:
      return false;
  }
}

function cmpMatches(p: Predicate, ctx: Record<string, unknown>): boolean {
  const actual = lookupString(ctx, p.attr ?? "");
  if (actual === undefined) return false;
  try {
    return new RegExp(p.pattern ?? "").test(actual);
  } catch {
    return false;
  }
}

function lookup(ctx: Record<string, unknown>, path: string): unknown {
  if (!path) return undefined;
  const parts = path.split(".");
  let cur: unknown = ctx;
  for (const part of parts) {
    if (cur === null || typeof cur !== "object") return undefined;
    cur = (cur as Record<string, unknown>)[part];
  }
  return cur;
}

function lookupString(
  ctx: Record<string, unknown>,
  path: string,
): string | undefined {
  const v = lookup(ctx, path);
  return typeof v === "string" ? v : undefined;
}

function toNumber(v: unknown): number | undefined {
  if (typeof v === "number" && Number.isFinite(v)) return v;
  return undefined;
}

function jsonEqual(a: unknown, b: unknown): boolean {
  if (a === b) return true;
  if (a === null || b === null) return a === b;
  if (typeof a !== typeof b) return false;
  if (typeof a === "object") {
    return JSON.stringify(a) === JSON.stringify(b);
  }
  return false;
}

function valueMatchesType(v: unknown, t: RulesTree["value_type"]): boolean {
  switch (t) {
    case "boolean":
      return typeof v === "boolean";
    case "string":
      return typeof v === "string";
    case "number":
      return typeof v === "number";
    case "object":
      return typeof v === "object" && v !== null && !Array.isArray(v);
  }
}
