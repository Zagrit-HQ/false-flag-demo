// Shared types for the 100-fixture edit-corpus suite. Each fixture
// represents one parallel-safe e2e test: its own project, its own
// flag, its own edit round-trip.

export type Strategy = "cel" | "typescript";
export type ValueType = "boolean" | "string" | "number";

export interface CorpusFixture {
  /** Unique slug used for the project name and Playwright test title. */
  id: string;
  strategy: Strategy;
  valueType: ValueType;
  defaultValue: unknown;
  /** Source published in globalSetup so the edit page loads with the
   *  correct strategy and Monaco has something to show. */
  setupSource: string;
  /** IR for the API's required `source` field on the setup publish. */
  setupIR: unknown;
  /** Source the test will paste into Monaco and save. For CEL flags
   *  this is the IR JSON the dashboard's `javascript`-language Shiki
   *  highlighter renders; for TS flags this is a real `ff.flag(...)`
   *  block compiled server-side via esbuild+goja. */
  testSource: string;
  /** Unique substring from testSource asserted to appear in the
   *  rendered source-code text content after save. Pick something
   *  short and unambiguous; Playwright's toContainText reads textContent
   *  which collapses Shiki span boundaries, so substrings that span
   *  tokens are fine. */
  assertSubstring: string;
}

/** All fixtures (CEL + TS) the spec iterates over. */
export type Corpus = CorpusFixture[];
