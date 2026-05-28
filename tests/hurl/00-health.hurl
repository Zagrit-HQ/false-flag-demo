# Smoke check the FalseFlag control-plane API health probes. scripts/smoke.sh
# brings up cmd/falseflag-api before invoking hurl.

GET http://localhost:8080/healthz
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.status"  == "ok"
jsonpath "$.service" == "falseflag-api"
jsonpath "$.probe"   == "healthz"

GET http://localhost:8080/readyz
HTTP 200
[Asserts]
jsonpath "$.status"  == "ok"
jsonpath "$.probe"   == "readyz"

GET http://localhost:8080/v1/health
HTTP 200
[Asserts]
jsonpath "$.status"  == "ok"
jsonpath "$.probe"   == "v1.health"
