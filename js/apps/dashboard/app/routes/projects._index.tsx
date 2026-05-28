import type { LoaderFunctionArgs, MetaFunction } from "@remix-run/node";
import { Link, useLoaderData } from "@remix-run/react";

import { type Project, listProjects, zod } from "@falseflag/generated-client";

import { EmptyState, ErrorBanner } from "~/components/ErrorBanner";
import { Page } from "~/components/Nav";
import { StrategyBadge } from "~/components/StrategyBadge";
import { withApiFetch } from "~/lib/api.server";

export const meta: MetaFunction = () => [{ title: "Projects · FalseFlag" }];

interface LoaderData {
  projects: Project[];
  error?: string;
}

export async function loader(_: LoaderFunctionArgs): Promise<LoaderData> {
  return withApiFetch(async () => {
    try {
      const res = await listProjects();
      if (res.status !== 200) {
        return { projects: [], error: `HTTP ${res.status}` };
      }
      const parsed = zod.listProjectsResponse.safeParse(res.data);
      if (!parsed.success) return { projects: [], error: parsed.error.message };
      return { projects: parsed.data.items };
    } catch (e) {
      return { projects: [], error: (e as Error).message };
    }
  });
}

export default function ProjectsIndex() {
  const { projects, error } = useLoaderData<typeof loader>();
  return (
    <Page crumbs={[{ to: "/projects", label: "Projects" }]}>
      <h1 className="text-2xl font-bold">Projects</h1>
      <p className="mt-1 text-sm text-falseflag-900/70">
        Every project lives in a single configuration strategy. Click into one
        to see its flags, environments, snapshots, and audit log.
      </p>
      <ErrorBanner error={error} />
      {projects.length === 0 && !error ? (
        <div className="mt-6">
          <EmptyState message="No projects yet. Use the CLI: `falseflag project list` or seed the database with `make seed`." />
        </div>
      ) : (
        <ul
          className="mt-6 divide-y divide-gray-200 rounded-md border border-gray-200 bg-white"
          data-testid="projects"
        >
          {projects.map((p) => (
            <li
              key={p.id}
              className="flex items-center justify-between px-4 py-3"
            >
              <div>
                <Link
                  to={`/projects/${p.slug}`}
                  className="font-medium hover:text-falseflag-500"
                >
                  {p.display_name}
                </Link>
                <div className="text-sm text-falseflag-900/60">
                  <code>{p.slug}</code>
                </div>
              </div>
              <StrategyBadge strategy={p.config_strategy} />
            </li>
          ))}
        </ul>
      )}
    </Page>
  );
}
