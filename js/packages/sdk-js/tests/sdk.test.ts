import { describe, expect, it } from "vitest";
import { compile, createClient, evaluate } from "../src/index.js";
import type { RulesTree } from "../src/index.js";

describe("createClient", () => {
  it("rejects empty baseUrl", () => {
    expect(() => createClient({ baseUrl: "", projectSlug: "demo" })).toThrow(
      /baseUrl/,
    );
  });

  it("rejects empty projectSlug", () => {
    expect(() =>
      createClient({ baseUrl: "http://x", projectSlug: "" }),
    ).toThrow(/projectSlug/);
  });

  it("exposes the configured baseUrl and project", () => {
    const c = createClient({
      baseUrl: "http://proxy.example",
      projectSlug: "demo",
    });
    expect(c.baseUrl).toBe("http://proxy.example");
    expect(c.projectSlug).toBe("demo");
  });
});

describe("compile + evaluate (no network)", () => {
  const ir: RulesTree = {
    value_type: "boolean",
    default: false,
    rules: [
      {
        id: "r1",
        when: { kind: "eq", attr: "user.plan", value: "pro" },
        value: true,
      },
    ],
  };

  it("returns the default when no rules match", () => {
    const d = evaluate(compile(ir), { user: { plan: "free" } }, 1);
    expect(d.value).toBe(false);
    expect(d.reason).toBe("default");
  });

  it("returns the matched rule value", () => {
    const d = evaluate(compile(ir), { user: { plan: "pro" } }, 7);
    expect(d.value).toBe(true);
    expect(d.reason).toBe("rule_matched");
    expect(d.rule_id).toBe("r1");
    expect(d.version).toBe(7);
  });
});
