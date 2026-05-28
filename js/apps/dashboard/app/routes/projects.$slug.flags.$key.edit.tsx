import type {
  ActionFunctionArgs,
  LoaderFunctionArgs,
  MetaFunction,
} from "@remix-run/node";
import { redirect } from "@remix-run/node";
import {
  Form,
  Link,
  useActionData,
  useLoaderData,
  useNavigation,
} from "@remix-run/react";
import { Suspense, lazy, useState } from "react";

import {
  type Flag,
  type FlagVersion,
  type Strategy,
  getFlag,
  listFlagVersions,
  publishFlagVersion,
} from "@falseflag/generated-client";

import { EditorSkeleton } from "~/components/EditorSkeleton";
import { ErrorBanner } from "~/components/ErrorBanner";
import { Page } from "~/components/Nav";
import { StrategyBadge } from "~/components/StrategyBadge";
import type { CompileErrorDetail } from "~/components/editor.client";
import { withApiFetch } from "~/lib/api.server";

// Monaco lives in its own Vite chunk via React.lazy + the .client.tsx
// browser-only suffix. The view route bundle stays Monaco-free.
const CodeEditor = lazy(() => import("~/components/editor.client"));

export const meta: MetaFunction<typeof loader> = ({ data }) => [
  {
    title: data?.flag
      ? `Edit ${data.flag.key} · FalseFlag`
      : "Edit flag · FalseFlag",
  },
];

interface LoaderData {
  slug: string;
  key: string;
  flag?: Flag;
  latest?: FlagVersion;
  initialSource: string;
  strategy: Strategy;
  error?: string;
}

const langForStrategy: Record<Strategy, "typescript" | "javascript" | "json"> =
  {
    typescript: "typescript",
    cel: "javascript",
    json: "json",
  };

export async function loader({
  params,
}: LoaderFunctionArgs): Promise<LoaderData> {
  const slug = params.slug ?? "";
  const key = params.key ?? "";
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
      const head = latest ?? versions[0];
      const strategy = (head?.strategy ?? "json") as Strategy;
      const initialSource = head?.source_text
        ? head.source_text
        : JSON.stringify(head?.compiled ?? {}, null, 2);
      return { slug, key, flag, latest: head, initialSource, strategy };
    } catch (e) {
      return {
        slug,
        key,
        initialSource: "",
        strategy: "json" as Strategy,
        error: (e as Error).message,
      };
    }
  });
}

interface ActionData {
  errors?: CompileErrorDetail[];
  message?: string;
  source?: string;
}

export async function action({
  params,
  request,
}: ActionFunctionArgs): Promise<ActionData | Response> {
  const slug = params.slug ?? "";
  const key = params.key ?? "";
  const form = await request.formData();
  const strategy = String(form.get("strategy") ?? "json") as Strategy;
  const sourceText = String(form.get("source_text") ?? "");

  return withApiFetch(async () => {
    try {
      const res = await publishFlagVersion(slug, key, {
        strategy,
        // JSON and CEL both author their source as IR JSON in the
        // Monaco editor; parse it so the API gets the real shape.
        // TypeScript ships its source via source_text; the server
        // compiles it and overwrites the IR, so source can stay {}.
        source:
          strategy === "json" || strategy === "cel"
            ? safeParse(sourceText)
            : {},
        source_text: sourceText,
      });
      if (res.status === 200 || res.status === 201) {
        // Land on the view route with ?published=v{N}; the view route
        // renders a toast linking to "Publish snapshot" so the audience
        // sees the edit → snapshot → SDK pipeline explicitly.
        const published = (res.data as { version?: number })?.version;
        const suffix = published ? `?published=v${published}` : "";
        return redirect(`/projects/${slug}/flags/${key}${suffix}`);
      }
      // 422 path — surface structured compile details if present.
      const body = res.data as {
        message?: string;
        details?: CompileErrorDetail[];
      };
      return {
        message: body.message ?? `HTTP ${res.status}`,
        errors: body.details ?? [],
        source: sourceText,
      };
    } catch (e) {
      return { message: (e as Error).message, source: sourceText };
    }
  });
}

function safeParse(s: string): unknown {
  try {
    return JSON.parse(s);
  } catch {
    return {};
  }
}

export default function EditFlag() {
  const { slug, key, flag, latest, initialSource, strategy, error } =
    useLoaderData<typeof loader>();
  const actionData = useActionData<typeof action>();
  const nav = useNavigation();
  const submitting = nav.state === "submitting";
  const [source, setSource] = useState(actionData?.source ?? initialSource);

  return (
    <Page
      crumbs={[
        { to: "/projects", label: "Projects" },
        { to: `/projects/${slug}`, label: slug },
        { to: `/projects/${slug}/flags`, label: "Flags" },
        { to: `/projects/${slug}/flags/${key}`, label: key },
        { to: `/projects/${slug}/flags/${key}/edit`, label: "Edit" },
      ]}
    >
      <ErrorBanner error={error} />
      <div className="flex items-baseline justify-between">
        <h1 className="font-mono text-2xl font-bold">{key}</h1>
        <div className="flex items-center gap-3">
          <StrategyBadge strategy={strategy} />
        </div>
      </div>
      {flag && (
        <p className="mt-1 text-sm text-falseflag-900/70">
          Editing {flag.name} · type <code>{flag.value_type}</code>
          {latest && (
            <span className="ml-2 text-falseflag-900/50">
              (last published v{latest.version})
            </span>
          )}
        </p>
      )}

      <Form method="post" className="mt-6 space-y-3">
        <input type="hidden" name="strategy" value={strategy} />
        {actionData?.message && (
          <div
            className="rounded-md border border-red-300 bg-red-50 px-4 py-2 text-sm text-red-800"
            data-testid="compile-error-banner"
          >
            <strong>compile failed:</strong> {actionData.message}
            {actionData.errors && actionData.errors.length > 0 && (
              <ul className="mt-1 list-inside list-disc text-xs">
                {actionData.errors.map((e: CompileErrorDetail, i: number) => (
                  <li key={`${e.line}-${i}`}>
                    line {e.line}
                    {e.column ? `:${e.column}` : ""} — {e.text}
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
        <Suspense fallback={<EditorSkeleton />}>
          <CodeEditor
            value={source}
            language={langForStrategy[strategy]}
            onChange={setSource}
            errors={actionData?.errors}
          />
        </Suspense>
        {/* Hidden mirror of the editor's value, so the Remix Form
            picks it up as form data on submit. */}
        <textarea
          name="source_text"
          value={source}
          readOnly
          className="hidden"
        />
        <div className="flex items-center justify-end gap-3">
          <Link
            to={`/projects/${slug}/flags/${key}`}
            className="rounded-md border border-gray-300 px-3 py-1 text-sm text-gray-600 hover:bg-gray-50"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-falseflag-500 px-4 py-1 text-sm text-white hover:bg-falseflag-600 disabled:opacity-50"
            data-testid="save-cta"
          >
            {submitting ? "Saving…" : "Save"}
          </button>
        </div>
      </Form>
    </Page>
  );
}
