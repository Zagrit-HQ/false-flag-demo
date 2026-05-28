import type { LoaderFunctionArgs, MetaFunction } from "@remix-run/node";
import { useLoaderData } from "@remix-run/react";

import { type Snapshot, listSnapshots } from "@falseflag/generated-client";

import { EmptyState, ErrorBanner } from "~/components/ErrorBanner";
import { Page } from "~/components/Nav";
import { withApiFetch } from "~/lib/api.server";

export const meta: MetaFunction = () => [{ title: "Snapshots · FalseFlag" }];

interface LoaderData {
  slug: string;
  snapshots: Snapshot[];
  error?: string;
}

export async function loader({
  params,
}: LoaderFunctionArgs): Promise<LoaderData> {
  const slug = params.slug ?? "";
  return withApiFetch(async () => {
    try {
      const res = await listSnapshots(slug, { limit: 50 });
      const items = (res.data as { items?: Snapshot[] }).items ?? [];
      return { slug, snapshots: items };
    } catch (e) {
      return { slug, snapshots: [], error: (e as Error).message };
    }
  });
}

export default function SnapshotsList() {
  const { slug, snapshots, error } = useLoaderData<typeof loader>();
  return (
    <Page
      crumbs={[
        { to: "/projects", label: "Projects" },
        { to: `/projects/${slug}`, label: slug },
        { to: `/projects/${slug}/snapshots`, label: "Snapshots" },
      ]}
    >
      <h1 className="text-2xl font-bold">Snapshots</h1>
      <p className="mt-1 text-sm text-falseflag-900/70">
        Each snapshot is an immutable, project-wide compiled bundle. SDKs and
        the proxy poll the latest one every ~10s.
      </p>
      <ErrorBanner error={error} />
      {snapshots.length === 0 ? (
        <div className="mt-6">
          <EmptyState message="No snapshots compiled yet." />
        </div>
      ) : (
        <table
          className="mt-6 w-full overflow-hidden rounded-md border border-gray-200 bg-white text-sm"
          data-testid="snapshots-table"
        >
          <thead className="bg-gray-50 text-left text-xs uppercase text-falseflag-900/60">
            <tr>
              <th className="px-4 py-2">Version</th>
              <th className="px-4 py-2">ID</th>
              <th className="px-4 py-2">Created</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {snapshots.map((s) => (
              <tr key={s.id}>
                <td className="px-4 py-2 font-bold">v{s.version}</td>
                <td className="px-4 py-2 font-mono text-xs text-falseflag-900/60">
                  {s.id.slice(0, 8)}…
                </td>
                <td className="px-4 py-2 text-falseflag-900/60">
                  {s.created_at?.slice(0, 19) ?? "?"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </Page>
  );
}
