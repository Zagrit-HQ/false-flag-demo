# segments.hurl — project-scoped segment CRUD.

POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-segments","display_name":"Segments","config_strategy":"json"}
HTTP 201

POST http://localhost:8080/v1/projects/hurl-segments/segments
Content-Type: application/json
X-Actor: hurl
{
  "key": "beta-users",
  "name": "Beta Users",
  "description": "internal beta cohort",
  "predicate": {"kind":"eq","attr":"user.beta","value":true}
}
HTTP 201
[Asserts]
jsonpath "$.key"  == "beta-users"
jsonpath "$.name" == "Beta Users"

GET http://localhost:8080/v1/projects/hurl-segments/segments/beta-users
HTTP 200
[Asserts]
jsonpath "$.predicate.kind" == "eq"
jsonpath "$.predicate.attr" == "user.beta"

PUT http://localhost:8080/v1/projects/hurl-segments/segments/beta-users
Content-Type: application/json
X-Actor: hurl
{
  "name": "Internal Beta",
  "description": "internal beta cohort",
  "predicate": {"kind":"in","attr":"user.cohort","values":["alpha","beta"]}
}
HTTP 200
[Asserts]
jsonpath "$.predicate.kind" == "in"

# Invalid predicate → 400 (rollout missing salt).
POST http://localhost:8080/v1/projects/hurl-segments/segments
Content-Type: application/json
{
  "key": "bad",
  "name": "Bad",
  "predicate": {"kind":"rollout","attr":"user.id","percent":50}
}
HTTP 400

GET http://localhost:8080/v1/projects/hurl-segments/segments
HTTP 200
[Asserts]
jsonpath "$.items" count == 1
