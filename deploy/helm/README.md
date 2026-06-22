# Helm

Helm chart for the FalseFlag demo stack.

By default the chart deploys:

- Postgres
- API REST + ConnectRPC service
- evaluation proxy
- MCP server
- Remix dashboard

The Kubernetes operator is opt-in and disabled by default.

## Install

```bash
helm install falseflag deploy/helm/falseflag-operator \
  --namespace falseflag-system \
  --create-namespace
```

Enable the operator when you want CRD reconciliation:

```bash
helm upgrade --install falseflag deploy/helm/falseflag-operator \
  --namespace falseflag-system \
  --create-namespace \
  --set operator.enabled=true
```

The chart installs the seven CRDs from `deploy/helm/falseflag-operator/crds/`
on first install. Helm does **not** manage CRD lifecycle on upgrade;
to upgrade the CRDs, re-apply them with `kubectl apply -f deploy/crds/`.

## Configuration

| Key | Default |
| --- | --- |
| `global.imageTag` | `dev` |
| `postgresql.enabled` | `true` |
| `api.enabled` | `true` |
| `proxy.enabled` | `true` |
| `mcp.enabled` | `true` |
| `dashboard.enabled` | `true` |
| `operator.enabled` | `false` |
| `operator.actor` | `controller/falseflag-operator` |

For an external database, disable the in-chart Postgres and set the API
DSN:

```bash
helm upgrade --install falseflag deploy/helm/falseflag-operator \
  --namespace falseflag-system \
  --set postgresql.enabled=false \
  --set-string api.databaseURL='postgres://falseflag:falseflag@postgres:5432/falseflag?sslmode=disable'
```

## Uninstall

```bash
helm uninstall falseflag -n falseflag-system
# CRDs are intentionally retained; delete manually if desired:
kubectl delete -f deploy/crds/
```
