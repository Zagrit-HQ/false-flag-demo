import { describe, expect, it } from "vitest";

import { compile, evaluate } from "@falseflag/sdk";

import { jsonEqualish, loadCorpus } from "../src/index.js";

describe("cross-runtime evaluation corpus", () => {
  const fixtures = loadCorpus();

  it("loads at least 12 fixtures", () => {
    expect(fixtures.length).toBeGreaterThanOrEqual(12);
  });

  for (const fx of fixtures) {
    it(`${fx.id}: JS evaluator agrees with expected Decision`, () => {
      const compiled = compile(fx.ir);
      const got = evaluate(compiled, fx.context, 1);

      expect(jsonEqualish(got.value, fx.expected.value)).toBe(true);
      expect(got.reason).toBe(fx.expected.reason);
      if (fx.expected.rule_id) {
        expect(got.rule_id).toBe(fx.expected.rule_id);
      } else {
        expect(got.rule_id).toBeUndefined();
      }
    });
  }
});
