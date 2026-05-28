# evaluate.hurl — exercises /evaluate against published versions of
# each strategy. Depends on flags.hurl having seeded the projects.

# JSON strategy: pro user → treatment.
POST http://localhost:8080/v1/projects/hurl-flags-json/flags/checkout/evaluate
Content-Type: application/json
{"context":{"user":{"plan":"pro"}}}
HTTP 200
[Asserts]
jsonpath "$.reason"  == "default"
jsonpath "$.value"   == true

# (The latest checkout version is v2 which has no rules and default=true.)

# CEL strategy: US, age 25 → premium.
POST http://localhost:8080/v1/projects/hurl-flags-cel/flags/premium/evaluate
Content-Type: application/json
{"context":{"user":{"age":25,"country":"US"}}}
HTTP 200
[Asserts]
jsonpath "$.value"   == "premium"
jsonpath "$.reason"  == "rule_matched"
jsonpath "$.rule_id" == "r1"
jsonpath "$.version" == 1

# CEL strategy: CA → default basic.
POST http://localhost:8080/v1/projects/hurl-flags-cel/flags/premium/evaluate
Content-Type: application/json
{"context":{"user":{"age":25,"country":"CA"}}}
HTTP 200
[Asserts]
jsonpath "$.value"  == "basic"
jsonpath "$.reason" == "default"

# CEL strategy: missing age → default basic (CEL eval errors are
# treated as predicate mismatches per the demo's leniency policy).
POST http://localhost:8080/v1/projects/hurl-flags-cel/flags/premium/evaluate
Content-Type: application/json
{"context":{"user":{"country":"US"}}}
HTTP 200
[Asserts]
jsonpath "$.value" == "basic"

# TypeScript strategy: prefers dark → {mode: dark}.
POST http://localhost:8080/v1/projects/hurl-flags-ts/flags/theme/evaluate
Content-Type: application/json
{"context":{"user":{"prefers":"dark"}}}
HTTP 200
[Asserts]
jsonpath "$.value.mode" == "dark"
jsonpath "$.reason"     == "rule_matched"

# TypeScript strategy: prefers light → default {mode: light}.
POST http://localhost:8080/v1/projects/hurl-flags-ts/flags/theme/evaluate
Content-Type: application/json
{"context":{"user":{"prefers":"light"}}}
HTTP 200
[Asserts]
jsonpath "$.value.mode" == "light"
jsonpath "$.reason"     == "default"

# Unknown flag → 404.
POST http://localhost:8080/v1/projects/hurl-flags-json/flags/missing/evaluate
Content-Type: application/json
{"context":{}}
HTTP 404
