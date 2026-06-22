import type {
  ActionFunctionArgs,
  LoaderFunctionArgs,
  MetaFunction,
} from "@remix-run/node";
import { redirect } from "@remix-run/node";
import { Form, Link, useLoaderData } from "@remix-run/react";

import {
  type Flag,
  type FlagVersion,
  compileSnapshot,
  getFlag,
  listFlagVersions,
} from "@falseflag/generated-client";

import { CodeBlock } from "~/components/CodeBlock";
import { ErrorBanner } from "~/components/ErrorBanner";
import { Page } from "~/components/Nav";
import { StrategyBadge } from "~/components/StrategyBadge";
import { withApiFetch } from "~/lib/api.server";
import { highlightSource } from "~/lib/highlighter.server";

export const meta: MetaFunction<typeof loader> = ({ data }) => [
  {
    title: data?.flag ? `${data.flag.key} · FalseFlag` : "Flag · FalseFlag",
  },
];

interface LoaderData {
  slug: string;
  key: string;
  flag?: Flag;
  latest?: FlagVersion;
  versions: FlagVersion[];
  /** Shiki-rendered HTML; populated when latest has source_text. */
  sourceHtml?: string;
  /** Pretty-printed IR; populated when source_text is absent. */
  fallbackJson?: string;
  /** Caption shown under the code block (e.g. provenance notes). */
  caption?: string;
  /** When set, render a toast linking to "Publish snapshot". The
   *  value is the version tag from the edit redirect, e.g. "v3". */
  publishedVersion?: string;
  error?: string;
}

// action — handles the "Publish snapshot" CTA inside the toast. POSTs
// compileSnapshot for this project, then sends the user to the
// snapshots index where the freshly compiled row sits at the top.
export async function action({
  params,
  request,
}: ActionFunctionArgs): Promise<Response> {
  const slug = params.slug ?? "";
  const form = await request.formData();
  if (form.get("intent") !== "publish-snapshot") {
    return new Response("unknown intent", { status: 400 });
  }
  return withApiFetch(async () => {
    await compileSnapshot(slug, {});
    return redirect(`/projects/${slug}/snapshots`);
  });
}

export async function loader({
  params,
  request,
}: LoaderFunctionArgs): Promise<LoaderData> {
  const slug = params.slug ?? "";
  const key = params.key ?? "";
  const publishedVersion =
    new URL(request.url).searchParams.get("published") ?? undefined;
  return withApiFetch(async () => {
    try {
      const [flagRes, versionsRes] = await Promise.all([
        getFlag(slug, key),
        listFlagVersions(slug, key),
      ]);
      const flagBody = flagRes.data as {
        flag?: Flag;
        latest_version?: FlagVersion;
      };
      const flag = flagBody.flag ?? (flagRes.data as unknown as Flag);
      const latest = flagBody.latest_version;
      const versions =
        (versionsRes.data as { items?: FlagVersion[] }).items ?? [];

      let sourceHtml: string | undefined;
      let fallbackJson: string | undefined;
      let caption: string | undefined;
      if (latest) {
        const sourceText = latest.source_text ?? undefined;
        if (sourceText) {
          try {
            sourceHtml = await highlightSource(sourceText, latest.strategy);
          } catch {
            // Shiki failure should never block the page render — fall
            // through to the IR pretty-print path with a caption note.
            fallbackJson = JSON.stringify(latest.compiled, null, 2);
            caption = "syntax highlighter unavailable — showing compiled IR";
          }
        } else {
          fallbackJson = JSON.stringify(latest.compiled, null, 2);
          caption = "compiled IR — original source not stored";
        }
      }

      return {
        slug,
        key,
        flag,
        latest,
        versions,
        sourceHtml,
        fallbackJson,
        caption,
        publishedVersion,
      };
    } catch (e) {
      return { slug, key, versions: [], error: (e as Error).message };
    }
  });
}

export default function FlagDetail() {
  const {
    slug,
    key,
    flag,
    latest,
    versions,
    sourceHtml,
    fallbackJson,
    caption,
    publishedVersion,
    error,
  } = useLoaderData<typeof loader>();
  return (
    <Page
      crumbs={[
        { to: "/projects", label: "Projects" },
        { to: `/projects/${slug}`, label: slug },
        { to: `/projects/${slug}/flags`, label: "Flags" },
        { to: `/projects/${slug}/flags/${key}`, label: key },
      ]}
    >
      <ErrorBanner error={error} />
      <div className="flex items-baseline justify-between">
        <h1 className="font-mono text-2xl font-bold">{key}</h1>
        <div className="flex items-center gap-3">
          {latest && <StrategyBadge strategy={latest.strategy} />}
          {latest && (
            <Link
              to={`/projects/${slug}/flags/${key}/edit`}
              className="rounded-md border border-falseflag-500 px-3 py-1 text-sm text-falseflag-500 hover:bg-falseflag-500 hover:text-white"
              data-testid="edit-cta"
            >
              Edit
            </Link>
          )}
          <Link
            to={`/projects/${slug}/flags/${key}/trace`}
            className="rounded-md border border-falseflag-500 px-3 py-1 text-sm text-falseflag-500 hover:bg-falseflag-500 hover:text-white"
            data-testid="trace-cta"
          >
            Evaluate / trace →
          </Link>
        </div>
      </div>
      {flag && (
        <p className="mt-1 text-sm text-falseflag-900/70">
          {flag.name} · type <code>{flag.value_type}</code>
        </p>
      )}

      {publishedVersion && (
        <div
          className="mt-4 flex items-center justify-between rounded-md border border-green-300 bg-green-50 px-4 py-3 text-sm text-green-900"
          data-testid="publish-toast"
        >
          <span>
            <strong>{publishedVersion} published.</strong> Compile a snapshot to
            propagate this edit to SDKs and the proxy.
          </span>
          <Form method="post">
            <input type="hidden" name="intent" value="publish-snapshot" />
            <button
              type="submit"
              className="ml-4 rounded-md border border-green-600 bg-green-600 px-3 py-1 text-sm font-semibold text-white hover:bg-green-700"
              data-testid="publish-snapshot-cta"
            >
              Publish snapshot
            </button>
          </Form>
        </div>
      )}

      <section className="mt-6" data-testid="source-code">
        <h2 className="mb-2 text-lg font-semibold">Latest version</h2>
        {latest ? (
          <CodeBlock
            html={sourceHtml}
            fallbackJson={fallbackJson}
            caption={caption}
          />
        ) : (
          <div className="text-sm text-falseflag-900/60">
            no version published yet
          </div>
        )}
      </section>

      <section className="mt-8">
        <h2 className="mb-2 text-lg font-semibold">Version history</h2>
        <ul
          className="divide-y divide-gray-200 rounded-md border border-gray-200 bg-white text-sm"
          data-testid="versions"
        >
          {versions.length === 0 ? (
            <li className="px-4 py-3 text-falseflag-900/60">none</li>
          ) : (
            versions.map((v) => (
              <li
                key={v.id}
                className="flex items-center justify-between px-4 py-2"
              >
                <span>v{v.version}</span>
                <span className="text-xs text-falseflag-900/60">
                  {v.strategy} · {v.published_at?.slice(0, 19) ?? "?"}
                </span>
              </li>
            ))
          )}
        </ul>
      </section>
    </Page>
  );
}
