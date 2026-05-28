import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@remix-run/react", async () => {
  const actual = await vi.importActual<object>("@remix-run/react");
  return {
    ...actual,
    useLoaderData: () => ({
      projects: [
        {
          id: "p-1",
          slug: "demo",
          display_name: "Demo Project",
          config_strategy: "json",
          created_at: "2026-05-20T00:00:00Z",
          updated_at: "2026-05-20T00:00:00Z",
        },
      ],
    }),
    Link: ({ children, to }: { children: React.ReactNode; to: string }) => (
      <a href={to}>{children}</a>
    ),
  };
});

import ProjectsIndex from "../app/routes/projects._index";

describe("dashboard projects index", () => {
  it("renders the projects heading and seeded items", () => {
    render(<ProjectsIndex />);
    expect(
      screen.getByRole("heading", { name: /^Projects$/ }),
    ).toBeInTheDocument();
    expect(screen.getByTestId("projects")).toHaveTextContent("Demo Project");
    expect(screen.getByTestId("strategy-json")).toBeInTheDocument();
  });
});
