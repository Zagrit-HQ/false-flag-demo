# Helm

Helm chart for the FalseFlag Kubernetes operator.

## Install

```bash
helm install falseflag-operator deploy/helm/falseflag-operator \
  --namespace falseflag-system \
  --create-namespace
```

The chart installs the seven CRDs from `deploy/helm/falseflag-operator/crds/`
on first install. Helm does **not** manage CRD lifecycle on upgrade —
to upgrade the CRDs, re-apply them with `kubectl apply -f
deploy/crds/`.

## Configuration

| Key                | Default                                                       |
| ------------------ | ------------------------------------------------------------- |
| `image.repository` | `ghcr.io/depot/falseflag/operator`                            |
| `image.tag`        | `dev`                                                         |
| `api.baseURL`      | `http://falseflag-api.default.svc.cluster.local:8090`         |
| `api.actor`        | `controller/falseflag-operator` — appears in audit log records |
| `probes.metricsAddr` | `:8082`                                                     |
| `probes.healthAddr`  | `:8083`                                                     |

## Uninstall

```bash
helm uninstall falseflag-operator -n falseflag-system
# CRDs are intentionally retained — delete manually if desired:
kubectl delete -f deploy/crds/
```
