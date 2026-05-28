// @falseflag/sdk — TypeScript SDK for FalseFlag.
//
// The headline API is createClient({baseUrl, projectSlug}) which
// returns an OpenFeature-shaped object that fetches the latest IR
// from the API and evaluates locally. For tests and library
// consumers the lower-level compile/evaluate helpers are also
// exported.

export type { Compiled } from "./evaluator.js";
export { compile, evaluate } from "./evaluator.js";

export type {
  Decision,
  DecisionReason,
  Predicate,
  PredicateKind,
  Rule,
  RulesTree,
  Strategy,
  ValueType,
} from "./ir.js";

export { rolloutBucket, inBucket } from "./rollout.js";

export {
  compileCEL,
  evalCEL,
  type CelProgram,
} from "./cel-lite.js";

export type {
  FalseFlagClient,
  FalseFlagClientOptions,
  SnapshotInfo,
} from "./client.js";
export { createClient } from "./client.js";

export type {
  EvaluationContext,
  FalseFlagProvider,
  FalseFlagProviderMetadata,
  FalseFlagProviderOptions,
} from "./provider.js";
export { createProvider } from "./provider.js";
