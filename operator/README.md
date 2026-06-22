# FalseFlag Kubernetes Operator

Reconciles FalseFlag CRDs into the slice-3 Connect API. The operator
is a *control-plane consumer* — every change to a CR turns into one or
more Connect RPCs against the API on `:8090`. The API stays the single
source of truth; Kubernetes is the declarative input.

## Layout

- `operator/api/v1alpha1/**` — CRD Go types. controller-gen generates
  `zz_generated.deepcopy.go` and `deploy/crds/*.yaml` from these.
- `internal/operator/run.go` — manager bootstrap.
- `internal/operator/clientapi/` — Connect API client bundle with the
  `X-Actor` interceptor.
- `internal/operator/controllers/` — seven reconcilers, one per CRD.
- `internal/operator/translate/` — pure-Go CR → IR translation.

## CRDs

| Kind            | Scope      | Notes                                        |
| --------------- | ---------- | -------------------------------------------- |
| `Project`       | Namespaced | spec.projectSlug identifies the upstream     |
| `Environment`   | Namespaced | child of a Project, no finalizer             |
| `Segment`       | Namespaced | finalizer; predicate inlines at publish time |
| `RolloutPolicy` | Namespaced | reference-only; inlines into Flag publishes  |
| `Flag`          | Namespaced | finalizer; publishes one version per binding |
| `FlagBinding`   | Namespaced | fans out PublishFlagVersion per environment  |
| `FlagSnapshot`  | Namespaced | read-only; polls GetLatestSnapshot           |

## Quickstart

```bash
make up                                # API + Postgres + dashboard
kind create cluster
kubectl apply -k deploy/kustomize/overlays/dev
kubectl apply -k deploy/samples
kubectl get projects.falseflag.dev -A
```

## Environment variables

| Variable                          | Default                                                       |
| --------------------------------- | ------------------------------------------------------------- |
| `FALSEFLAG_API_BASE_URL`          | `http://falseflag-api.default.svc.cluster.local:8090`         |
| `FALSEFLAG_OPERATOR_ACTOR`        | `controller/falseflag-operator`                               |
| `FALSEFLAG_OPERATOR_METRICS_ADDR` | `:8082`                                                       |
| `FALSEFLAG_OPERATOR_HEALTH_ADDR`  | `:8083`                                                       |
| `FALSEFLAG_OPERATOR_LEADER_ELECT` | `false`                                                       |

## Demo-quality omissions

- No leader election; single replica only.
- No watches between CRs — the 30s requeue covers cross-CR updates.
- Finalizers exist but the API has no Delete RPCs for flags/segments,
  so CR deletion only removes the finalizer (the upstream row stays).
- ClusterRole is permissive across `falseflag.dev/*` resources.

## Tests

Per-controller tests use `sigs.k8s.io/controller-runtime/pkg/client/fake`
so plain `go test ./...` covers reconciliation. Run them with:

```bash
go test ./internal/operator/...
```

There is no envtest dependency in slice 4.
