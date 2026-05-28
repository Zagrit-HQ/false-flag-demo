// createClient is the OpenFeature-shaped entry point. It can run in
// two modes:
//
//   1. Per-flag mode (default). Each evaluate() call may fetch the
//      flag's IR from /v1/projects/{slug}/flags/{key} with a short
//      in-memory cache. Convenient for dashboards, CLIs, and ad-hoc
//      consumers.
//
//   2. Snapshot mode. After start() is called the client polls
//      /v1/projects/{slug}/snapshots/latest on `pollIntervalMs`
//      (default 10s) and serves every evaluation from the cached
//      snapshot. This is the mode the proxy and long-running
//      services use.
//
// Cross-runtime golden tests don't go through this layer — they call
// evaluate() directly in evaluator.ts so the test stays network-free.

import { type Compiled, compile, evaluate } from "./evaluator.js";
import type { Decision, DecisionReason, RulesTree } from "./ir.js";

export interface FalseFlagClientOptions {
  /** Base URL of the FalseFlag control plane. */
  baseUrl: string;
  /** Project slug; flag fetches are scoped to it. */
  projectSlug: string;
  /** Optional default evaluation context merged into each call. */
  defaultContext?: Record<string, unknown>;
  /**
   * Snapshot poll interval in milliseconds. Only used after start().
   * Defaults to 10_000 (10s). Set to 0 to disable background polling
   * entirely; the first poll still runs at start().
   */
  pollIntervalMs?: number;
  /**
   * Optional fetch override (useful for tests). Defaults to globalThis.fetch.
   */
  fetch?: typeof fetch;
  /**
   * Optional logger (useful for tests). Defaults to console.warn.
   */
  warn?: (msg: string) => void;
}

export interface SnapshotInfo {
  /** Snapshot id (UUID) returned by the API. */
  id: string;
  /** Snapshot version. */
  version: number;
  /** ISO-8601 timestamp the snapshot was created. */
  createdAt: string;
  /** Per-flag compiled IR keyed by flag key. */
  flags: Record<string, Compiled>;
}

export interface FalseFlagClient {
  readonly baseUrl: string;
  readonly projectSlug: string;
  /** Start snapshot polling. Resolves once the first poll lands. */
  start(): Promise<void>;
  /** Stop snapshot polling. Safe to call multiple times. */
  stop(): void;
  /** Current snapshot (or null if start() has not yet succeeded). */
  getSnapshot(): SnapshotInfo | null;
  getBooleanValue(
    flag: string,
    defaultValue: boolean,
    ctx?: Record<string, unknown>,
  ): Promise<boolean>;
  getStringValue(
    flag: string,
    defaultValue: string,
    ctx?: Record<string, unknown>,
  ): Promise<string>;
  getNumberValue(
    flag: string,
    defaultValue: number,
    ctx?: Record<string, unknown>,
  ): Promise<number>;
  getObjectValue<T>(
    flag: string,
    defaultValue: T,
    ctx?: Record<string, unknown>,
  ): Promise<T>;
  /** Returns the full decision (value + reason + rule_id + version). */
  evaluate(flag: string, ctx?: Record<string, unknown>): Promise<Decision>;
}

interface CacheEntry {
  compiled: Compiled;
  version: number;
  fetchedAt: number;
}

const PER_FLAG_CACHE_TTL_MS = 5_000;
const DEFAULT_POLL_INTERVAL_MS = 10_000;

export function createClient(opts: FalseFlagClientOptions): FalseFlagClient {
  if (!opts.baseUrl) throw new Error("FalseFlag SDK: baseUrl is required");
  if (!opts.projectSlug)
    throw new Error("FalseFlag SDK: projectSlug is required");

  const realFetch = opts.fetch ?? globalThis.fetch;
  const warn = opts.warn ?? ((m: string) => console.warn(m));
  const perFlagCache = new Map<string, CacheEntry>();
  let snapshot: SnapshotInfo | null = null;
  let pollTimer: ReturnType<typeof setTimeout> | null = null;
  let stopped = false;

  async function pollSnapshot(): Promise<void> {
    const url = `${opts.baseUrl}/v1/projects/${opts.projectSlug}/snapshots/latest`;
    const resp = await realFetch(url);
    if (resp.status === 404) {
      // No snapshot compiled yet. Keep last-good if any.
      if (!snapshot) warn(`FalseFlag SDK: no snapshot for ${opts.projectSlug}`);
      return;
    }
    if (!resp.ok) {
      throw new Error(`FalseFlag SDK: fetch ${url} → ${resp.status}`);
    }
    const body = (await resp.json()) as {
      id: string;
      version: number;
      created_at: string;
      compiled: { flags: Record<string, RulesTree> };
    };
    const flags: Record<string, Compiled> = {};
    for (const [key, ir] of Object.entries(body.compiled.flags ?? {})) {
      flags[key] = compile(ir);
    }
    snapshot = {
      id: body.id,
      version: body.version,
      createdAt: body.created_at,
      flags,
    };
  }

  function scheduleNextPoll(): void {
    if (stopped) return;
    const interval = opts.pollIntervalMs ?? DEFAULT_POLL_INTERVAL_MS;
    if (interval <= 0) return;
    pollTimer = setTimeout(() => {
      pollSnapshot()
        .catch((err) => warn(`FalseFlag SDK: poll error: ${String(err)}`))
        .finally(scheduleNextPoll);
    }, interval);
    // Allow Node to exit even if the timer is still pending. The
    // returned timer object only has unref() in Node; browsers return
    // a number. We unref defensively without committing to either type.
    const t = pollTimer as unknown as { unref?: () => void };
    if (typeof t.unref === "function") t.unref();
  }

  async function loadFlagPerFlag(flag: string): Promise<CacheEntry | null> {
    const now = Date.now();
    const cached = perFlagCache.get(flag);
    if (cached && now - cached.fetchedAt < PER_FLAG_CACHE_TTL_MS) return cached;
    const url = `${opts.baseUrl}/v1/projects/${opts.projectSlug}/flags/${flag}`;
    const resp = await realFetch(url);
    if (resp.status === 404) return null;
    if (!resp.ok)
      throw new Error(`FalseFlag SDK: fetch ${url} → ${resp.status}`);
    const body = (await resp.json()) as {
      flag: { value_type: string };
      latest_version?: { version: number; compiled: unknown };
    };
    if (!body.latest_version) return null;
    const ir = body.latest_version.compiled as RulesTree;
    const entry: CacheEntry = {
      compiled: compile(ir),
      version: body.latest_version.version,
      fetchedAt: now,
    };
    perFlagCache.set(flag, entry);
    return entry;
  }

  async function evaluateFlag(
    flag: string,
    ctx?: Record<string, unknown>,
  ): Promise<Decision> {
    const mergedCtx = { ...(opts.defaultContext ?? {}), ...(ctx ?? {}) };
    // Prefer the snapshot if loaded.
    if (snapshot) {
      const compiled = snapshot.flags[flag];
      if (!compiled) {
        return defaultDecision(null, "default", snapshot.version);
      }
      return evaluate(compiled, mergedCtx, snapshot.version);
    }
    // Fall back to per-flag fetch.
    const entry = await loadFlagPerFlag(flag);
    if (!entry) {
      return defaultDecision(null);
    }
    return evaluate(entry.compiled, mergedCtx, entry.version);
  }

  function defaultDecision<T>(
    value: T,
    reason: DecisionReason = "default",
    version = 0,
  ): Decision {
    return { value, reason, version };
  }

  function coerce<T>(
    d: Decision,
    fallback: T,
    predicate: (v: unknown) => boolean,
  ): T {
    return predicate(d.value) ? (d.value as T) : fallback;
  }

  return {
    baseUrl: opts.baseUrl,
    projectSlug: opts.projectSlug,
    async start() {
      stopped = false;
      await pollSnapshot();
      scheduleNextPoll();
    },
    stop() {
      stopped = true;
      if (pollTimer) {
        clearTimeout(pollTimer);
        pollTimer = null;
      }
    },
    getSnapshot() {
      return snapshot;
    },
    async evaluate(flag, ctx) {
      return evaluateFlag(flag, ctx);
    },
    async getBooleanValue(flag, defaultValue, ctx) {
      return coerce(
        await evaluateFlag(flag, ctx),
        defaultValue,
        (v) => typeof v === "boolean",
      );
    },
    async getStringValue(flag, defaultValue, ctx) {
      return coerce(
        await evaluateFlag(flag, ctx),
        defaultValue,
        (v) => typeof v === "string",
      );
    },
    async getNumberValue(flag, defaultValue, ctx) {
      return coerce(
        await evaluateFlag(flag, ctx),
        defaultValue,
        (v) => typeof v === "number",
      );
    },
    async getObjectValue<T>(
      flag: string,
      defaultValue: T,
      ctx?: Record<string, unknown>,
    ): Promise<T> {
      return coerce(
        await evaluateFlag(flag, ctx),
        defaultValue,
        (v) => typeof v === "object" && v !== null && !Array.isArray(v),
      ) as T;
    },
  };
}
