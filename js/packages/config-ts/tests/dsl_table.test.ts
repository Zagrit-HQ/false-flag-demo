import { describe, expect, it } from "vitest";

import {
  FalseFlag,
  all,
  always,
  any,
  cel,
  eq,
  flag,
  gt,
  gte,
  isIn,
  lt,
  lte,
  matches,
  neq,
  not,
  rollout,
  rule,
} from "../src/index.js";

// Many small unit tests on every builder. The volume is deliberate:
// shard-ready CI demos benefit from many independent test cases.

describe("predicate builders return correct shape", () => {
  it.each([
    ["eq", () => eq("k", "v"), { kind: "eq", attr: "k", value: "v" }],
    ["neq", () => neq("k", "v"), { kind: "neq", attr: "k", value: "v" }],
    [
      "in",
      () => isIn("k", ["a", "b"]),
      { kind: "in", attr: "k", values: ["a", "b"] },
    ],
    ["gt", () => gt("k", 1), { kind: "gt", attr: "k", value: 1 }],
    ["gte", () => gte("k", 1), { kind: "gte", attr: "k", value: 1 }],
    ["lt", () => lt("k", 1), { kind: "lt", attr: "k", value: 1 }],
    ["lte", () => lte("k", 1), { kind: "lte", attr: "k", value: 1 }],
    [
      "matches",
      () => matches("k", "^x"),
      { kind: "matches", attr: "k", pattern: "^x" },
    ],
    [
      "rollout",
      () => rollout("user.id", "salt", 50),
      { kind: "rollout", attr: "user.id", salt: "salt", percent: 50 },
    ],
    [
      "cel",
      () => cel("ctx.user.age >= 18"),
      { kind: "cel", source: "ctx.user.age >= 18" },
    ],
    ["always", () => always(), { kind: "always" }],
  ])("%s", (_name, make, want) => {
    expect(make()).toEqual(want);
  });
});

describe("predicate builders accept many value types", () => {
  it.each([true, false, 0, 1, -3.14, "", "alpha", "漢字", null])(
    "eq accepts %p",
    (v) => {
      const n = eq("k", v);
      expect(n.kind).toBe("eq");
      expect(n.attr).toBe("k");
      expect(n.value).toEqual(v);
    },
  );
  it.each([
    [[]],
    [["a"]],
    [["a", "b", "c"]],
    [[1, 2, 3]],
    [[true, false]],
    [["mixed", 42, true, null]],
  ])("in accepts %p", (vs) => {
    expect(isIn("k", vs)).toEqual({ kind: "in", attr: "k", values: vs });
  });
});

describe("rollout boundaries", () => {
  it.each([0, 1, 5, 10, 25, 33, 50, 67, 75, 90, 99, 100])("percent=%i", (p) => {
    const r = rollout("user.id", "s", p);
    expect(r.percent).toBe(p);
    expect(r.attr).toBe("user.id");
    expect(r.salt).toBe("s");
  });
});

describe("logical combinators", () => {
  it("all wraps children in of[]", () => {
    expect(all(eq("a", 1), eq("b", 2))).toEqual({
      kind: "all",
      of: [
        { kind: "eq", attr: "a", value: 1 },
        { kind: "eq", attr: "b", value: 2 },
      ],
    });
  });

  it("any wraps children in of[]", () => {
    expect(any(always())).toEqual({ kind: "any", of: [{ kind: "always" }] });
  });

  it("not wraps in of_one", () => {
    expect(not(eq("k", "v"))).toEqual({
      kind: "not",
      of_one: { kind: "eq", attr: "k", value: "v" },
    });
  });

  it.each([1, 2, 3, 5, 10, 25, 50])("all with %i children", (n) => {
    const kids = Array.from({ length: n }, (_, i) => eq("k", `v${i}`));
    const node = all(...kids);
    expect(node.kind).toBe("all");
    const of = (node as unknown as { of: unknown[] }).of;
    expect(of).toHaveLength(n);
  });

  it("nesting all/any/not survives", () => {
    const n = all(any(eq("a", 1), eq("b", 2)), not(always()));
    expect(JSON.parse(JSON.stringify(n))).toEqual(n);
  });
});

describe("rule + flag", () => {
  it("rule returns a stable shape", () => {
    expect(rule("id", always(), 42)).toEqual({
      id: "id",
      when: { kind: "always" },
      value: 42,
    });
  });
  it.each(["boolean", "string", "number", "object"] as const)(
    "flag accepts value_type=%s",
    (vt) => {
      const f = flag({
        value_type: vt,
        default:
          vt === "object"
            ? {}
            : vt === "boolean"
              ? false
              : vt === "number"
                ? 0
                : "",
        rules: [],
      });
      expect(f.value_type).toBe(vt);
    },
  );

  it("flag preserves rule order", () => {
    const f = flag({
      value_type: "boolean",
      default: false,
      rules: [
        rule("r1", always(), true),
        rule("r2", eq("a", 1), false),
        rule("r3", not(always()), true),
      ],
    });
    expect(f.rules.map((r) => r.id)).toEqual(["r1", "r2", "r3"]);
  });

  it.each([10, 50, 100, 500])("flag accepts %i rules", (n) => {
    const rules = Array.from({ length: n }, (_, i) =>
      rule(`r${i}`, eq("k", i), true),
    );
    const f = flag({ value_type: "boolean", default: false, rules });
    expect(f.rules).toHaveLength(n);
  });
});

describe("FalseFlag namespace mirrors named exports", () => {
  it("FalseFlag.flag is flag", () => {
    expect(FalseFlag.flag).toBe(flag);
  });
  it("FalseFlag.in is the rename of isIn", () => {
    expect(FalseFlag.in).toBe(isIn);
  });
  it.each([
    "eq",
    "neq",
    "gt",
    "gte",
    "lt",
    "lte",
    "matches",
    "rollout",
    "all",
    "any",
    "not",
    "cel",
    "always",
    "rule",
  ] as const)("FalseFlag.%s exists", (k) => {
    expect(typeof FalseFlag[k]).toBe("function");
  });
});

describe("JSON-serialisable output (no closures)", () => {
  it.each([
    () => flag({ value_type: "boolean", default: false, rules: [] }),
    () =>
      flag({
        value_type: "string",
        default: "x",
        rules: [rule("r", always(), "y")],
      }),
    () =>
      flag({
        value_type: "boolean",
        default: false,
        rules: [
          rule("r", all(eq("a", 1), any(gt("b", 2), not(lt("c", 3)))), true),
        ],
      }),
    () =>
      flag({
        value_type: "boolean",
        default: false,
        rules: [rule("r", rollout("user.id", "salt", 25), true)],
      }),
    () =>
      flag({
        value_type: "object",
        default: { mode: "light" },
        rules: [rule("r", eq("user.prefers", "dark"), { mode: "dark" })],
      }),
  ])("doc %#", (make) => {
    const f = make();
    expect(JSON.parse(JSON.stringify(f))).toEqual(f);
  });
});
