# FalseFlag — Conference demo script

8–10 minute pacing for the PlatformCon talk about accelerating slow
CI/CD with Depot. The story arc:

1. *Here is a believably-sized platform.*
2. *Here's what it does when it works.*
3. *Here is the build/test pipeline that backs it.*
4. *Here is where Depot makes that pipeline 10× faster.*

The repo is intentionally large (Go + TypeScript monorepo, three
strategies, Kubernetes operator, MCP server) so the CI parallelism
story has real fan-out to talk against.

---

## Pre-flight (do off-stage)

```bash
docker compose down -v
docker compose up -d --build
make seed
```

Open browser tabs in this order:

1. `http://localhost:3030/projects` (dashboard projects index)
2. A terminal at the repo root
3. The Depot dashboard / CI dashboard (slice 7b territory)

## 0:00 – 1:00 · Context

> "This is a fake feature flag platform called FalseFlag. It does
> not exist as a product. It exists because building a *plausible*
> feature flag platform — Go API, TypeScript SDK, Kubernetes
> operator, MCP server, three configuration DSLs — generates the
> kind of CI workload my real customers have. And that's what I
> actually want to talk about today: how to make that workload
> stop being a 30-minute wait on every PR."

Show the architecture diagram (`docs/architecture/diagram.mmd`
rendered to SVG). Trace the boxes in a single sentence each.

## 1:00 – 3:00 · `docker compose up` + dashboard tour

```bash
docker compose ps
```

> "Six services. One Postgres, one API, one MCP server, one Go
> proxy, one Kubernetes operator, one Remix dashboard. They all
> build in this repo."

Open `http://localhost:3030/projects`. Click `acme-internal`.

> "Three projects, seven flags spanning three different
> configuration strategies — JSON IR, CEL expressions, and
> TypeScript-as-code."

Click `feature-x`.

> "This is the TypeScript strategy. The flag is authored as a real
> `.ts` file using the `@falseflag/config` DSL. The dashboard
> renders the author's source verbatim — Shiki on the server side,
> no WASM, no client-side TS parser."

## 3:00 – 5:00 · Edit a flag, publish a snapshot

Click **Edit**.

> "Monaco lazy-loads. The view route's bundle is 3.6 kB; Monaco
> lives in its own chunk and ships from CDN. So this is one click
> away from a normal flag dashboard, but the production view stays
> light."

Change `default: false` to `default: true`. Click **Save**.

> "The dashboard POSTs my TypeScript source to the API. The API
> runs it through esbuild + goja — both pure Go, both inside the
> API process — and returns a compiled IR. That IR is the
> authoritative version: the same byte sequence the proxy will
> evaluate against."

Land on the view route. Point at the green toast.

> "Editing a flag created a new immutable version, but it hasn't
> propagated to anyone yet. SDKs and the proxy poll *snapshots*,
> not individual versions. The toast makes that architectural beat
> visible. One click to compile a snapshot."

Click **Publish snapshot**. Land on `/projects/acme-internal/snapshots`.

## 5:00 – 7:00 · Confirm propagation

In the terminal:

```bash
curl -sX POST http://localhost:8081/v1/evaluate \
  -H 'Content-Type: application/json' \
  -d '{"key":"feature-x","context":{"user":{"email":"alice@acme.internal"}}}' \
  | jq
```

```
{
  "value": true,
  "reason": "default",
  "version": N
}
```

> "The proxy polls the API every ten seconds. By the time this
> snapshot publish finished, the proxy already pulled the new
> bundle. We just demoed the full pipeline: dashboard → API →
> compile → snapshot → proxy → evaluation. With no page refresh."

## 7:00 – 9:00 · The CI story

```bash
ls .github/workflows/
cat .github/workflows/ci.yml | head -40
```

> "Fourteen jobs. Generate-check, lint-go, lint-js, lint-openapi,
> typecheck-js, test-go, test-js,
> sdk-conformance, contract-test, build-images, image-scan, and
> dashboard-e2e. On stock `ubuntu-latest` it's about
> 30 minutes of wall clock per push."

```bash
make sdk-conformance
```

> "Run the cross-runtime corpus through Go and TS SDKs. 27
> fixtures, byte-identical decisions on both. Slow because of cold
> startups."

> "Now imagine doing this 16 times in parallel, across two
> architectures, with the right Docker layer cache, every time
> someone pushes a branch. That is what Depot does for this repo."

*(Switch to the Depot dashboard / show before/after timings here.
This is the slice 7b material.)*

## 9:00 – 10:00 · Wrap

> "Three takeaways:
>
> 1. The platform is real enough to have real CI. A demo against a
>    toy repo doesn't tell you anything about how Depot behaves
>    when there are 16 jobs and they all want the same Docker
>    layers.
> 2. Every architectural choice that makes CI hard — generated
>    code, multi-runtime conformance, Kubernetes manifests, image
>    scans — is also a choice that makes the *application* better.
>    Depot lets you keep both.
> 3. The repo is at github.com/depot/falseflag. The slice plans
>    are in `docs/plans/` if you want to see how it was built in
>    public."

## Q&A starting points

- **"How long does each CI job take on ubuntu-latest vs Depot?"**
  See `docs/ci-baseline.md`.
- **"How does the snapshot polling avoid thundering herd?"**
  10-second jittered poll, in-memory cache; demo-quality.
- **"Why server-side TypeScript compile and not a sandboxed
  browser eval?"** esbuild + goja, no CGO, no subprocess, fits
  inside the distroless API image.
- **"Can I run this against real production data?"** No. It's a
  demo platform. Use OpenFeature-shaped APIs and a real SDK in
  production. The point is the build/test pipeline, not the
  product.
