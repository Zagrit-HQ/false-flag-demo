# Kustomize

Kustomize overlays for the FalseFlag operator.

## Layout

- `base/` — namespace, CRDs (bundled into `crds.yaml`), RBAC,
  ConfigMap, Deployment. Targets the `falseflag-system` namespace.
- `overlays/dev/` — patches the ConfigMap so the operator points at
  `host.docker.internal:8090` for local kind clusters.

## Apply

```bash
kubectl apply -k deploy/kustomize/overlays/dev
```

Regenerate the bundled CRDs after touching `operator/api/v1alpha1`:

```bash
make generate
( cat deploy/kustomize/base/crds.yaml | head -10; \
  for f in deploy/crds/*.yaml; do echo "---"; cat "$f"; done ) > \
  deploy/kustomize/base/crds.yaml.new
mv deploy/kustomize/base/crds.yaml.new deploy/kustomize/base/crds.yaml
```

This is wired into `make generate` indirectly — the bundled CRDs are
checked into git, so `make generate-check` catches drift.
