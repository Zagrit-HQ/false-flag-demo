import { defineConfig } from "orval";

// Two outputs from the same OpenAPI spec:
// - falseflag: typed fetch client → src/generated/api.ts
// - falseflagZod: Zod request/response schemas → src/generated/api.zod.ts
// Each output uses mode: "single" so the existing import surface stays
// stable on regeneration.
export default defineConfig({
  // Each generator targets a single, named output file with
  // clean: false. The clean flag clears the entire output directory,
  // which is dangerous when two generators write to the same folder —
  // running the second one would delete the first one's output. We
  // rely on Orval overwriting individual target files instead.
  falseflag: {
    input: "../../../api/openapi/openapi.yaml",
    output: {
      target: "./src/generated/api.ts",
      client: "fetch",
      mode: "single",
      clean: false,
    },
  },
  falseflagZod: {
    input: "../../../api/openapi/openapi.yaml",
    output: {
      target: "./src/generated/api.zod.ts",
      client: "zod",
      mode: "single",
      clean: false,
      fileExtension: ".zod.ts",
    },
  },
});
