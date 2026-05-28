# projects.hurl — exercises the /v1/projects surface end-to-end.
#
# Assumes a clean DB; scripts/smoke.sh runs TRUNCATE before invoking
# this file. Re-running against a non-truncated DB will fail the
# duplicate-slug assertion.

# Create project A
POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-projects-a","display_name":"Project A","config_strategy":"json"}
HTTP 201
[Asserts]
jsonpath "$.slug"            == "hurl-projects-a"
jsonpath "$.display_name"    == "Project A"
jsonpath "$.config_strategy" == "json"
jsonpath "$.id"              matches "^[0-9a-f-]{36}$"

# Duplicate slug → 409
POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-projects-a","display_name":"Conflict","config_strategy":"json"}
HTTP 409
[Asserts]
jsonpath "$.error" exists

# Create project B (cel strategy)
POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-projects-b","display_name":"Project B","config_strategy":"cel"}
HTTP 201

# Get a project by slug
GET http://localhost:8080/v1/projects/hurl-projects-a
HTTP 200
[Asserts]
jsonpath "$.slug" == "hurl-projects-a"

# 404 for unknown slug
GET http://localhost:8080/v1/projects/does-not-exist
HTTP 404

# List returns both
GET http://localhost:8080/v1/projects
HTTP 200
[Asserts]
jsonpath "$.items"        count >= 2
jsonpath "$.items[*].slug" includes "hurl-projects-a"
jsonpath "$.items[*].slug" includes "hurl-projects-b"

# Invalid strategy rejected
POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-projects-c","display_name":"Bad","config_strategy":"ruby"}
HTTP 400
