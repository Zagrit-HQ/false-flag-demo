import { describe, expect, it } from "vitest";

import { loader } from "../app/routes/_index";

describe("dashboard / redirect", () => {
  it("redirects / to /projects", async () => {
    const res = await loader();
    expect(res).toBeInstanceOf(Response);
    expect((res as Response).status).toBe(302);
    expect((res as Response).headers.get("Location")).toBe("/projects");
  });
});
