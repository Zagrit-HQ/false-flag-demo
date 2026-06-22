# Demo Docs

Conference demo scripts, quickstart notes, and CI comparison
documentation for FalseFlag at PlatformCon.

## Contents

- [`script.md`](./script.md) — the 8–10 minute conference-talk
  pacing with what-to-say + what-to-type cues.

## Off-stage pre-flight

```bash
docker compose down -v
docker compose up -d --build
make seed
```

See the [top-level README](../../README.md) for the broader
quickstart and dev-loop commands.
