# Demo Docs

Conference demo scripts, quickstart notes, and CI comparison
documentation for FalseFlag at PlatformCon.

## Contents

- [`smoke-walkthrough.md`](./smoke-walkthrough.md) — step-by-step
  pass through the demo path with expected outputs. ~5 minutes
  from a clean clone. Use this for self-testing and dry runs.
- [`script.md`](./script.md) — the 8–10 minute conference-talk
  pacing with what-to-say + what-to-type cues.

## Off-stage pre-flight

```bash
docker compose down -v
docker compose up -d --build
make seed
make smoke    # sanity check; expect 14/14 green
```

See the [top-level README](../../README.md) for the broader
quickstart and dev-loop commands.
