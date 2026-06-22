import type { LoaderFunctionArgs, MetaFunction } from "@remix-run/node";
import { Link, useLoaderData } from "@remix-run/react";

import {
  type Environment,
  type Flag,
  type Project,
  type Snapshot,
  getLatestSnapshot,
  getProject,
  listEnvironments,
  listFlags,
} from "@falseflag/generated-client";

import { EmptyState, ErrorBanner } from "~/components/ErrorBanner";
import { Page } from "~/components/Nav";
import { StrategyBadge } from "~/components/StrategyBadge";
import { withApiFetch } from "~/lib/api.server";

export const meta: MetaFunction<typeof loader> = ({ data }) => [
  {
    title: data?.project
      ? `${data.project.display_name} · FalseFlag`
      : "Project · FalseFlag",
  },
];

interface LoaderData {
  slug: string;
  project?: Project;
  flags: Flag[];
  environments: Environment[];
  latestSnapshot?: Snapshot;
  error?: string;
}

export async function loader({
  params,
}: LoaderFunctionArgs): Promise<LoaderData> {
  const slug = params.slug ?? "";
  return withApiFetch(async () => {
    try {
      const projectRes = await getProject(slug);
      if (projectRes.status !== 200) {
        return {
          slug,
          flags: [],
          environments: [],
          error: `project HTTP ${projectRes.status}`,
        };
      }
      const [flagsRes, envsRes, snapRes] = await Promise.all([
        listFlags(slug),
        listEnvironments(slug),
        getLatestSnapshot(slug),
      ]);
      const project =
        (projectRes.data as { project?: Project }).project ??
        (projectRes.data as Project);
      const flags = (flagsRes.data as { items?: Flag[] }).items ?? [];
      const environments =
        (envsRes.data as { items?: Environment[] }).items ?? [];
      const latestSnapshot =
        snapRes.status === 200 ? (snapRes.data as Snapshot) : undefined;
      return { slug, project, flags, environments, latestSnapshot };
    } catch (e) {
      return { slug, flags: [], environments: [], error: (e as Error).message };
    }
  });
}

export default function ProjectDetail() {
  const { slug, project, flags, environments, latestSnapshot, error } =
    useLoaderData<typeof loader>();
  return (
    <Page
      crumbs={[
        { to: "/projects", label: "Projects" },
        { to: `/projects/${slug}`, label: project?.display_name ?? slug },
      ]}
    >
      <ErrorBanner error={error} />
      <h1 className="text-2xl font-bold">
        {project?.display_name ?? slug}{" "}
        {project && <StrategyBadge strategy={project.config_strategy} />}
      </h1>
      <p className="mt-1 text-sm text-falseflag-900/70">
        <code>{slug}</code> · created {project?.created_at?.slice(0, 10) ?? "?"}
      </p>

      <section
        className="mt-6 grid grid-cols-1 gap-4 md:grid-cols-3"
        data-testid="project-stats"
      >
        <Stat label="Flags" value={String(flags.length)} />
        <Stat label="Environments" value={String(environments.length)} />
        <Stat
          label="Latest snapshot"
          value={latestSnapshot ? `v${latestSnapshot.version}` : "—"}
        />
      </section>

      <section className="mt-8">
        <div className="flex items-baseline justify-between">
          <h2 className="text-lg font-semibold">Flags</h2>
          <Link
            to={`/projects/${slug}/flags`}
            className="text-sm text-falseflag-500 hover:underline"
          >
            view all →
          </Link>
        </div>
        {flags.length === 0 ? (
          <div className="mt-3">
            <EmptyState message="No flags published yet." />
          </div>
        ) : (
          <ul
            className="mt-3 divide-y divide-gray-200 rounded-md border border-gray-200 bg-white"
            data-testid="flag-preview"
          >
            {flags.slice(0, 5).map((f) => (
              <li
                key={f.id}
                className="flex items-center justify-between px-4 py-2"
              >
                <Link
                  to={`/projects/${slug}/flags/${f.key}`}
                  className="font-mono text-sm hover:text-falseflag-500"
                >
                  {f.key}
                </Link>
                <span className="text-xs text-falseflag-900/60">
                  {f.value_type}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="mt-8 grid grid-cols-1 gap-4 md:grid-cols-3">
        <Link
          to={`/projects/${slug}/snapshots`}
          className="rounded-md border border-gray-200 bg-white p-4 hover:border-falseflag-500"
        >
          <div className="font-medium">Snapshots</div>
          <div className="text-xs text-falseflag-900/60">
            history of compiled releases
          </div>
        </Link>
        <Link
          to={`/projects/${slug}/audit`}
          className="rounded-md border border-gray-200 bg-white p-4 hover:border-falseflag-500"
        >
          <div className="font-medium">Audit log</div>
          <div className="text-xs text-falseflag-900/60">
            who changed what, when
          </div>
        </Link>
        <Link
          to={`/projects/${slug}/flags`}
          className="rounded-md border border-gray-200 bg-white p-4 hover:border-falseflag-500"
        >
          <div className="font-medium">All flags</div>
          <div className="text-xs text-falseflag-900/60">
            strategy, type, version
          </div>
        </Link>
      </section>
    </Page>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-gray-200 bg-white p-4">
      <div className="text-xs uppercase tracking-wide text-falseflag-900/60">
        {label}
      </div>
      <div className="mt-1 text-2xl font-bold">{value}</div>
    </div>
  );
}
