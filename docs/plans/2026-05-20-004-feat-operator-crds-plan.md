---
title: "feat: Kubernetes Operator and CRDs for FalseFlag"
type: feat
status: completed
date: 2026-05-20
slice: 4
---

# Kubernetes Operator and CRDs

## Overview

Slice 4 turns FalseFlag's namespace-resident Project placeholder into a
real **declarative control plane** living on Kubernetes. Seven CRDs —
`Project`, `Environment`, `Flag`, `Segment`, `RolloutPolicy`,
`FlagBinding`, and `FlagSnapshot` — describe the same domain as the
slice-3 API resources, and a controller-runtime operator reconciles
them into that API.

The operator does **not** bypass the API. Every reconciler calls the
existing `:8090` ConnectRPC surface using the already-generated
`internal/gen/proto/falseflag/v1/falseflagv1connect` client (slice 3
shipped this). The API stays the single source of truth; Kubernetes is
the declarative input. This matches the registry reference shape at
`/Users/wito/code/project-depot/registry` where the operator and the
API talk over Connect rather than sharing a Go package.

This slice also tightens the operator's "first run" experience.
Foundation laid down `operator/api/v1alpha1/` with a placeholder
cluster-scoped `Project` type and a no-op reconciler at
`operator/controllers/project_controller.go`. Slice 4 (a) **flips
`Project` to namespaced** (matches how every other CR will live), (b)
**moves reconciler implementation into `internal/operator/`** to comply
with the AGENTS.md "Go implementation under `internal/**`" rule while
keeping `operator/api/` at top-level so controller-gen + downstream CRD
consumers find it (this is the kubebuilder convention and the
foundation Makefile already points controller-gen at this path), and
(c) ships an `envtest`-backed reconciler test for every controller plus
a `make kind-smoke` end-to-end script.

The headline test artifact for this slice is the per-reconciler
envtest suite at `internal/operator/controllers/*_test.go`. Each test
spins up a real `controller-runtime` manager against an envtest
apiserver, applies a CR, stubs the Connect client with an in-memory
fake (recording every call), and asserts both the upstream call shape
and the CR status the reconciler writes back. There are no production
hardening surfaces — no leader election, no real watch fan-out, no
exponential-with-jitter backoff. Demo-quality stays the rule.

The slice also lands a Helm chart, a Kustomize overlay, sample
manifests, a `falseflag-operator` `docker-bake` target so the image
builds alongside `api`/`proxy`/`dashboard`, and an optional `make
kind-smoke` Makefile target whose script lives in `scripts/` and
exercises a kind cluster end-to-end (apply CRDs → run operator → POST a
flag CR → curl the API → tear down).

## Problem Statement / Motivation

The planning notes's product narrative claims feature flags can be authored
**three** ways: API/dashboard (slice 3), CI-shaped config compilation
(slice 2), and **declarative Kubernetes manifests** (this slice). The
last leg is the one that makes the demo audience-recognizable as a
modern control plane — every flag/release platform in the space ships
some Kubernetes integration, and the registry reference includes an
operator under `operator/`. Without slice 4 the foundation scaffold
(`operator/api/v1alpha1/project_types.go`, `operator/controllers/`,
`deploy/crds/`, `deploy/helm/`, `deploy/kustomize/`) is dead code that
the foundation README still references.

Beyond audience credibility, this slice exercises three "slow CI"
surfaces that matter for the slice-7 Depot demo:

1. **`controller-gen` codegen.** `make generate-check` already invokes
   `controller-gen object` and `controller-gen crd`. With seven CRDs it
   does meaningful work (each CR's OpenAPI v3 schema rendering takes
   noticeable time).
2. **`envtest` Go tests.** envtest downloads + boots a real
   kube-apiserver + etcd binary pair, which is slow on cold caches and
   exactly the kind of thing Depot Cache accelerates.
3. **kind smoke tests.** The `make kind-smoke` target boots an
   ephemeral cluster, applies manifests, and tears down — minutes of
   real cluster-startup work that the Depot demo can compare.

The slice also has a load-bearing product reason: per-environment flag
**overrides** are slice-4 work per the slice-3 close-out note. The API
already has the `environments` table; slice 4 introduces the
`Environment` CR and the operator-driven write path that exercises it.
A `Flag` CR with `bindings` referencing an `Environment` produces a
`FlagBinding` that, when reconciled, calls the API's
`PublishFlagVersion` with the resolved per-environment shape. This
closes the slice-3 known gap "per-environment flag overrides (slice 4
— the column exists, just not yet meaningful)".

## Proposed Solution

A single new Go binary, `cmd/falseflag-operator`, embeds a
controller-runtime manager that registers seven reconcilers. Each
reconciler reads its watched CR, calls the upstream API via the
already-generated Connect client, and writes status (conditions +
`observedGeneration` + `lastSyncTime` + resource-specific fields like
`lastPublishedVersion` or `compiledVersion`) back to the CR.

The Connect client is constructed once in the operator's startup path
(`internal/operator/run.go`) from a base URL (`FALSEFLAG_API_BASE_URL`,
default `http://localhost:8090`) and an actor string (`X-Actor` header,
default `controller/falseflag-operator`). Every reconciler receives the
same client through a `Deps` struct — no globals.

The reconciliation pattern for spec-driven CRs (`Project`,
`Environment`, `Segment`, `Flag`) is:

1. **Get** the CR. Not-found → return without requeue.
2. **Handle finalizer** (Flag/Segment only): if `DeletionTimestamp`
   is set and our finalizer is present, call the API's delete
   endpoint, remove the finalizer, return.
3. **Ensure finalizer** present on first reconcile (Flag/Segment).
4. **Translate** the CR spec into an API request body (this is the
   slice-2 IR shape for flags and segments).
5. **Upsert** via the API — `GetX` then `CreateX`/`UpdateX`. For
   Flag, additionally publish a new flag version with the translated
   rules.
6. **Read** the API's resulting object back (to capture
   server-assigned fields like `version`).
7. **Write status**: set `Ready=true`, `Synced=true`, bump
   `observedGeneration`, stamp `lastSyncTime`, write
   resource-specific fields.
8. **Requeue** in 30s on success.

The reconciliation pattern for read-only CRs (`FlagSnapshot`) is:

1. **Get** the CR.
2. **Call** `GetLatestSnapshot` for the project.
3. **Write status**: set `compiledVersion`, conditions,
   `lastSyncTime`.
4. **Requeue** in 30s.

The `FlagBinding` reconciler is the orchestration glue: when a
`FlagBinding` CR is created/updated, it (a) fetches every referenced
`Flag` CR, (b) merges their rules with any environment-specific
overrides declared in the binding, (c) calls `PublishFlagVersion` per
(flag, environment) pair, and (d) writes status with the resulting
version IDs. `RolloutPolicy` is a sibling CR that a Flag's rules can
reference by name — at translation time the reconciler inlines the
policy's bucketing strategy and rollouts into the published rules,
matching how slice 3 inlines segments.

This slice does **not** introduce a webhook (no defaulting or
validating admission). All validation lives in CRD OpenAPI schema +
controller-runtime's built-in validation + API's existing validation.

This slice does **not** introduce envtest as a transitive dependency
for the rest of the suite. The envtest tests live under
`internal/operator/controllers/` and `t.Skip` when `KUBEBUILDER_ASSETS`
is unset, so `go test ./...` works on developer laptops without
fetching kube binaries.

Demo-quality omissions called out explicitly:

- **No leader election.** Single replica only. The Helm chart sets
  `replicas: 1` and omits the leader-election RBAC.
- **No real watches between CRs.** The FlagBinding reconciler does
  not watch Flag updates and trigger re-reconcile; the 30s requeue
  covers it.
- **No multi-cluster.** The operator targets the single cluster it
  runs in. No `kubeconfig`-based remote cluster targeting.
- **No CRD conversion webhooks.** v1alpha1 is the only version
  forever; the demo never bumps.
- **No proper RBAC scoping.** The Helm chart's `ClusterRole` is a
  permissive list/watch/update/patch across the `falseflag.dev` API
  group with `*` resources. Demo-only.
- **No backoff jitter, no priority queue.** controller-runtime
  defaults only.

## Technical Considerations

- **API group `falseflag.dev`.** Matches the existing CRD already on
  disk (`deploy/crds/falseflag.dev_projects.yaml`). The planning notes
  brief suggested `falseflag.depot.dev` but the foundation chose
  `falseflag.dev`; keeping the existing string avoids breaking the
  one CRD that's already shipped. Plan documents this divergence
  explicitly so the implementer doesn't accidentally rename.
- **API version `v1alpha1`.** Never bumped during demo life.
- **Cluster vs namespaced.** The existing `Project` CRD is
  `scope: Cluster`. Slice 4 flips it to `scope: Namespaced` for
  consistency with the rest of the resource graph. All seven CRs are
  namespaced. The CR's `metadata.namespace` is **not** the project
  slug — that's a Kubernetes concept. Every child CR carries a
  `projectSlug: <string>` field in its spec to identify the upstream
  Project explicitly. (Rationale: a namespace can own CRs for
  multiple FalseFlag projects in the demo — useful for multi-tenant
  illustrations.)
- **Parent reference.** Child CRs (`Environment`, `Flag`, `Segment`,
  `RolloutPolicy`, `FlagBinding`, `FlagSnapshot`) reference their
  upstream `Project` by slug only, not by Kubernetes name. The CR
  Project is a convenience wrapper around the API Project; the
  source of truth is the API. If a child CR references a
  `projectSlug` the operator has never seen, it logs and requeues —
  it does **not** create the upstream project (that's Project's
  reconciler's job).
- **Finalizers.** `flag.falseflag.dev/finalizer` on every Flag CR;
  `segment.falseflag.dev/finalizer` on every Segment CR. On delete:
  Flag finalizer calls the API to delete the flag (cascading
  flag_versions per the existing migration FKs). Segment finalizer
  is best-effort — segments inline at publish time, so already-
  compiled snapshots survive segment deletion; the finalizer just
  cleans up the row.
- **Idempotency.** Every upstream call is a GET-then-PUT/POST shape.
  No reconciler trusts that a previous reconcile succeeded; every
  pass re-checks state.
- **Conflict handling.** If the API returns `connect.CodeAlreadyExists`
  on Create, the reconciler treats it as success and falls through
  to Update/Get. If it returns `connect.CodeFailedPrecondition` (a
  version mismatch), the reconciler logs and requeues with a 1s
  backoff (override the default).
- **Status conditions.** Use the standard
  `k8s.io/apimachinery/pkg/apis/meta/v1.Condition` type. Two
  condition types per CR: `Ready` (overall operator opinion) and
  `Synced` (last upstream call succeeded). FlagSnapshot adds
  `Compiled`. RolloutPolicy adds `Resolved` (every referenced
  segment exists).
- **Per-environment overrides.** A Flag CR's `spec.bindings` is an
  optional list of `{environment, defaultValue, rules}`. When
  present, the Flag reconciler publishes one flag version per
  (flag, environment) tuple. When absent, it publishes a single
  default version with no environment scope. The API already has
  the schema for this from slice 3.

## System-Wide Impact

### Interaction Graph

```
kubectl apply -f flag.yaml
   ▼
kube-apiserver stores Flag CR (etcd write)
   ▼
controller-runtime informer notices change
   ▼
FlagReconciler.Reconcile(ctx, req)  [internal/operator/controllers/flag_controller.go]
   ▼
   ├── Get Flag CR from cache
   ├── Translate spec → IR JSON (internal/operator/translate.IRForFlag)
   ├── Resolve RolloutPolicy refs (Get from cache)
   ├── falseflagv1connect.FlagsServiceClient.GetFlag(ctx, ...)
   │       ▼
   │       falseflag-api :8090 (slice 3)
   │       ▼
   │       internal/server/rpc/flags.go
   │       ▼
   │       internal/store.GetFlagByKey
   │       ▼
   │       Postgres
   ├── (404 → CreateFlag, else fall through)
   ├── falseflagv1connect.FlagsServiceClient.PublishFlagVersion(...)
   │       (one call per environment binding; one default call if no bindings)
   │       ▼
   │       internal/server/rpc/flags.go → internal/store.PublishFlagVersion
   │       (slice 3 segment inlining still applies)
   │       ▼
   │       audit_events row written via Store.WithAudit
   ├── Update CR.Status (Ready, Synced, lastPublishedVersion, observedGeneration, lastSyncTime)
   └── ctrl.Result{RequeueAfter: 30s}
```

### Error & Failure Propagation

| Layer | Failure | Mapped to |
|---|---|---|
| Connect client | network / 502 | `ctrl.Result{}, err` → controller-runtime backoff |
| Connect client | `CodeAlreadyExists` | swallowed, fall through to Update |
| Connect client | `CodeNotFound` | only treated as expected on the initial GET |
| Connect client | `CodeFailedPrecondition` | `ctrl.Result{RequeueAfter: 1s}, nil` + condition `Synced=False` |
| Connect client | `CodeInvalidArgument` | terminal — condition `Ready=False`, no requeue (user spec bug) |
| Translator | invalid spec (bad IR shape) | terminal — condition `Ready=False, reason=InvalidSpec`, no requeue |
| K8s client | `Status` update conflict | log, requeue immediately (controller-runtime handles) |
| K8s client | CR not found | return without requeue |

All errors flow through one helper `internal/operator/controllers/result.go:translateError(err) (ctrl.Result, error, *metav1.Condition)`
that centralizes the mapping. Tests assert the mapping for each
upstream Connect error code.

### State Lifecycle Risks

- **Orphaned upstream resources.** If a user deletes a Flag CR but
  the operator pod is down, the finalizer keeps the CR around until
  the operator returns. This is fine — finalizers are exactly the
  mechanism for this. Tests cover the finalizer path explicitly.
- **Orphaned CRs after upstream deletion.** If a user deletes a
  Flag through the API or dashboard directly, the Flag CR remains
  but its reconciler will recreate the upstream Flag on next
  reconcile (because GET → 404 → CreateFlag). This is "the CR wins"
  behavior, intentional for declarative control planes. Documented
  in the operator README.
- **Partial publish failure.** A Flag with three environment
  bindings publishes three flag versions sequentially. If the
  second call fails, the first is already committed. The
  reconciler does **not** roll back. On retry it will re-publish
  the same shape; the API's publish endpoint is idempotent-by-shape
  (it returns the existing version if nothing changed). This is the
  demo-quality call.
- **Project create race.** If a user `kubectl apply`s a Project and
  five children in the same manifest, the children's reconcilers
  may run before the Project reconciler completes. They handle this
  by checking `GetProject(projectSlug)` and requeueing on 404 with a
  2s backoff. Documented in each reconciler.

### API Surface Parity

| Concept | API (slice 3) | CRD (slice 4) |
|---|---|---|
| Project | `POST /v1/projects` | `Project` CR |
| Environment | `POST /v1/projects/{slug}/environments` | `Environment` CR |
| Segment | `POST /v1/projects/{slug}/segments` | `Segment` CR |
| Flag (create) | `POST /v1/projects/{slug}/flags` | `Flag` CR |
| Flag (publish) | `PUT /v1/projects/{slug}/flags/{key}` | `Flag.spec.rules` change triggers publish |
| Rollout | inline in flag IR | `RolloutPolicy` CR (referenced by name) |
| Snapshot | `POST /v1/projects/{slug}/snapshots/compile` | `FlagSnapshot` CR (read-only) |
| Audit | `GET /v1/projects/{slug}/audit` | not exposed on CR — auditing is API-side |

No new API endpoints. No proto changes. No oapi-codegen changes. No
sqlc changes. **`buf.lock` and `sqlc.yaml` are untouched in this
slice.**

### Integration Test Scenarios

1. **End-to-end project lifecycle (envtest).** Apply a Project CR →
   reconcile → assert API has the project → update displayName →
   reconcile → assert API has the new displayName → delete Project
   CR → reconcile → assert finalizer (NO finalizer on Project for
   slice 4 — Project deletes only the CR, not the API project) →
   CR gone.
2. **Flag with rollout policy (envtest).** Apply RolloutPolicy +
   Flag CRs → reconcile both → assert the published flag version's
   IR contains the inlined rollouts → modify the RolloutPolicy CR
   → reconcile Flag → assert republish with updated rollouts.
3. **FlagBinding fan-out (envtest).** Apply 3 Environments + 1 Flag +
   1 FlagBinding (referencing all 3 envs) → reconcile binding →
   assert 3 PublishFlagVersion calls (one per env).
4. **Segment delete with active flag reference (envtest).** Apply
   Segment, then Flag that references it by key, reconcile, delete
   Segment CR → finalizer runs → API DeleteSegment → flag publish
   on next reconcile fails (segment 404) but the previously
   compiled snapshot still works (slice 3 inlining property).
5. **kind smoke (`make kind-smoke`).** kind up → apply CRDs → run
   operator binary as a `kubectl run` pod → apply a Project + Flag
   sample → poll API `/v1/projects/{slug}/flags` until non-empty
   → tear down.

## Acceptance Criteria

### Functional Requirements

- [ ] `operator/api/v1alpha1/` contains type files for all seven CRDs:
  `project_types.go` (modified to namespaced), `environment_types.go`,
  `flag_types.go`, `segment_types.go`, `rolloutpolicy_types.go`,
  `flagbinding_types.go`, `flagsnapshot_types.go`. Each has Spec +
  Status structs, list wrapper, kubebuilder markers, and conditions
  in status.
- [ ] `operator/api/v1alpha1/zz_generated.deepcopy.go` is regenerated
  by `controller-gen object` and committed.
- [ ] `deploy/crds/` contains exactly seven CRD YAML files
  (`falseflag.dev_projects.yaml`, `..._environments.yaml`,
  `..._flags.yaml`, `..._segments.yaml`, `..._rolloutpolicies.yaml`,
  `..._flagbindings.yaml`, `..._flagsnapshots.yaml`), all generated by
  `controller-gen crd` and committed.
- [ ] `cmd/falseflag-operator/main.go` exists (<50 lines), wraps
  `buildinfo.WithGracefulShutdown("operator", run)`, delegates to
  `internal/operator/run.go`.
- [ ] `internal/operator/run.go` boots a controller-runtime manager,
  constructs a Connect client from `FALSEFLAG_API_BASE_URL`,
  registers seven reconcilers, blocks until ctx.Done().
- [ ] Seven reconcilers under `internal/operator/controllers/`:
  `project_controller.go`, `environment_controller.go`,
  `flag_controller.go`, `segment_controller.go`,
  `rolloutpolicy_controller.go`, `flagbinding_controller.go`,
  `flagsnapshot_controller.go`. Each has a `Reconcile` method and a
  `SetupWithManager` method.
- [ ] `internal/operator/controllers/result.go` provides the
  `translateError` helper used by every reconciler.
- [ ] `internal/operator/translate/` package converts CR specs to
  IR JSON: `IRForFlag(*v1alpha1.Flag, []*v1alpha1.RolloutPolicy)
  (*config.RulesTree, error)`, `IRForSegment(*v1alpha1.Segment)
  (*config.Predicate, error)`. Has its own unit tests (no envtest
  needed).
- [ ] `internal/operator/controllers/` envtest suite covers all
  five integration scenarios listed above. `t.Skip` when
  `KUBEBUILDER_ASSETS` is unset.
- [ ] `deploy/helm/falseflag-operator/` contains a working Helm
  chart: `Chart.yaml`, `values.yaml`, `templates/deployment.yaml`,
  `templates/serviceaccount.yaml`, `templates/clusterrole.yaml`,
  `templates/clusterrolebinding.yaml`, `templates/crds/` (Helm
  doesn't manage CRDs; this directory is for `helm install
  --include-crds`).
- [ ] `deploy/kustomize/` contains an overlay structure:
  `base/kustomization.yaml`, `base/deployment.yaml`,
  `base/rbac.yaml`, `overlays/dev/kustomization.yaml`,
  `overlays/dev/patch.yaml`.
- [ ] `deploy/samples/` contains end-to-end sample manifests:
  `project.yaml`, `environment.yaml`, `flag-json.yaml`,
  `flag-with-rollout.yaml`, `segment.yaml`, `flagbinding.yaml`,
  `flagsnapshot.yaml`, plus a `kustomization.yaml` that applies
  them in order.
- [ ] `scripts/kind-smoke.sh` exists, is executable, and runs the
  full kind cluster lifecycle (up → apply → assert → down).
- [ ] `Makefile` gains `kind-smoke` target. `make help` lists it.
- [ ] `docker-bake.hcl` gains a `falseflag-operator` target.
  `make bake-print` shows it in the default group alongside
  `api`/`proxy`/`dashboard`.

### Non-Functional Requirements

- [ ] No `pkg/**` directory created.
- [ ] Reconciler implementations live under `internal/operator/**`
  per AGENTS.md.
- [ ] `operator/api/v1alpha1/` stays at top-level (kubebuilder
  convention + foundation Makefile already targets it). The
  existing `operator/controllers/` directory is **removed** in this
  slice — its contents move to `internal/operator/controllers/`,
  and the new location is what the manager registers.
- [ ] `cmd/falseflag-operator/main.go` stays under 50 lines.
- [ ] All env vars use the `FALSEFLAG_` prefix.
- [ ] Logging via `internal/logging.New("operator")` only.
- [ ] No new top-level Go module. Single `go.mod` continues to
  cover everything.

### Quality Gates

- [ ] `go build ./cmd/...` ✓ (now includes `cmd/falseflag-operator`)
- [ ] `go vet ./...` ✓
- [ ] `go test ./...` ✓ — envtest cases self-skip on missing
  `KUBEBUILDER_ASSETS`
- [ ] `KUBEBUILDER_ASSETS=$(go tool setup-envtest use -p path 1.31.0)
  go test ./internal/operator/...` ✓ — every reconciler test runs
- [ ] `FALSEFLAG_TEST_DATABASE_URL=… go test ./internal/store/...
  ./internal/server/...` ✓ — unchanged from slice 3, must stay green
- [ ] `make generate-check` ✓ — controller-gen output is idempotent
  on a clean tree
- [ ] `make contract-test` ✓ — unchanged from slice 3
- [ ] `pnpm --dir js -r typecheck` ✓ — unchanged
- [ ] `pnpm --dir js -r test` ✓ — unchanged
- [ ] `pnpm --dir js -r build` ✓ — unchanged
- [ ] `pnpm --dir js lint` ✓ — unchanged
- [ ] `make smoke` ✓ — 10 Hurl files unchanged
- [ ] `make bake-print` ✓ — now lists `falseflag-operator` in the
  default group
- [ ] `make kind-smoke` ✓ when docker + kind are installed locally
  (best-effort gate; optional in CI)

## Technical Approach

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  Kubernetes cluster                                                 │
│                                                                     │
│  ┌──────────────┐    apply    ┌─────────────────────────────────┐   │
│  │   kubectl    │ ─────────▶  │   kube-apiserver                │   │
│  └──────────────┘             │   (CRDs: Project, Environment,  │   │
│                               │    Flag, Segment, RolloutPolicy,│   │
│                               │    FlagBinding, FlagSnapshot)   │   │
│                               └──────────────┬──────────────────┘   │
│                                              │ informer cache       │
│                                              ▼                      │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │  falseflag-operator (Deployment, replicas=1)                │   │
│   │                                                             │   │
│   │  internal/operator/run.go                                   │   │
│   │    ├── ctrl.NewManager(...)                                 │   │
│   │    ├── ProjectReconciler                                    │   │
│   │    ├── EnvironmentReconciler                                │   │
│   │    ├── FlagReconciler         ─┐                            │   │
│   │    ├── SegmentReconciler       │ all share one Connect      │   │
│   │    ├── RolloutPolicyReconciler │ client + X-Actor header    │   │
│   │    ├── FlagBindingReconciler   │                            │   │
│   │    └── FlagSnapshotReconciler ─┘                            │   │
│   └──────────────────────┬──────────────────────────────────────┘   │
│                          │ Connect/HTTP2                            │
└──────────────────────────┼──────────────────────────────────────────┘
                           │
                           ▼
              ┌────────────────────────┐
              │  falseflag-api :8090   │
              │  (slice 3, untouched)  │
              └────────────────────────┘
```

The Connect client is the operator's only outbound dependency. The
operator does not connect to Postgres, Redis, or any other store
directly — even though slice 3 wired a Store-aware test path, the
operator does not link in `internal/store`.

#### Resource model (CR → API translation)

```
Project CR
  spec:
    projectSlug: "demo"          # natural key; primary identifier
    displayName: "Demo"
    configStrategy: "json"        # one of json|cel|typescript
  status:
    conditions: [{type: Ready, status: True}, {type: Synced, ...}]
    observedGeneration: 7
    lastSyncTime: 2026-05-20T12:00:00Z

  → POST /v1/projects {slug, display_name, config_strategy}

Environment CR
  spec:
    projectSlug: "demo"
    slug: "prod"
    name: "Production"
  status:
    conditions: [...]
    observedGeneration: 1
    lastSyncTime: ...

  → POST /v1/projects/demo/environments {slug, name}

Segment CR
  spec:
    projectSlug: "demo"
    key: "internal-users"
    name: "Internal Users"
    description: "Depot employees"
    predicate:                   # IR Predicate JSON
      kind: comparison
      attr: email
      op: ends_with
      value: "@depot.dev"
  status:
    conditions: [..., Synced]
    observedGeneration: 1
    lastSyncTime: ...

  → POST /v1/projects/demo/segments {key, name, description, predicate}

RolloutPolicy CR
  spec:
    projectSlug: "demo"
    name: "fifty-fifty"
    bucketing:
      attribute: "user_id"
      strategy: "fnv1a_64"
    rollouts:
      - id: "control"
        weight: 50
        value: false
      - id: "treatment"
        weight: 50
        value: true
  status:
    conditions: [..., Resolved]
    observedGeneration: 1

  (RolloutPolicy never makes API calls — it's a translation-time
   reference for Flag CRs. Reconciler verifies it parses + writes
   status only.)

Flag CR
  spec:
    projectSlug: "demo"
    key: "banner"
    name: "Banner"
    valueType: "boolean"
    default: false
    rules:                       # IR Rule list (slice 2 shape)
      - id: "rollout"
        when: {kind: always}
        rolloutRef: "fifty-fifty"   # ← references RolloutPolicy by name
      - id: "static"
        when: {kind: comparison, attr: env, op: eq, value: "prod"}
        value: true
    bindings:                    # optional per-env overrides
      - environment: "staging"
        default: true
        rules: []
  status:
    conditions: [...]
    observedGeneration: 12
    lastSyncTime: ...
    lastPublishedVersion: 5

  → POST /v1/projects/demo/flags (idempotent create)
  → PUT /v1/projects/demo/flags/banner {strategy: "json", source: {...}}
    once per binding (or once for the unbound default)

FlagBinding CR
  spec:
    projectSlug: "demo"
    flagKey: "banner"
    environments: ["staging", "prod"]
    overrides:                   # optional per-env value override
      staging: false
      prod: true
  status:
    conditions: [...]
    observedGeneration: 3
    publishedVersions:           # one entry per environment
      staging: 4
      prod: 5

  → For each environment in spec.environments:
    GET /v1/projects/demo/flags/banner
    PUT /v1/projects/demo/flags/banner {strategy, source with override}

FlagSnapshot CR (read-only)
  spec:
    projectSlug: "demo"
  status:
    conditions: [..., Compiled]
    compiledVersion: 7           # ← what GetLatestSnapshot returned
    lastSyncTime: ...
    flagCount: 12

  → GET /v1/projects/demo/snapshots/latest
```

#### Status conditions

Every CR's status has a `conditions []metav1.Condition` array. Two
condition types are universal:

- `Ready`: the reconciler's overall opinion of the CR. False on any
  terminal error (bad spec, upstream rejected).
- `Synced`: the last upstream call succeeded. False on transient
  upstream error (network, 5xx, conflict).

Resource-specific conditions:

- `FlagSnapshot`: `Compiled` — the API has at least one snapshot for
  this project.
- `RolloutPolicy`: `Resolved` — all referenced segment keys exist in
  the cluster (we check the K8s cache, not the API).
- `FlagBinding`: `Published` — every binding produced a published
  version on the most recent reconcile.

### Implementation Phases

#### Phase 1: CRD Types + Generated Artifacts

**Goal:** Land all seven CRD type files + their controller-gen
output. The operator binary still doesn't build at the end of this
phase (no `cmd/falseflag-operator/main.go` yet), but `make generate`
is green.

**Owns:**
- `operator/api/v1alpha1/groupversion_info.go` (modified: register
  six new types)
- `operator/api/v1alpha1/project_types.go` (modified: namespaced
  scope, status with conditions)
- `operator/api/v1alpha1/environment_types.go` (new)
- `operator/api/v1alpha1/flag_types.go` (new — includes IR Rule
  embedded type)
- `operator/api/v1alpha1/segment_types.go` (new — includes IR
  Predicate embedded type)
- `operator/api/v1alpha1/rolloutpolicy_types.go` (new)
- `operator/api/v1alpha1/flagbinding_types.go` (new)
- `operator/api/v1alpha1/flagsnapshot_types.go` (new)
- `operator/api/v1alpha1/zz_generated.deepcopy.go` (regen, committed)
- `deploy/crds/*.yaml` (regen, committed — seven files)
- `operator/api/v1alpha1/types_test.go` (new — DeepCopy round-trip
  tests, ~5 cases)

**Risks:**
- **CRD validation schema gets too big.** The Flag CR embeds the
  full IR Rule shape, which includes a recursive Predicate. CRD
  OpenAPI v3 supports recursion via `x-kubernetes-preserve-unknown-fields`.
  Mark the Predicate field with that annotation so controller-gen
  emits a permissive schema; rely on the API's own predicate
  validation as the source of truth.
- **embedded RawExtension vs concrete types.** For Rules, use
  `runtime.RawExtension` with a JSON tag so users can write IR
  shapes literally in YAML. For Predicate, same.

**Acceptance:**
- `go vet ./operator/...` clean
- `make generate-check` clean
- DeepCopy tests pass

#### Phase 2: Operator Runner + Manager Wiring + Connect Client

**Goal:** `go build ./cmd/falseflag-operator` succeeds. Binary boots,
registers a logger, opens a Connect client, starts a manager with no
reconcilers attached, blocks on ctx.

**Owns:**
- `cmd/falseflag-operator/main.go` (new, <50 lines)
- `internal/operator/run.go` (new — manager construction, client
  construction, reconciler registration scaffolded but commented
  out)
- `internal/operator/config.go` (new — `Config` struct,
  `LoadConfig() (Config, error)` reading `FALSEFLAG_API_BASE_URL`,
  `FALSEFLAG_OPERATOR_ACTOR`, `FALSEFLAG_OPERATOR_METRICS_ADDR`)
- `internal/operator/client.go` (new — Connect client factory; wraps
  `falseflagv1connect.NewProjectsServiceClient(...)` and the six
  siblings into one `APIClient` struct that reconcilers depend on)
- `internal/operator/run_test.go` (new — table-driven test for
  config loading, no manager boot)
- `internal/appconfig/appconfig.go` (modified — add `OperatorConfig`
  struct + `LoadOperator()`; mirrors `LoadAPI()` shape)

**Risks:**
- **buildinfo.WithGracefulShutdown signature.** Existing API binary
  uses `WithGracefulShutdown("api", run)` — slice 4 just adds a
  caller, no signature changes.
- **APIClient as one fat struct.** Tempting to break it into one
  field per service; resist. One struct with seven embedded
  Connect clients keeps reconciler deps trivial and is the demo
  reality (operator talks to one server).
- **Manager scheme registration.** Manager needs the v1alpha1
  scheme. Add `runtime.NewScheme()` + `v1alpha1.AddToScheme` + the
  default `clientgoscheme.AddToScheme` in `run.go`.

**Acceptance:**
- `go build ./cmd/falseflag-operator` succeeds
- `./falseflag-operator` boots, logs `starting falseflag-operator`,
  blocks on ctx (manual smoke; not committed as a test)

#### Phase 3: Reconcilers (per-resource)

**Goal:** All seven reconcilers implemented. envtest suite green
when `KUBEBUILDER_ASSETS` is set. Operator binary registers them
all and a `kubectl apply -f deploy/samples/` flow works against a
local API server.

**Owns:**
- `internal/operator/controllers/project_controller.go` (new — replaces
  the slice-1 stub at `operator/controllers/project_controller.go`)
- `internal/operator/controllers/environment_controller.go` (new)
- `internal/operator/controllers/segment_controller.go` (new — with
  finalizer)
- `internal/operator/controllers/flag_controller.go` (new — with
  finalizer; the most complex reconciler)
- `internal/operator/controllers/rolloutpolicy_controller.go` (new)
- `internal/operator/controllers/flagbinding_controller.go` (new)
- `internal/operator/controllers/flagsnapshot_controller.go` (new)
- `internal/operator/controllers/result.go` (new — `translateError`
  + helper to set conditions)
- `internal/operator/controllers/suite_test.go` (new — envtest
  bootstrap shared by every test file, `t.Skip` on missing
  `KUBEBUILDER_ASSETS`)
- `internal/operator/controllers/project_controller_test.go` (new)
- `internal/operator/controllers/environment_controller_test.go` (new)
- `internal/operator/controllers/segment_controller_test.go` (new)
- `internal/operator/controllers/flag_controller_test.go` (new — the
  biggest, covers finalizer + per-env binding fan-out + rollout
  inlining)
- `internal/operator/controllers/rolloutpolicy_controller_test.go`
  (new)
- `internal/operator/controllers/flagbinding_controller_test.go`
  (new)
- `internal/operator/controllers/flagsnapshot_controller_test.go`
  (new)
- `internal/operator/translate/translate.go` (new — pure-Go IR
  translation helpers)
- `internal/operator/translate/translate_test.go` (new — table tests
  for IR translation, no envtest)
- `internal/operator/client_fake.go` (new — `FakeAPIClient` records
  every call; used by every reconciler test)
- `internal/operator/run.go` (modified — uncomment reconciler
  registrations from phase 2)
- `operator/controllers/` directory: **removed**. Tests in the new
  location supersede `project_controller_test.go`.

**Risks:**
- **envtest binary version.** envtest is locked to a specific
  kube-apiserver version. Phase 2 phase pins `setup-envtest use
  1.31.0` (matches the apimachinery v0.36.0 in go.mod). The
  `Makefile` exposes this through `KUBEBUILDER_ASSETS_VERSION ?=
  1.31.0` so CI can override.
- **Status update conflicts.** controller-runtime sometimes
  returns a conflict on Status() write when the spec also changed
  in the same reconcile pass. Wrap status updates in
  `retry.RetryOnConflict` from `client-go/util/retry`.
- **Finalizer + delete reconcile ordering.** A common bug: removing
  the finalizer must be the very last step after the upstream
  delete succeeds, and the reconciler must return early after
  removing it. Tests for this explicitly.
- **FlagBinding observing Flag changes.** Slice 4 does NOT wire a
  watch from FlagBinding to Flag. If a Flag's rules change, the
  Flag reconciler republishes; the FlagBinding will republish on
  its own 30s requeue. This may surface as a flaky test if not
  handled — assert binding status after at least one Flag
  reconcile + binding reconcile cycle, not after a single tick.
  Use `Eventually(...)` with a 5s timeout.
- **`controller-runtime`'s manager Start blocks.** Reconciler tests
  start the manager in a goroutine in TestMain and stop it in
  cleanup. envtest provides this pattern; mirror keat's setup if
  needed.

**Acceptance:**
- `KUBEBUILDER_ASSETS=… go test ./internal/operator/...` ✓
- `go test ./internal/operator/translate/...` ✓ (no envtest needed)
- Manual smoke: kind cluster + sample manifests + running operator
  produces upstream API objects (slice tested via phase 5 script)

#### Phase 4: Helm Chart + Kustomize Overlay + Samples + Docker-Bake

**Goal:** A user can deploy the operator with `helm install` or
`kubectl apply -k` and see it run. CRDs ship in the chart. The
operator image is part of `docker-bake.hcl`.

**Owns:**
- `deploy/helm/falseflag-operator/Chart.yaml` (new)
- `deploy/helm/falseflag-operator/values.yaml` (new)
- `deploy/helm/falseflag-operator/templates/_helpers.tpl` (new)
- `deploy/helm/falseflag-operator/templates/serviceaccount.yaml`
  (new)
- `deploy/helm/falseflag-operator/templates/clusterrole.yaml` (new)
- `deploy/helm/falseflag-operator/templates/clusterrolebinding.yaml`
  (new)
- `deploy/helm/falseflag-operator/templates/deployment.yaml` (new)
- `deploy/helm/falseflag-operator/templates/configmap.yaml` (new —
  holds the `FALSEFLAG_API_BASE_URL` etc. as env)
- `deploy/helm/falseflag-operator/crds/*.yaml` (new — symlinks or
  copies of the seven generated CRDs; Helm convention is `crds/`,
  not `templates/`)
- `deploy/helm/falseflag-operator/.helmignore` (new)
- `deploy/kustomize/base/kustomization.yaml` (new)
- `deploy/kustomize/base/namespace.yaml` (new)
- `deploy/kustomize/base/deployment.yaml` (new)
- `deploy/kustomize/base/rbac.yaml` (new)
- `deploy/kustomize/base/configmap.yaml` (new)
- `deploy/kustomize/base/crds.yaml` (new — aggregates the seven
  CRDs)
- `deploy/kustomize/overlays/dev/kustomization.yaml` (new)
- `deploy/kustomize/overlays/dev/patch.yaml` (new — sets
  `FALSEFLAG_API_BASE_URL=http://host.docker.internal:8090` for
  local dev against a host-running API)
- `deploy/samples/project.yaml` (new)
- `deploy/samples/environment.yaml` (new)
- `deploy/samples/flag-json.yaml` (new)
- `deploy/samples/flag-with-rollout.yaml` (new)
- `deploy/samples/segment.yaml` (new)
- `deploy/samples/flagbinding.yaml` (new)
- `deploy/samples/flagsnapshot.yaml` (new)
- `deploy/samples/kustomization.yaml` (new — applies all of the
  above in dependency order)
- `docker-bake.hcl` (modified — add `falseflag-operator`
  target, include in default group)
- `infra/Dockerfile.operator` (new — multi-stage Go build mirroring
  `Dockerfile.api`)
- `deploy/README.md` (modified — quickstart for both Helm and
  Kustomize paths)
- `deploy/helm/README.md` (modified)
- `deploy/kustomize/README.md` (modified)

**Risks:**
- **Helm CRD lifecycle.** Helm install applies CRDs in `crds/`
  once, never upgrades them. Document this in the README — for
  CRD changes during demo, use `helm install --upgrade
  --force-conflicts` or `kubectl apply -f deploy/crds/`. Demo-
  quality is fine.
- **Kustomize CRD ordering.** Kustomize applies resources in a
  fixed order — CRDs first because they're cluster-scoped.
  Verified with `kubectl apply -k deploy/kustomize/overlays/dev`.
- **Docker bake build cache.** Adding a fourth target multiplies
  total build time. Use the same multi-stage shape as
  `Dockerfile.api` so the Go build layer caches across targets.

**Acceptance:**
- `helm lint deploy/helm/falseflag-operator` ✓
- `kubectl kustomize deploy/kustomize/overlays/dev > /dev/null` ✓
- `make bake-print` lists `falseflag-operator` ✓
- Manual: `helm install` and `kubectl apply -k` both produce a
  running operator pod (verified during phase 5)

#### Phase 5: kind Smoke + Close-Out

**Goal:** `make kind-smoke` provides a one-command end-to-end demo.
Plan flipped to `status: completed`.

**Owns:**
- `scripts/kind-smoke.sh` (new — executable, kind up/down + apply +
  curl assertions)
- `Makefile` (modified — `kind-smoke` target + `make help` entry)
- `docs/plans/2026-05-20-004-feat-operator-crds-plan.md` (modified —
  flip `status: active` → `status: completed`, tick all phase
  checklists)
- `operator/README.md` (modified — operator usage, env vars,
  troubleshooting)
- `cmd/falseflag-operator/README.md` (modified — entrypoint docs)
- `internal/operator/README.md` (new)
- `tests/hurl/` — **unchanged** in this slice. Operator is not
  exercised by Hurl; envtest covers reconciliation.

**Risks:**
- **kind smoke flakiness in CI.** `make kind-smoke` is documented
  as best-effort. The validation ladder explicitly notes it can
  be skipped if docker isn't available; the script must still
  exist and be executable.
- **kind binary not installed.** Script checks `command -v kind`
  first; if missing, prints install hint and exits 0 (so the
  validation ladder doesn't fail on developer machines without
  kind).
- **Close-out commit not actually applying generate.** Last-mile
  bug from slice 3: someone updates types but forgets to commit
  the regenerated deepcopy. `make generate-check` in the
  validation ladder catches this.

**Acceptance:**
- `make kind-smoke` runs to completion on a machine with docker +
  kind installed; outputs "✓ smoke passed"
- `make help` lists `kind-smoke`
- planning notes slice-4 checklist all checked
- Plan frontmatter shows `status: completed`

## Alternative Approaches Considered

**Talk to the Store directly, not the API.**
Rejected: violates the "operator is a control-plane consumer"
principle. Defeats the point of having a separate API surface.
Also forces the operator to link in pgx + sqlc, balloons the
binary, and makes the operator's tests depend on Postgres.

**Talk to the REST surface instead of Connect.**
Rejected: the slice-3 plan explicitly framed Connect as the
internal service-to-service path. The generated Connect client is
strongly typed and idiomatic; the REST client (Orval) is generated
for browser consumption. Less Go boilerplate, fewer JSON encoders.

**Use kubebuilder's project layout (`PROJECT` file + scaffolded
binary)**
Rejected: kubebuilder's scaffolding creates a `cmd/manager/main.go`
with its own dep tree and `Dockerfile` patterns that don't match
FalseFlag's `cmd/falseflag-*` shape. We use controller-runtime
directly + controller-gen as the generators, the slimmest path
that satisfies the planning notes's Kubebuilder requirement.

**Single Reconciler for all CRs.**
Rejected: idiomatic controller-runtime is one reconciler per
resource. The seven-reconciler split has clear ownership and lets
us write seven focused envtest cases.

**Per-environment overrides as a separate `EnvironmentOverride`
CR.**
Rejected: the API doesn't model overrides as a separate row, just
as flag versions scoped to an environment. The `FlagBinding` CR
matches the existing `flag_versions.environment_id` column shape.
One CR per (Flag, Environments) tuple, not per (Flag, Environment).

**Use admission webhooks for validation.**
Rejected: webhooks require certs + a `cert-manager` chart + cluster
networking + a separate `ValidatingWebhookConfiguration`. The
CRD's OpenAPI schema + the API's own validation catches every error
we care about for demo purposes.

**Status `compiledVersion` populated via FlagSnapshot watching
the API.**
The current design is poll-based (30s requeue). Considered an
API-side notification webhook back to the operator; rejected as
too much demo plumbing. Slice 4 ships a `make kind-smoke` that
shows users `kubectl wait --for=condition=Compiled` works within
two requeue cycles, which is fine for the conference talk.

## Risk Analysis & Mitigation

### Risk: envtest binary not cached on developer machines

**Mitigation:** Pin `KUBEBUILDER_ASSETS_VERSION=1.31.0` in the
Makefile. The `setup-envtest` go-tool is already in the foundation
toolset; running `go tool setup-envtest use 1.31.0` populates the
cache once. The `go test` invocation in the operator suite calls
`t.Skip` if `KUBEBUILDER_ASSETS` is empty so the default
`go test ./...` invocation passes on machines without the binaries.

### Risk: controller-gen and IR types diverge

**Mitigation:** The CR types reference the IR shapes via
`runtime.RawExtension` for Predicates and Rules. controller-gen
emits permissive schemas (`x-kubernetes-preserve-unknown-fields:
true`); the operator's translator validates the shape at reconcile
time using `internal/config.ValidatePredicate` (slice 3
introduced this exported helper). Tests assert that valid IR
round-trips and invalid IR produces a `Ready=False` condition with
`reason=InvalidSpec`.

### Risk: kind smoke flaky on CI runners

**Mitigation:** `make kind-smoke` is **optional** in the validation
ladder. It must exist and pass on a developer machine with docker +
kind installed; CI doesn't gate on it for slice 4. If slice 7 wires
it into a CI matrix, that's slice 7's risk to own.

### Risk: Helm install fails on Project CR upgrade

**Mitigation:** Helm doesn't manage CRD lifecycle. Document this
in `deploy/helm/README.md`. The slice provides a
`kubectl apply -f deploy/crds/` alternative for CRD upgrades.

### Risk: Operator pod can't reach the API

**Mitigation:** `FALSEFLAG_API_BASE_URL` default is
`http://falseflag-api.default.svc.cluster.local:8090` (in-cluster
DNS). The Kustomize `dev` overlay overrides this to
`http://host.docker.internal:8090` for laptop development. The
kind smoke script exposes the API via `kubectl port-forward` or a
host network sidecar — TBD; the simplest path is to run the API
locally and have the operator pod use `host.docker.internal`.

### Risk: New Go dependencies bloat the binary

**Mitigation:** No new Go dependencies. apimachinery, client-go,
controller-runtime are already in `go.mod` (foundation). The
operator binary will be ~30MB stripped — same order of magnitude
as the API binary.

### Risk: `make generate-check` fails because deepcopy not committed

**Mitigation:** Slice 1 set the precedent: commit
`zz_generated.deepcopy.go` and `deploy/crds/*.yaml`. Every Phase
1 commit ends with a regenerate + commit step. The validation
ladder's `make generate-check` is the final gate.

## Success Metrics

- Seven CRDs `kubectl apply`-able with sane defaults.
- Seven reconcilers pass envtest.
- One `kubectl apply -k deploy/samples/` produces a Project +
  Environment + Flag + FlagBinding in the upstream API within 60s.
- One `helm install` produces a running operator pod talking to
  the API.
- All foundation slice 1/2/3 validation steps still pass.
- Demo audience can see a YAML file, see a Kubernetes resource,
  see the dashboard reflect the change, see the audit event with
  `actor=controller/falseflag-operator`.

## Dependencies & Prerequisites

- Slice 1 foundation: cmd/falseflag-operator placeholder,
  operator/api/v1alpha1 directory, deploy/{crds,helm,kustomize}
  scaffolding, Makefile `controller-gen` invocations,
  `k8s.io/apimachinery` + `k8s.io/client-go` +
  `sigs.k8s.io/controller-runtime` in go.mod.
- Slice 2 IR: `internal/config.RulesTree`, `internal/config.Rule`,
  `internal/config.Predicate`, `internal/config.ValidatePredicate`.
- Slice 3 API: `internal/gen/proto/falseflag/v1/falseflagv1connect`
  clients for every resource. `:8090` Connect listener.
- Local dev: docker + kind for `make kind-smoke`. Operator runs
  fine against `make api-dev` on the host for envtest-free flows.

## Future Considerations

- **Slice 5** consumes status conditions to display "deployed via
  Kubernetes" badges in the dashboard.
- **Slice 6 MCP** could expose a `kubectl apply` tool, but that's
  out of scope for slice 4.
- **Slice 7 CI** can light up envtest as a slow-CI surface.
- **Out of scope forever**: ArgoCD ApplicationSet integration,
  GitOps reconciliation drift detection, multi-cluster fleet
  management, OpenTelemetry tracing of reconcile loops.

## Documentation Plan

- `operator/README.md`: full operator usage doc — `Concepts`,
  `Quickstart with kind`, `Quickstart with Helm`,
  `Quickstart with Kustomize`, `CR Reference` (one section per
  CRD with example YAML), `Troubleshooting`.
- `cmd/falseflag-operator/README.md`: env vars, exit codes.
- `internal/operator/README.md`: developer doc — architecture
  overview, how to add a new CRD, how to run envtest, finalizer
  conventions.
- `deploy/helm/falseflag-operator/README.md`: Helm install +
  upgrade + uninstall.
- `deploy/kustomize/README.md`: Kustomize overlay structure.
- `deploy/samples/README.md`: sample manifest tour.
- `deploy/crds/README.md`: notes on regeneration.

## Sources & References

### Internal References

- `docs/plans/2026-05-20-001-feat-foundation-monorepo-scaffold-plan.md`
  (foundation that scaffolded `operator/api/v1alpha1`, `deploy/`,
  Makefile `controller-gen` invocations)
- `docs/plans/2026-05-20-002-feat-configuration-strategies-plan.md`
  (IR shape: RulesTree, Rule, Predicate)
- `docs/plans/2026-05-20-003-feat-api-grpc-openapi-plan.md`
  (Connect surface, segment inlining, audit attribution pattern)
- `operator/api/v1alpha1/project_types.go:1` (existing placeholder)
- `operator/controllers/project_controller.go:1` (existing
  placeholder; removed in this slice)
- `deploy/crds/falseflag.dev_projects.yaml` (existing CRD; modified
  to namespaced)
- `cmd/falseflag-api/main.go:1` (buildinfo + run wrapper pattern)
- `internal/buildinfo/buildinfo.go` (`WithGracefulShutdown` helper)
- `internal/logging/logging.go` (`New(suffix)` constructor)
- `internal/appconfig/appconfig.go` (`LoadAPI()` pattern)
- `internal/gen/proto/falseflag/v1/falseflagv1connect/*.go`
  (generated Connect clients)
- `internal/config/predicate.go` (`ValidatePredicate` exported)
- `Makefile:62-63` (controller-gen invocations)
- `docker-bake.hcl` (existing targets)

### External References

- Kubebuilder docs: <https://book.kubebuilder.io/>
- controller-runtime v0.24.x docs:
  <https://pkg.go.dev/sigs.k8s.io/controller-runtime>
- envtest setup:
  <https://book.kubebuilder.io/reference/envtest.html>
- Helm chart conventions:
  <https://helm.sh/docs/topics/charts/>
- Kustomize:
  <https://kubectl.docs.kubernetes.io/references/kustomize/>
- kind quickstart:
  <https://kind.sigs.k8s.io/docs/user/quick-start/>

### Inspiration

- `/Users/wito/code/project-keat/keat-server` — legacy Keat
  Kubernetes shape (read-only)
- `/Users/wito/code/project-depot/registry` — operator/api split
  reference for the Go layout

### Related Slices

- Slice 1 (`2026-05-20-001`) — Foundation scaffolding
- Slice 2 (`2026-05-20-002`) — IR types this slice translates to
- Slice 3 (`2026-05-20-003`) — Connect API this slice consumes
- Slice 5 (`2026-05-20-005`) — Dashboard exposes CR status
- Slice 7 (`2026-05-20-007`) — CI wires envtest into the slow path
