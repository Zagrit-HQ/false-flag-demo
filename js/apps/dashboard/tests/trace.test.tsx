import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TraceTree } from "../app/components/TraceTree";

describe("TraceTree", () => {
  it("renders nothing useful for a missing trace", () => {
    render(<TraceTree trace={undefined} />);
    expect(screen.getByText(/no trace returned/i)).toBeInTheDocument();
  });

  it("highlights matched rules and renders nested predicates", () => {
    render(
      <TraceTree
        trace={{
          rules: [
            {
              id: "rule-pro",
              matched: true,
              when: {
                kind: "and",
                children: [
                  { kind: "eq", attr: "user.plan", matched: true },
                  { kind: "rollout", attr: "user.id", matched: true },
                ],
              },
            },
            { id: "rule-default", matched: false, when: { kind: "always" } },
          ],
        }}
      />,
    );
    expect(screen.getByTestId("trace-rules")).toBeInTheDocument();
    expect(screen.getByTestId("trace-rule-rule-pro")).toHaveTextContent(
      "matched",
    );
    expect(screen.getByTestId("trace-rule-rule-default")).toHaveTextContent(
      "skipped",
    );
  });
});
