// FalseFlagProvider wraps createClient() in the OpenFeature-shaped
// provider interface documented in docs/sdk-openfeature.md. It is
// intentionally thin — the four resolveXxxEvaluation methods delegate
// to client.evaluate() and coerce the value to the requested type.

import {
  type FalseFlagClient,
  type FalseFlagClientOptions,
  createClient,
} from "./client.js";
import type { Decision } from "./ir.js";

export interface EvaluationContext extends Record<string, unknown> {
  targetingKey?: string;
}

export interface FalseFlagProviderMetadata {
  readonly name: string;
}

export interface FalseFlagProvider {
  readonly metadata: FalseFlagProviderMetadata;
  /** Underlying client. Useful for tests and for calling start()/stop(). */
  readonly client: FalseFlagClient;
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

export interface FalseFlagProviderOptions extends FalseFlagClientOptions {
  /** Optional provider name. Defaults to "falseflag". */
  name?: string;
  /** Optional pre-built client; useful for tests. */
  client?: FalseFlagClient;
}

export function createProvider(
  opts: FalseFlagProviderOptions,
): FalseFlagProvider {
  const client = opts.client ?? createClient(opts);
  const name = opts.name ?? "falseflag";

  function typed(
    value: unknown,
    ok: (v: unknown) => boolean,
    fallback: unknown,
    d: Decision,
  ): Decision {
    if (ok(value)) return d;
    return { ...d, value: fallback, reason: "type_mismatch" };
  }

  return {
    metadata: { name },
    client,
    async resolveBooleanEvaluation(key, defaultValue, context) {
      const d = await client.evaluate(key, context);
      return typed(d.value, (v) => typeof v === "boolean", defaultValue, d);
    },
    async resolveStringEvaluation(key, defaultValue, context) {
      const d = await client.evaluate(key, context);
      return typed(d.value, (v) => typeof v === "string", defaultValue, d);
    },
    async resolveNumberEvaluation(key, defaultValue, context) {
      const d = await client.evaluate(key, context);
      return typed(d.value, (v) => typeof v === "number", defaultValue, d);
    },
    async resolveObjectEvaluation<T>(
      key: string,
      defaultValue: T,
      context: EvaluationContext,
    ) {
      const d = await client.evaluate(key, context);
      return typed(
        d.value,
        (v) => typeof v === "object" && v !== null,
        defaultValue,
        d,
      );
    },
  };
}
