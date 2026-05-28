# FalseFlag Cross-Runtime Evaluation Corpus

Every `.json` file here is one golden fixture asserted by:

- `internal/eval/cross_runtime_test.go` (Go evaluator)
- `js/packages/shared-eval-corpus/src/cross-runtime.test.ts` (JS evaluator)

Both runtimes must produce identical Decisions for every fixture.
This is the single canonical contract for "JSON, CEL, and TypeScript
strategies share one runtime" — if the two evaluators ever drift,
this suite is what catches it.

## Fixture shape

```jsonc
{
  "id":          "stable-kebab-case-identifier",
  "description": "Human-readable one-liner.",
  "strategy":    "json | cel | typescript",
  "ir": {
    "value_type": "boolean | string | number | object",
    "default":    <value>,
    "rules":      [ /* see internal/config IR */ ]
  },
  "context":  { /* arbitrary eval context */ },
  "expected": {
    "value":   <value>,
    "reason":  "default | rule_matched | rollout_in_bucket | rollout_out_of_bucket | type_mismatch | error",
    "rule_id": "<optional, omitted for default>"
  }
}
```

## Adding a fixture

1. Write the file with a stable `id` (used in test names).
2. Run `make test` — both runtimes must agree.
3. If a fixture exercises CEL operators outside the demo subset,
   the JS evaluator may need its CEL-lite extended first.
