---
date: 2026-05-20
topic: moonconfig-historical-reference
source: /Users/wito/code/project-keat/keat-release
---

# MoonConfig Historical Reference

## Purpose

This note preserves the abandoned MoonConfig idea from `/Users/wito/code/project-keat/keat-release` as inspiration for a future TypeScript config-as-code strategy in the synthetic PlatformCon feature flag project.

The target direction is not to copy the old implementation directly. The old code is useful as a conceptual seed: developer-authored TypeScript release logic evaluated in a sandbox, compiled into static feature rules, and published to database plus edge storage.

## Historical Reference

Repository:

```text
/Users/wito/code/project-keat/keat-release
```

Added in:

```text
07b6918 2023-04-05 feat: latest wip
```

Removed around:

```text
a25ec26 2023-07-25 chore: web app redesign
```

Important files to inspect:

```bash
git -C /Users/wito/code/project-keat/keat-release show 07b6918:apps/api/src/utils/moonConfig.ts
git -C /Users/wito/code/project-keat/keat-release show 07b6918:apps/api/src/utils/esbuild.ts
git -C /Users/wito/code/project-keat/keat-release show 07b6918:apps/api/src/schema/MutationAppConfigure.ts
git -C /Users/wito/code/project-keat/keat-release show 07b6918:apps/api/src/schema/MutationAppRelease.ts
git -C /Users/wito/code/project-keat/keat-release show 07b6918:apps/cli/keat.release.3.ts
```

## Original Shape

The TypeScript config model looked like this:

```ts
Keat.environment("default", (t) => ({
  features: {
    checkout: t.stage("canary", { rollout: 5 }),
  },
}));

Keat.stage("canary", (t) => ({
  args: { rollout: t.arg.number() },
  rule: ({ rollout }) => ({ OR: ["preview", rollout] }),
}));

export default Keat.release();
```

The historical `keat.release.3.ts` sample used the same idea with a `ga` stage and multiple environments.

## How The WIP Worked

`bundleConfig(code)` used esbuild to bundle user-submitted config. It provided a virtual `keat-release` module that exported the DSL builder, so config authors could write against a controlled public API.

`createMoonConfig(code)` used `quickjs-emscripten`:

- Create a QuickJS context.
- Register a module loader that returned the bundled config code.
- Evaluate an import that assigned the default export to `globalThis.config`.
- Read config values with `moonConfig.get(path)`.
- Call config functions with `moonConfig.run(path, ...args)`.

`MutationAppConfigure` bundled the submitted source, evaluated it, extracted the config, stored source and bundle in `keatConfig`, upserted feature rules into the database, and published environment feature data to Cloudflare KV.

`MutationAppRelease` loaded stored config, modified source with a morphing utility, rebundled and reevaluated it, then updated feature rules and KV.

Conceptually:

```text
TypeScript release source
  -> esbuild bundle with allowed DSL imports
  -> QuickJS/WASM sandbox evaluation
  -> validated config object
  -> normalized static release state
  -> DB + edge storage
  -> SDK/proxy consumes only JSON/rules
```

## Planned Feature Direction

Reintroduce this as one project-scoped configuration strategy: **TypeScript config-as-code**.

A project should choose one active configuration strategy at a time:

- `json`: static JSON config through database, Redis, or Kubernetes CRDs.
- `cel`: CEL-authored targeting expressions compiled into the rule model.
- `typescript`: TypeScript config-as-code evaluated in a sandbox and compiled to static release state.

Only one needs to work for a project at a time. That keeps the product model simple while still creating real build, validation, and test complexity.

## Proposed Architecture

Config authoring:

- Add a DSL package such as `@keat/config`.
- Keep the DSL pure and deterministic.
- Model environments, stages, typed stage args, features, and rules.
- Avoid exposing side effects in the public API.

Bundling:

- Use esbuild to bundle config source.
- Allow only `@keat/config` imports for the MVP.
- Reject or externalize all other imports.
- Store source and bundled artifact for reproducibility.

Sandbox evaluation:

- Use QuickJS/WASM or a maintained equivalent.
- No Node globals, filesystem, network, process, timers, or dynamic imports by default.
- Add strict execution timeout and memory limits.
- Dispose runtime handles aggressively.

Validation:

- Validate evaluated output with Zod or an equivalent schema package.
- Config output must be pure data: environments, stages, features, rules, and typed args.
- Return diagnostics that are useful in the CLI, dashboard editor, and API.

Compilation:

- Compile evaluated config into static release JSON.
- Store normalized feature rules in the database.
- Publish edge-safe JSON to Redis/KV/CDN/object storage.
- Runtime SDKs and the evaluation proxy only read compiled config.

Release mutation:

- Defer automatic source mutation in the MVP.
- Prefer updating structured release state first.
- If source mutation is reintroduced, use AST/codemod tooling rather than fragile string edits.

## MVP Scope

Keep the first version intentionally tight:

- One config file.
- One allowed import: `@keat/config`.
- No external dependencies.
- Evaluate default export only.
- Support environment, stage, feature, typed args, and simple rule output.
- Compile to static JSON.
- Add API endpoint: validate config.
- Add API endpoint: save config.
- Add CLI command: `keat config validate`.

## Risks

- Sandbox escape and dependency imports.
- Infinite loops and memory exhaustion.
- Non-determinism through dates, randomness, timers, or hidden globals.
- Poor diagnostics when bundling or sandbox evaluation fails.
- Source mutation complexity.
- Type safety across editor, CLI, API, and stored config versions.
- DSL versioning and migration of stored configs.

## Historical WIP Caveats

Do not copy the old MoonConfig implementation as production code. It lacked:

- timeout controls
- memory controls
- robust handle disposal
- good error mapping
- strict import restrictions beyond esbuild conventions
- full argument marshaling
- deterministic execution policy

Use it as a prototype reference and narrative seed, not as the implementation baseline.
