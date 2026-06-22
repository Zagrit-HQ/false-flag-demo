// IR types — keep in sync with internal/config/ir.go.

export type ValueType = "boolean" | "string" | "number" | "object";

export type Strategy = "json" | "cel" | "typescript";

export type PredicateKind =
  | "eq"
  | "neq"
  | "in"
  | "gt"
  | "gte"
  | "lt"
  | "lte"
  | "matches"
  | "rollout"
  | "all"
  | "any"
  | "not"
  | "cel"
  | "always";

export interface Predicate {
  kind: PredicateKind;
  attr?: string;
  value?: unknown;
  values?: unknown[];
  pattern?: string;
  salt?: string;
  percent?: number;
  of?: Predicate[];
  of_one?: Predicate;
  source?: string;
}

export interface Rule {
  id: string;
  when: Predicate;
  value: unknown;
}

export interface RulesTree {
  value_type: ValueType;
  default: unknown;
  rules: Rule[];
}

export type DecisionReason =
  | "default"
  | "rule_matched"
  | "rollout_in_bucket"
  | "rollout_out_of_bucket"
  | "type_mismatch"
  | "error";

export interface Decision {
  value: unknown;
  reason: DecisionReason;
  rule_id?: string;
  version: number;
}
