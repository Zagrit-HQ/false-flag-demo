# FalseFlag Snapshot Canonical Output Format

A snapshot is the project-wide `{flags: {[key]: RulesTree}}` bundle the
API compiles from the latest published flag versions. The API stores
and returns it as JSON without any normalization beyond what the
JSON.Marshal / encoding/json defaults emit.

The CLI's `falseflag snapshot export` command writes a **canonical**
form suitable for diffs, demo screenshots, and reproducible artifacts.

## Canonical Rules

1. **Sorted keys.** The top-level `flags` map is emitted with keys in
   lexicographic order. Within each `RulesTree`, the `rules` array
   keeps its original order (rule order is semantically meaningful;
   first match wins).
2. **2-space indentation.** Pretty-printed with `json.MarshalIndent` /
   `JSON.stringify(obj, null, 2)`.
3. **Trailing newline.** Files always end with a single `\n` so they
   diff cleanly under `git diff`.
4. **UTF-8 without BOM.**
5. **YAML alternative.** `--format yaml` emits YAML with block-style
   maps (no flow style), keys sorted at every depth.

## Example

```json
{
  "snapshot": {
    "id": "9f64f0b9-...-...",
    "project_id": "01J5...",
    "version": 42,
    "created_at": "2026-05-20T14:33:11Z"
  },
  "compiled": {
    "flags": {
      "checkout-redesign": {
        "default": false,
        "rules": [
          {
            "id": "rule-1",
            "value": true,
            "when": { "kind": "string_in", "attribute": "user.plan", "values": ["pro"] }
          }
        ],
        "value_type": "boolean"
      },
      "free-trial-days": {
        "default": 7,
        "rules": [],
        "value_type": "number"
      }
    }
  }
}
```

The top-level envelope adds `snapshot` metadata (id / version /
created_at) alongside the raw `compiled` blob. This keeps the export
self-describing without losing the bytes the SDK consumes.

## Why This Document Exists

Phases 5 (CLI export) and 7 (golden corpus) both need to agree on the
exact bytes-on-disk form. Without a canonical format, every developer
working on slice 5 would risk producing inconsistent fixture files.
