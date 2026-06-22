# FalseFlag SDK — OpenFeature-Shaped Provider Contract

This document is the contract both FalseFlag SDKs (Go `internal/sdkgo`
and TypeScript `@falseflag/sdk`) must satisfy. It is "OpenFeature-shaped",
not full OpenFeature spec compliance — the demo doesn't ship hooks, the
provider registry, finally-stage logic, telemetry channels, or the
client/provider/hooks separation. We ship the four resolution methods
audiences recognize.

## EvaluationContext

The context passed to every evaluation is a flat map. Both SDKs accept:

```ts
type EvaluationContext = {
  targetingKey?: string;     // commonly the user id, used for rollout bucketing
  [attribute: string]: unknown;
};
```

Go equivalent:

```go
type EvalContext map[string]any
```

The IR (see `internal/config.RulesTree`) reads attributes as a nested
map: `user.id`, `user.plan`, `request.country`, etc. SDKs do **not**
flatten the context — callers pass nested objects as-is. The CEL-lite
evaluator (`internal/eval/predicates.go`,
`js/packages/sdk-js/src/cel-lite.ts`) walks the nested shape.

## Decision

Every resolution returns a `Decision`. The Decision shape is the same
across both runtimes (slice 2 enforces byte-identical Decision JSON
through `tests/eval-corpus/`).

```ts
type Decision = {
  value: boolean | string | number | object;
  reason: DecisionReason;
  rule_id?: string;
  version?: number;
};

type DecisionReason =
  | "default"          // no rule matched, returned the default
  | "rule_matched"     // a predicate evaluated true
  | "rollout_in_bucket"// rollout predicate caught the bucket
  | "type_mismatch"    // matched rule produced a value of the wrong type
  | "error";           // evaluator raised an error; default returned
```

## Resolution Methods

Both SDKs expose four methods, one per value type:

| Method | Default value type | Return type |
|---|---|---|
| `booleanValue` / `BooleanValue` | `boolean` | `boolean` |
| `stringValue` / `StringValue`   | `string`  | `string`  |
| `numberValue` / `NumberValue`   | `number`  | `number`  |
| `objectValue` / `ObjectValue`   | `T`       | `T`       |

If the SDK has no snapshot loaded, methods return the default with
`reason: "error"` (TS) or `Reason: "error"` (Go) and log a warning.
If the flag is missing from the snapshot, methods return the default
with `reason: "default"`.

## Provider Interface

The thin wrapper exposed for OpenFeature-style integration. TypeScript:

```ts
export interface FalseFlagProviderMetadata {
  readonly name: string;
}

export interface FalseFlagProvider {
  readonly metadata: FalseFlagProviderMetadata;
  resolveBooleanEvaluation(
    flagKey: string,
    defaultValue: boolean,
    context: EvaluationContext,
  ): Promise<Decision>;
  resolveStringEvaluation(
    flagKey: string,
    defaultValue: string,
    context: EvaluationContext,
  ): Promise<Decision>;
  resolveNumberEvaluation(
    flagKey: string,
    defaultValue: number,
    context: EvaluationContext,
  ): Promise<Decision>;
  resolveObjectEvaluation<T>(
    flagKey: string,
    defaultValue: T,
    context: EvaluationContext,
  ): Promise<Decision>;
}
```

Go equivalent:

```go
type ProviderMetadata struct {
    Name string
}

type Provider interface {
    Metadata() ProviderMetadata
    BooleanEvaluation(ctx context.Context, key string, def bool, evalCtx EvalContext) Decision
    StringEvaluation(ctx context.Context, key string, def string, evalCtx EvalContext) Decision
    NumberEvaluation(ctx context.Context, key string, def float64, evalCtx EvalContext) Decision
    ObjectEvaluation(ctx context.Context, key string, def any, evalCtx EvalContext) Decision
}
```

## Polling Cadence

Both SDKs poll `GET /v1/projects/{slug}/snapshots/latest` on a
configurable interval (default 10 seconds). The polled snapshot
contains `{compiled: {flags: {[key]: RulesTree}}}`. SDKs cache the
parsed snapshot in memory and serve evaluations locally between polls.

If a poll fails (network error, 5xx), the SDK keeps the last good
snapshot and logs a warning. If the first poll never succeeds, the SDK
serves the configured default for every evaluation with `reason:
"error"`.

## Things This Document Deliberately Does Not Specify

- **Hooks.** OpenFeature's `before`, `after`, `finally`, `error` hooks
  are not implemented.
- **Provider registry.** Both SDKs are a single provider; there is no
  `OpenFeature.setProvider()` global.
- **Tracking events / telemetry.** No `onResolutionDetails` callbacks.
- **TargetingMissingError, ParseError, TypeMismatchError.** Errors are
  collapsed into `reason: "error"` for simplicity.
- **Object value schema.** The SDK does not validate the structural
  shape of object-typed flag values against a schema; consumers cast.

These can be added in a future slice if the demo grows past
"OpenFeature-shaped" into "OpenFeature-compliant".
