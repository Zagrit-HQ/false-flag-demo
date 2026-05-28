import { describe, expect, it } from "vitest";
import * as client from "../src/index.js";

describe("@falseflag/generated-client", () => {
  it("exports at least one function from the generated client", () => {
    const exportedNames = Object.keys(client);
    expect(exportedNames.length).toBeGreaterThan(0);
  });
});
