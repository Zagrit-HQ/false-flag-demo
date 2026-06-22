# internal/operator

Manager bootstrap, Connect API client wrapper, and reconciler tree
for the FalseFlag Kubernetes operator.

## Layout

- `run.go` — registers the v1alpha1 scheme, builds the manager, and
  wires every reconciler. Exposed as `operator.Run` and invoked by
  `cmd/falseflag-operator/main.go`.
- `clientapi/` — leaf package holding the bundled Connect service
  clients with an `X-Actor` outbound interceptor. Reconcilers depend
  on it; `operator` and `controllers` both import it.
- `controllers/` — seven reconcilers (one per CRD) plus the shared
  `result.go` error-translation helper and condition setters.
- `translate/` — pure-Go CR → IR translation. Has its own unit tests;
  no Kubernetes dependencies, so it's the fastest part of the suite.

## Reconciler conventions

- Always Get the CR; on NotFound return without requeue.
- For Flag/Segment: handle finalizer before any business logic.
- Apply spec → IR translation, then call the API.
- Map upstream errors via `translateError` (centralizes the
  Connect-code → condition translation).
- Update status conditions + observed generation + last sync time
  whenever the upstream call succeeds.
- Default success requeue is 30s; conflicts use 1s backoff.

## Adding a new CRD

1. Add the type under `operator/api/v1alpha1/`.
2. Register it in `operator/api/v1alpha1/groupversion_info.go`.
3. Run `make generate` to regenerate `zz_generated.deepcopy.go` and
   the CRD YAML in `deploy/crds/`.
4. Add the reconciler file under `internal/operator/controllers/`.
5. Wire `SetupWithManager` into `internal/operator/run.go`.
6. Add a test using `sigs.k8s.io/controller-runtime/pkg/client/fake`.
