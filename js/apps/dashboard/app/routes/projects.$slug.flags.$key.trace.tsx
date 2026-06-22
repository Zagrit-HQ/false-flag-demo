import type {
  ActionFunctionArgs,
  LoaderFunctionArgs,
  MetaFunction,
} from "@remix-run/node";
import { Form, useActionData, useLoaderData } from "@remix-run/react";

import {
  type EvaluateTraceResponse,
  evaluateFlagWithTrace,
} from "@falseflag/generated-client";

import { ErrorBanner } from "~/components/ErrorBanner";
import { Page } from "~/components/Nav";
import { type TraceRoot, TraceTree } from "~/components/TraceTree";
import { withApiFetch } from "~/lib/api.server";

export const meta: MetaFunction = () => [
  { title: "Evaluation trace · FalseFlag" },
];

interface LoaderData {
  slug: string;
  key: string;
}

export async function loader({
  params,
}: LoaderFunctionArgs): Promise<LoaderData> {
  return { slug: params.slug ?? "", key: params.key ?? "" };
}

interface ActionData {
  decision?: EvaluateTraceResponse["decision"];
  trace?: TraceRoot;
  context?: string;
  error?: string;
}

export async function action({
  params,
  request,
}: ActionFunctionArgs): Promise<ActionData> {
  const slug = params.slug ?? "";
  const key = params.key ?? "";
  const form = await request.formData();
  const raw = String(form.get("context") ?? "{}");
  let ctx: Record<string, unknown> = {};
  try {
    ctx = JSON.parse(raw);
  } catch (e) {
    return {
      context: raw,
      error: `invalid JSON context: ${(e as Error).message}`,
    };
  }
  return withApiFetch(async () => {
    try {
      const res = await evaluateFlagWithTrace(slug, key, { context: ctx });
      if (res.status !== 200) {
        return { context: raw, error: `HTTP ${res.status}` };
      }
      const body = res.data as EvaluateTraceResponse;
      return {
        decision: body.decision,
        trace: body.trace as unknown as TraceRoot,
        context: raw,
      };
    } catch (e) {
      return { context: raw, error: (e as Error).message };
    }
  });
}

export default function TracePage() {
  const { slug, key } = useLoaderData<typeof loader>();
  const result = useActionData<typeof action>();
  const initial =
    result?.context ??
    JSON.stringify({ user: { id: "u-demo", plan: "pro" } }, null, 2);

  return (
    <Page
      crumbs={[
        { to: "/projects", label: "Projects" },
        { to: `/projects/${slug}`, label: slug },
        { to: `/projects/${slug}/flags`, label: "Flags" },
        { to: `/projects/${slug}/flags/${key}`, label: key },
        { to: `/projects/${slug}/flags/${key}/trace`, label: "Trace" },
      ]}
    >
      <h1 className="text-2xl font-bold">Evaluate {key}</h1>
      <p className="mt-1 text-sm text-falseflag-900/70">
        Submit an evaluation context to see how each rule fires.
      </p>
      <ErrorBanner error={result?.error} />

      <Form method="post" className="mt-6 space-y-4" data-testid="eval-form">
        <label className="block">
          <span className="text-xs uppercase text-falseflag-900/60">
            Context (JSON)
          </span>
          <textarea
            name="context"
            rows={6}
            defaultValue={initial}
            className="mt-1 w-full rounded-md border border-gray-300 bg-white p-3 font-mono text-xs"
            data-testid="context-input"
          />
        </label>
        <button
          type="submit"
          className="inline-flex items-center rounded-md bg-falseflag-500 px-4 py-2 text-sm font-medium text-white hover:bg-falseflag-900"
          data-testid="evaluate-btn"
        >
          Evaluate
        </button>
      </Form>

      {result?.decision && (
        <section
          className="mt-8 rounded-md border border-gray-200 bg-white p-4"
          data-testid="decision"
        >
          <h2 className="text-lg font-semibold">Decision</h2>
          <dl className="mt-2 grid grid-cols-2 gap-2 text-sm">
            <dt className="text-falseflag-900/60">Value</dt>
            <dd className="font-mono">
              {JSON.stringify(result.decision.value)}
            </dd>
            <dt className="text-falseflag-900/60">Reason</dt>
            <dd>{result.decision.reason}</dd>
            {result.decision.rule_id && (
              <>
                <dt className="text-falseflag-900/60">Rule</dt>
                <dd className="font-mono">{result.decision.rule_id}</dd>
              </>
            )}
          </dl>
        </section>
      )}

      {result?.trace && (
        <section className="mt-6" data-testid="trace">
          <h2 className="mb-2 text-lg font-semibold">Trace</h2>
          <TraceTree trace={result.trace} />
        </section>
      )}
    </Page>
  );
}
