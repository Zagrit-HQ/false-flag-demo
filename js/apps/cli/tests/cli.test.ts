import { PassThrough } from "node:stream";
import { describe, expect, it } from "vitest";
import { createProgram } from "../src/index.js";

function captureStdout(): {
  stream: NodeJS.WritableStream;
  collected: () => string;
} {
  const stream = new PassThrough();
  const chunks: Buffer[] = [];
  stream.on("data", (chunk: Buffer) => chunks.push(chunk));
  return {
    stream,
    collected: () => Buffer.concat(chunks).toString("utf8"),
  };
}

describe("falseflag CLI", () => {
  it("runs the health subcommand and prints JSON", async () => {
    const { stream, collected } = captureStdout();
    const program = createProgram({ out: stream });

    await program.parseAsync(["health"], { from: "user" });

    const output = collected();
    const parsed = JSON.parse(output.trim());
    expect(parsed).toEqual({
      status: "ok",
      cli: "falseflag",
      version: "0.1.0",
    });
  });

  it("exposes a version flag", () => {
    const program = createProgram();
    expect(program.version()).toBe("0.1.0");
  });

  it("project list prints rows from a mocked API response", async () => {
    const { stream, collected } = captureStdout();
    const errStream = new PassThrough();

    const fakeFetch: typeof fetch = async (url) => {
      expect(String(url)).toBe("http://api/v1/projects");
      return new Response(
        JSON.stringify({
          items: [
            {
              id: "00000000-0000-0000-0000-000000000001",
              slug: "demo",
              display_name: "Demo Project",
              config_strategy: "json",
              created_at: "2026-05-20T00:00:00Z",
              updated_at: "2026-05-20T00:00:00Z",
            },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    };

    const program = createProgram({
      out: stream,
      err: errStream,
      baseUrl: "http://api",
      fetch: fakeFetch,
    });
    await program.parseAsync(["project", "list"], { from: "user" });

    expect(collected().trim()).toBe("demo\tDemo Project");
  });

  it("flag list requires --project", async () => {
    const { stream, collected } = captureStdout();
    const errStream = new PassThrough();
    const fakeFetch: typeof fetch = async (url) => {
      expect(String(url)).toBe("http://api/v1/projects/demo/flags");
      return new Response(
        JSON.stringify({
          items: [
            {
              id: "00000000-0000-0000-0000-000000000002",
              project_id: "00000000-0000-0000-0000-000000000001",
              key: "checkout-v2",
              name: "Checkout V2",
              description: "",
              value_type: "boolean",
              default_value: false,
              created_at: "2026-05-20T00:00:00Z",
              updated_at: "2026-05-20T00:00:00Z",
            },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    };

    const program = createProgram({
      out: stream,
      err: errStream,
      baseUrl: "http://api",
      fetch: fakeFetch,
    });
    await program.parseAsync(["flag", "list", "--project", "demo"], {
      from: "user",
    });

    expect(collected().trim()).toBe("checkout-v2\tboolean\tCheckout V2");
  });
});
