# 11-proxy-evaluate.hurl — exercises the falseflag-proxy slice 5 surface.
#
# Assumes:
#   - falseflag-api running on :8080 (slice 3)
#   - falseflag-proxy running on :8081 with
#       FALSEFLAG_PROXY_PROJECT_SLUG=demo
#     (compose wires this in Phase 9)
#   - The demo project has at least one published flag named
#     `proxy-smoke-bool` returning a boolean (see db/seed/...).
#
# All requests should land 200 against a healthy proxy. The body
# values themselves aren't asserted line-by-line because the seed
# dataset's rule outcomes can evolve; we just confirm the proxy is
# alive, reports a snapshot, and answers /v1/evaluate.

# 1. liveness
GET http://localhost:8081/healthz
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"
jsonpath "$.service" == "falseflag-proxy"

# 2. readiness — should be ok once first poll has landed
GET http://localhost:8081/readyz
HTTP 200
[Asserts]
jsonpath "$.status" == "ok"

# 3. snapshot info exposes the loaded snapshot
GET http://localhost:8081/v1/snapshot
HTTP 200
[Asserts]
jsonpath "$.version" >= 1
jsonpath "$.flag_count" >= 0

# 4. evaluate returns a decision shape (value/reason/version)
POST http://localhost:8081/v1/evaluate
Content-Type: application/json
{
  "key": "proxy-smoke-bool",
  "default_value": false,
  "context": {"user": {"id": "u-smoke", "plan": "pro"}}
}
HTTP 200
[Asserts]
jsonpath "$.reason" exists
jsonpath "$.version" exists
