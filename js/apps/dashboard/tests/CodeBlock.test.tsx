import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { CodeBlock } from "~/components/CodeBlock";

describe("CodeBlock", () => {
  it("renders Shiki HTML when provided", () => {
    const { container } = render(
      <CodeBlock html="<pre><code>hello</code></pre>" />,
    );
    expect(screen.getByTestId("latest-version").innerHTML).toBe(
      "<pre><code>hello</code></pre>",
    );
    expect(
      container.querySelector('[data-testid="latest-version-caption"]'),
    ).toBeNull();
  });

  it("falls back to plain <pre> with caption when no html is supplied", () => {
    render(
      <CodeBlock
        fallbackJson={'{"a":1}'}
        caption="compiled IR — original source not stored"
      />,
    );
    expect(screen.getByTestId("latest-version").textContent).toBe('{"a":1}');
    expect(screen.getByTestId("latest-version-caption").textContent).toMatch(
      /compiled IR/,
    );
  });
});
