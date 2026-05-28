# Architecture Docs

Architecture notes and diagrams for the FalseFlag platform.

## Diagram

[`diagram.mmd`](./diagram.mmd) is the canonical Mermaid source.
Render to SVG locally with the Mermaid CLI:

```bash
npm install -g @mermaid-js/mermaid-cli   # one-time
mmdc -i docs/architecture/diagram.mmd -o docs/architecture/diagram.svg
```

The diagram covers:

- **User surfaces.** Dashboard (Remix), CLI, LLM agents via MCP.
- **Control plane.** Go API server (REST + ConnectRPC), Kubernetes
  operator, MCP server.
- **Persistence.** Postgres 16 with goose-managed migrations.
- **Evaluation surfaces.** Proxy + Go/TS SDKs, all polling
  immutable snapshots.
- **Kubernetes.** Optional CRD-based configuration via the operator.

The architecture deliberately mirrors a production feature-flag
platform's surface area so the conference demo about accelerating
slow CI/CD with Depot has a believable, large repo to talk against.
See [`docs/METAPLAN.md`](../METAPLAN.md) for the slice-by-slice
build history.
