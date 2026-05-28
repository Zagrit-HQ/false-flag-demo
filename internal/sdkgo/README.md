# internal/sdkgo

The in-process Go SDK for FalseFlag. It polls
`/v1/projects/{slug}/snapshots/latest` over plain HTTP and evaluates
flags locally using the same `internal/eval` evaluator the server runs
— so a decision returned here is byte-identical to one returned by the
API's `/evaluate` endpoint.

## Quick start

```go
client, err := sdkgo.NewClient(sdkgo.Options{
    BaseURL:     "http://localhost:8080",
    ProjectSlug: "acme-web",
})
if err != nil { /* ... */ }
if err := client.Start(ctx); err != nil { /* ... */ }
defer client.Stop()

provider := sdkgo.NewProvider(client, "")

d := provider.BooleanEvaluation(ctx, "checkout-redesign", false, sdkgo.EvalContext{
    "user": map[string]any{"id": "u-42", "plan": "pro"},
})
fmt.Println(d.Value, d.Reason)
```

## Polling

`Start(ctx)` runs one synchronous poll, then spawns a goroutine that
polls again every `PollInterval` (default `10s`). Set
`PollInterval < 0` to disable background polling — the SDK still does
one poll from `Start`.

When a poll fails (network, 5xx), the SDK logs a warning via the
configured `slog.Logger` and keeps the last good snapshot. Evaluations
continue to be served from cache.

If the first `Start` poll fails, `Start` returns the error and the
SDK has no snapshot. All subsequent `Evaluate` calls return
`Decision{Reason: "error"}` until a later poll succeeds.

## OpenFeature shape

This SDK implements four resolution methods (`BooleanEvaluation`,
`StringEvaluation`, `NumberEvaluation`, `ObjectEvaluation`) that match
the OpenFeature provider surface in spirit but not in spec letter. No
hooks, no finally stage, no provider registry. See
[`docs/sdk-openfeature.md`](../../docs/sdk-openfeature.md).

## Demo-quality posture

- No retries with backoff. The next tick simply tries again.
- No authentication header. The control plane in slice 5 has no auth.
- No metrics. The slice 7 CI/Depot demo doesn't need them.
- No long-poll, no SSE — plain interval polling.
