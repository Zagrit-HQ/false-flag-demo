# connect-smoke.hurl — Connect-over-JSON happy paths against the
# second listener on :8090. Each request uses
# Content-Type: application/json so the connect-go server selects the
# JSON codec (binary protobuf works too but is opaque from curl).

POST http://localhost:8090/falseflag.v1.HealthService/Check
Content-Type: application/json
{}
HTTP 200
[Asserts]
jsonpath "$.status" == "SERVING_STATUS_SERVING"

POST http://localhost:8090/falseflag.v1.ProjectsService/ListProjects
Content-Type: application/json
{}
HTTP 200
[Asserts]
# At least the seeded projects from 01-projects/02-flags/04-08 must exist.
jsonpath "$.items" exists

POST http://localhost:8090/falseflag.v1.ProjectsService/GetProject
Content-Type: application/json
{"slug":"hurl-snaps"}
HTTP 200
[Asserts]
jsonpath "$.project.slug" == "hurl-snaps"

# Not-found maps to Connect's not_found code over JSON.
POST http://localhost:8090/falseflag.v1.ProjectsService/GetProject
Content-Type: application/json
{"slug":"does-not-exist"}
HTTP 404
[Asserts]
jsonpath "$.code" == "not_found"

POST http://localhost:8090/falseflag.v1.SnapshotsService/GetLatestSnapshot
Content-Type: application/json
{"projectSlug":"hurl-snaps"}
HTTP 200
[Asserts]
jsonpath "$.snapshot.version" >= 1
