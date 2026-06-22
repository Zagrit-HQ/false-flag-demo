# Sample CRs

End-to-end Project → FlagSnapshot example manifests.

Apply order matters: Project must exist upstream before the child CRs
can reconcile, but Kustomize applies these in the order listed in
`kustomization.yaml`. The operator's per-reconcile requeue handles
the eventual consistency.

```bash
kubectl apply -k deploy/samples
kubectl wait --for=condition=Ready -n default projects.falseflag.dev/demo --timeout=60s
kubectl wait --for=condition=Published -n default flagbindings.falseflag.dev/banner-binding --timeout=60s
```

After reconciliation, the FalseFlag API has:

- project `demo` (config strategy: json)
- environments `prod` and `staging`
- segment `internal-users` matching `.*@depot\.dev$`
- flag `banner` (boolean, default false, always-on rule)
- flag `checkout-v2` (boolean, default false, gradual-launch rollout)
- two flag versions per environment for `banner` (via FlagBinding)
- the latest snapshot version reflected in `flagsnapshots.falseflag.dev/demo-latest`
