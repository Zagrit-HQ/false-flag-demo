import { describe, expect, it } from "vitest";

import { compile, evaluate } from "../src/evaluator.js";
import type { Predicate, RulesTree } from "../src/ir.js";

function flag(
  p: Predicate,
  value: unknown = true,
  defaultValue: unknown = false,
): RulesTree {
  return {
    value_type: "boolean",
    default: defaultValue,
    rules: [{ id: "r", when: p, value }],
  };
}

describe("evaluate — default path", () => {
  it("returns default for empty rules", () => {
    const ir: RulesTree = { value_type: "boolean", default: false, rules: [] };
    const d = evaluate(compile(ir), {});
    expect(d.value).toBe(false);
    expect(d.reason).toBe("default");
  });

  it.each([false, true, "", "x", 0, 1, -3, [], {}, null])(
    "preserves default %p",
    (def) => {
      const ir: RulesTree = { value_type: "boolean", default: def, rules: [] };
      const d = evaluate(compile(ir), {});
      expect(d.value).toEqual(def);
    },
  );
});

describe("evaluate — eq", () => {
  it.each([
    [{ user: { plan: "pro" } }, true],
    [{ user: { plan: "free" } }, false],
    [{ user: {} }, false],
    [{}, false],
  ])("ctx=%j → matches=%p", (ctx, want) => {
    const d = evaluate(
      compile(flag({ kind: "eq", attr: "user.plan", value: "pro" })),
      ctx,
    );
    expect(d.value).toBe(want);
  });
});

describe("evaluate — in", () => {
  const ir = flag({ kind: "in", attr: "plan", values: ["pro", "ent"] });
  it.each([
    ["pro", true],
    ["ent", true],
    ["free", false],
    ["trial", false],
  ])("plan=%s matches=%p", (plan, want) => {
    const d = evaluate(compile(ir), { plan });
    expect(d.value).toBe(want);
  });
});

describe("evaluate — ordering predicates", () => {
  const ages = [-1, 0, 17, 18, 19, 21, 50, 100];
  const cases: Array<{
    kind: Predicate["kind"];
    bound: number;
    fn: (a: number, b: number) => boolean;
  }> = [
    { kind: "gt", bound: 18, fn: (a, b) => a > b },
    { kind: "gte", bound: 18, fn: (a, b) => a >= b },
    { kind: "lt", bound: 18, fn: (a, b) => a < b },
    { kind: "lte", bound: 18, fn: (a, b) => a <= b },
  ];
  for (const { kind, bound, fn } of cases) {
    describe(`${kind}`, () => {
      it.each(ages)("age=%i", (age) => {
        const d = evaluate(compile(flag({ kind, attr: "age", value: bound })), {
          age,
        });
        expect(d.value).toBe(fn(age, bound));
      });
    });
  }
});

describe("evaluate — matches", () => {
  it.each([
    ["^a", "alpha", true],
    ["^a", "beta", false],
    ["z$", "buzz", true],
    ["lph", "alpha", true],
    ["^foo$", "foo", true],
    ["^foo$", "foobar", false],
  ])("pattern=%s val=%s matches=%p", (pattern, val, want) => {
    const d = evaluate(compile(flag({ kind: "matches", attr: "k", pattern })), {
      k: val,
    });
    expect(d.value).toBe(want);
  });
});

describe("evaluate — rule order and short-circuit", () => {
  it("picks the first matching rule", () => {
    const ir: RulesTree = {
      value_type: "string",
      default: "default",
      rules: [
        {
          id: "r1",
          when: { kind: "eq", attr: "k", value: "v" },
          value: "first",
        },
        {
          id: "r2",
          when: { kind: "eq", attr: "k", value: "v" },
          value: "second",
        },
      ],
    };
    const d = evaluate(compile(ir), { k: "v" });
    expect(d.value).toBe("first");
    expect(d.rule_id).toBe("r1");
  });

  it.each([
    ["a", "rule-a"],
    ["b", "rule-b"],
    ["c", "rule-c"],
    ["d", "default"],
  ])("k=%s -> %s", (k, want) => {
    const ir: RulesTree = {
      value_type: "string",
      default: "default",
      rules: [
        {
          id: "r-a",
          when: { kind: "eq", attr: "k", value: "a" },
          value: "rule-a",
        },
        {
          id: "r-b",
          when: { kind: "eq", attr: "k", value: "b" },
          value: "rule-b",
        },
        {
          id: "r-c",
          when: { kind: "eq", attr: "k", value: "c" },
          value: "rule-c",
        },
      ],
    };
    const d = evaluate(compile(ir), { k });
    expect(d.value).toBe(want);
  });
});

describe("evaluate — version passthrough", () => {
  it.each([1, 2, 3, 5, 7, 10, 100, 1000])("v=%i", (v) => {
    const ir: RulesTree = { value_type: "boolean", default: false, rules: [] };
    const d = evaluate(compile(ir), {}, v);
    expect(d.version).toBe(v);
  });
});

describe("evaluate — boolean/all/any/not nesting", () => {
  it.each([
    [{ a: 1, b: 1 }, true],
    [{ a: 1, b: 2 }, false],
    [{ a: 2, b: 1 }, false],
  ])("all(eq,eq) ctx=%j -> %p", (ctx, want) => {
    const d = evaluate(
      compile(
        flag({
          kind: "all",
          of: [
            { kind: "eq", attr: "a", value: 1 },
            { kind: "eq", attr: "b", value: 1 },
          ],
        }),
      ),
      ctx,
    );
    expect(d.value).toBe(want);
  });

  it.each([
    [{ a: 1 }, true],
    [{ b: 1 }, true],
    [{ c: 1 }, false],
  ])("any(eq,eq) ctx=%j -> %p", (ctx, want) => {
    const d = evaluate(
      compile(
        flag({
          kind: "any",
          of: [
            { kind: "eq", attr: "a", value: 1 },
            { kind: "eq", attr: "b", value: 1 },
          ],
        }),
      ),
      ctx,
    );
    expect(d.value).toBe(want);
  });

  it.each([
    [{ trial: true }, false],
    [{ trial: false }, true],
    [{}, true],
  ])("not(eq trial=true) ctx=%j -> %p", (ctx, want) => {
    const d = evaluate(
      compile(
        flag({
          kind: "not",
          of_one: { kind: "eq", attr: "trial", value: true },
        }),
      ),
      ctx,
    );
    expect(d.value).toBe(want);
  });
});

describe("evaluate — deep attribute lookup", () => {
  it.each([
    ["a", { a: "v" }, true],
    ["a.b", { a: { b: "v" } }, true],
    ["a.b.c", { a: { b: { c: "v" } } }, true],
    ["a.b.c.d", { a: { b: { c: { d: "v" } } } }, true],
    ["a.b.c.d.e", { a: { b: { c: { d: { e: "v" } } } } }, true],
    ["a.b", { a: {} }, false],
    ["a.b", {}, false],
  ])("attr=%s ctx=%j -> %p", (attr, ctx, want) => {
    const d = evaluate(
      compile(flag({ kind: "eq", attr, value: "v" })),
      ctx as Record<string, unknown>,
    );
    expect(d.value).toBe(want);
  });
});

describe("evaluate — many flags", () => {
  // Exercise the evaluator across many flags compiled in one pass.
  for (let i = 0; i < 25; i++) {
    it(`flag-${i} matches its rule`, () => {
      const d = evaluate(
        compile(flag({ kind: "eq", attr: "k", value: `v${i}` })),
        { k: `v${i}` },
      );
      expect(d.value).toBe(true);
    });
  }
});
