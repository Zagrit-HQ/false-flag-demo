import type { LoaderFunctionArgs, MetaFunction } from "@remix-run/node";
import { Link, useLoaderData } from "@remix-run/react";

import { type Flag, listFlags } from "@falseflag/generated-client";

import { EmptyState, ErrorBanner } from "~/components/ErrorBanner";
import { Page } from "~/components/Nav";
import { withApiFetch } from "~/lib/api.server";

export const meta: MetaFunction = () => [{ title: "Flags · FalseFlag" }];

interface LoaderData {
  slug: string;
  flags: Flag[];
  error?: string;
}

export async function loader({
  params,
}: LoaderFunctionArgs): Promise<LoaderData> {
  const slug = params.slug ?? "";
  return withApiFetch(async () => {
    try {
      const res = await listFlags(slug);
      const items = (res.data as { items?: Flag[] }).items ?? [];
      return { slug, flags: items };
    } catch (e) {
      return { slug, flags: [], error: (e as Error).message };
    }
  });
}

export default function FlagsList() {
  const { slug, flags, error } = useLoaderData<typeof loader>();
  return (
    <Page
      crumbs={[
        { to: "/projects", label: "Projects" },
        { to: `/projects/${slug}`, label: slug },
        { to: `/projects/${slug}/flags`, label: "Flags" },
      ]}
    >
      <h1 className="text-2xl font-bold">Flags in {slug}</h1>
      <ErrorBanner error={error} />
      {flags.length === 0 ? (
        <div className="mt-6">
          <EmptyState message="No flags in this project." />
        </div>
      ) : (
        <table
          className="mt-6 w-full overflow-hidden rounded-md border border-gray-200 bg-white text-sm"
          data-testid="flags-table"
        >
          <thead className="bg-gray-50 text-left text-xs uppercase text-falseflag-900/60">
            <tr>
              <th className="px-4 py-2">Key</th>
              <th className="px-4 py-2">Type</th>
              <th className="px-4 py-2">Default</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {flags.map((f) => (
              <tr key={f.id}>
                <td className="px-4 py-2 font-mono">
                  <Link
                    to={`/projects/${slug}/flags/${f.key}`}
                    className="hover:text-falseflag-500"
                  >
                    {f.key}
                  </Link>
                </td>
                <td className="px-4 py-2">{f.value_type}</td>
                <td className="px-4 py-2 text-falseflag-900/60">
                  <code>{JSON.stringify(f.default_value)}</code>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </Page>
  );
}
