# evaluate-trace.hurl — exercises the explained-evaluation endpoint
# against the project seeded by 02-flags.hurl (hurl-flags-cel).

POST http://localhost:8080/v1/projects/hurl-flags-cel/flags/premium/evaluate-trace
Content-Type: application/json
{"context":{"user":{"age":25,"country":"US"}}}
HTTP 200
[Asserts]
jsonpath "$.decision.value"           == "premium"
jsonpath "$.decision.reason"          == "rule_matched"
jsonpath "$.decision.rule_id"         == "r1"
jsonpath "$.trace.default_used"       == false
jsonpath "$.trace.matched_rule_id"    == "r1"
jsonpath "$.trace.evaluated_rules"    count == 1
jsonpath "$.trace.evaluated_rules[0].matched"        == true
jsonpath "$.trace.evaluated_rules[0].predicate.kind" == "cel"

POST http://localhost:8080/v1/projects/hurl-flags-cel/flags/premium/evaluate-trace
Content-Type: application/json
{"context":{"user":{"age":17,"country":"US"}}}
HTTP 200
[Asserts]
jsonpath "$.decision.value"      == "basic"
jsonpath "$.decision.reason"     == "default"
jsonpath "$.trace.default_used"  == true
jsonpath "$.trace.evaluated_rules[0].matched" == false

# Unknown flag → 404.
POST http://localhost:8080/v1/projects/hurl-flags-cel/flags/does-not-exist/evaluate-trace
Content-Type: application/json
{"context":{}}
HTTP 404
