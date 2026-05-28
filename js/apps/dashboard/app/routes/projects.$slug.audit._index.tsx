import type { LoaderFunctionArgs, MetaFunction } from "@remix-run/node";
import { Form, useLoaderData } from "@remix-run/react";

import { type AuditEvent, listAuditEvents } from "@falseflag/generated-client";

import { EmptyState, ErrorBanner } from "~/components/ErrorBanner";
import { Page } from "~/components/Nav";
import { withApiFetch } from "~/lib/api.server";

export const meta: MetaFunction = () => [{ title: "Audit · FalseFlag" }];

interface LoaderData {
  slug: string;
  events: AuditEvent[];
  filters: { action: string; actor: string };
  error?: string;
}

export async function loader({
  params,
  request,
}: LoaderFunctionArgs): Promise<LoaderData> {
  const slug = params.slug ?? "";
  const url = new URL(request.url);
  const action = url.searchParams.get("action") ?? "";
  const actor = url.searchParams.get("actor") ?? "";
  return withApiFetch(async () => {
    try {
      const res = await listAuditEvents(slug, {
        action: action || undefined,
        actor: actor || undefined,
        limit: 100,
      });
      const items = (res.data as { items?: AuditEvent[] }).items ?? [];
      return { slug, events: items, filters: { action, actor } };
    } catch (e) {
      return {
        slug,
        events: [],
        filters: { action, actor },
        error: (e as Error).message,
      };
    }
  });
}

export default function AuditList() {
  const { slug, events, filters, error } = useLoaderData<typeof loader>();
  return (
    <Page
      crumbs={[
        { to: "/projects", label: "Projects" },
        { to: `/projects/${slug}`, label: slug },
        { to: `/projects/${slug}/audit`, label: "Audit" },
      ]}
    >
      <h1 className="text-2xl font-bold">Audit log</h1>
      <ErrorBanner error={error} />

      <Form
        method="get"
        className="mt-4 flex flex-wrap items-end gap-3"
        data-testid="audit-filters"
      >
        <label className="block text-sm">
          <span className="text-xs uppercase text-falseflag-900/60">
            Action
          </span>
          <input
            name="action"
            defaultValue={filters.action}
            className="mt-1 block rounded-md border border-gray-300 px-2 py-1 text-sm"
            placeholder="publish_version"
          />
        </label>
        <label className="block text-sm">
          <span className="text-xs uppercase text-falseflag-900/60">Actor</span>
          <input
            name="actor"
            defaultValue={filters.actor}
            className="mt-1 block rounded-md border border-gray-300 px-2 py-1 text-sm"
            placeholder="cli/alice@example.com"
          />
        </label>
        <button
          type="submit"
          className="rounded-md bg-falseflag-500 px-3 py-1.5 text-sm text-white hover:bg-falseflag-900"
        >
          Filter
        </button>
      </Form>

      {events.length === 0 ? (
        <div className="mt-6">
          <EmptyState message="No audit events match the current filters." />
        </div>
      ) : (
        <table
          className="mt-6 w-full overflow-hidden rounded-md border border-gray-200 bg-white text-sm"
          data-testid="audit-table"
        >
          <thead className="bg-gray-50 text-left text-xs uppercase text-falseflag-900/60">
            <tr>
              <th className="px-4 py-2">When</th>
              <th className="px-4 py-2">Action</th>
              <th className="px-4 py-2">Actor</th>
              <th className="px-4 py-2">Flag</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {events.map((e) => (
              <tr key={e.id}>
                <td className="px-4 py-2 font-mono text-xs">
                  {e.created_at?.slice(0, 19) ?? "?"}
                </td>
                <td className="px-4 py-2">{e.action}</td>
                <td className="px-4 py-2 font-mono text-xs">
                  {e.actor ?? "—"}
                </td>
                <td className="px-4 py-2 font-mono text-xs">
                  {e.flag_id ? `${e.flag_id.slice(0, 8)}…` : "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </Page>
  );
}
