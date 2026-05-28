import { describe, expect, it } from "vitest";

import { FalseFlag } from "../src/index.js";

describe("@falseflag/config DSL", () => {
  it("builds a flag-centric IR document with eq + in + rollout", () => {
    const doc = FalseFlag.flag({
      value_type: "boolean",
      default: false,
      rules: [
        FalseFlag.rule(
          "internal",
          FalseFlag.matches("user.email", ".+@depot\\.dev$"),
          true,
        ),
        FalseFlag.rule(
          "us-pro",
          FalseFlag.all(
            FalseFlag.eq("user.country", "US"),
            FalseFlag.in("user.plan", ["pro", "enterprise"]),
          ),
          true,
        ),
        FalseFlag.rule(
          "rollout-25",
          FalseFlag.rollout("user.id", "checkout-v2", 25),
          true,
        ),
      ],
    });

    expect(doc.value_type).toBe("boolean");
    expect(doc.default).toBe(false);
    expect(doc.rules).toHaveLength(3);
    expect(doc.rules[0]?.when).toMatchObject({ kind: "matches" });
    expect(doc.rules[1]?.when).toMatchObject({ kind: "all" });
    expect(doc.rules[2]?.when).toMatchObject({ kind: "rollout", percent: 25 });
  });

  it("returns plain data (JSON-serialisable, no closures)", () => {
    const doc = FalseFlag.flag({
      value_type: "string",
      default: "control",
      rules: [
        FalseFlag.rule(
          "cel-rule",
          FalseFlag.cel("ctx.user.age >= 21"),
          "treatment",
        ),
      ],
    });
    expect(() => JSON.stringify(doc)).not.toThrow();
    expect(JSON.parse(JSON.stringify(doc))).toEqual(doc);
  });

  it("composes any + not", () => {
    const pred = FalseFlag.any(
      FalseFlag.not(FalseFlag.eq("user.trial", true)),
      FalseFlag.always(),
    );
    expect(pred.kind).toBe("any");
    const children = pred.of as unknown[];
    expect(Array.isArray(children)).toBe(true);
    expect(children).toHaveLength(2);
  });
});
