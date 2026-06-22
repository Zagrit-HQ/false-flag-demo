import { vitePlugin as remix } from "@remix-run/dev";
import { defineConfig } from "vite";
import tsconfigPaths from "vite-tsconfig-paths";

export default defineConfig({
  plugins: [
    remix({
      future: {
        v3_fetcherPersist: true,
        v3_relativeSplatPath: true,
        v3_throwAbortReason: true,
        v3_lazyRouteDiscovery: true,
        v3_singleFetch: true,
      },
    }),
    tsconfigPaths(),
  ],
  // Monaco ships a large transitive dep tree; pre-bundling it keeps
  // the dev-mode cold-start sane. The .client.tsx + React.lazy pair
  // keep production builds out of the SSR bundle and out of the
  // view-route client chunk — the editor lands only on the edit route.
  optimizeDeps: {
    include: ["monaco-editor", "@monaco-editor/react"],
  },
});
