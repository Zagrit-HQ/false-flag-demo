# falseflag-operator

Entrypoint for the FalseFlag Kubernetes operator. The binary is a
16-line wrapper around `internal/operator.Run`, which boots the
controller-runtime manager and registers seven reconcilers.

See `operator/README.md` for the full operator guide.

## Run locally

```bash
go run ./cmd/falseflag-operator
```

Reads kubeconfig from `$KUBECONFIG` (default `~/.kube/config`); if no
cluster is reachable the binary logs a warning and blocks on the
context, exiting cleanly on SIGINT/SIGTERM. Talks to the API at
`$FALSEFLAG_API_BASE_URL`.
