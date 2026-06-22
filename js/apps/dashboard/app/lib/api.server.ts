// api.server.ts is a server-only helper that prefixes the Orval
// generated client's relative URLs with the configured API base URL.
//
// The Orval client uses bare fetch() against relative paths like
// "/v1/projects". Remix loaders need to call the API server-side
// during SSR. This helper installs a wrapping fetch on globalThis
// for the duration of one async block, then restores the original.
//
// Usage:
//
//   import { withApiFetch } from "~/lib/api.server";
//
//   export async function loader() {
//     return withApiFetch(async () => {
//       const res = await listProjects();
//       return { projects: res.data.items };
//     });
//   }

export interface ApiFetchOptions {
  /** Override the API base URL. Defaults to FALSEFLAG_API_BASE_URL or localhost. */
  baseUrl?: string;
}

export async function withApiFetch<T>(
  body: () => Promise<T>,
  opts: ApiFetchOptions = {},
): Promise<T> {
  const baseUrl =
    opts.baseUrl ??
    process.env.FALSEFLAG_API_BASE_URL ??
    "http://localhost:8080";
  const prior = globalThis.fetch;
  const apiFetch: typeof fetch = (input, init) => {
    if (typeof input === "string" && input.startsWith("/")) {
      return prior(`${baseUrl}${input}`, init);
    }
    return prior(input, init);
  };
  globalThis.fetch = apiFetch;
  try {
    return await body();
  } finally {
    globalThis.fetch = prior;
  }
}

/**
 * Returns the configured API base URL. Used by route components that
 * need to render an external link to the API (e.g., "View raw JSON").
 */
export function apiBaseUrl(): string {
  return process.env.FALSEFLAG_API_BASE_URL ?? "http://localhost:8080";
}
