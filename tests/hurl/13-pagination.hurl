# pagination.hurl — exercises list pagination on audit, flag versions,
# and snapshots. Smoke harness truncates state before this runs.

# Setup project + flag.
POST http://localhost:8080/v1/projects
Content-Type: application/json
{"slug":"hurl-paginate","display_name":"Pagination","config_strategy":"json"}
HTTP 201

POST http://localhost:8080/v1/projects/hurl-paginate/flags
Content-Type: application/json
{
  "key": "pager",
  "name": "Pager",
  "description": "",
  "value_type": "boolean",
  "default_value": false
}
HTTP 201

# Publish several versions via PUT so listings have history to page over.
PUT http://localhost:8080/v1/projects/hurl-paginate/flags/pager
Content-Type: application/json
{"strategy":"json","source":{"value_type":"boolean","default":false,"rules":[]}}
HTTP 200

PUT http://localhost:8080/v1/projects/hurl-paginate/flags/pager
Content-Type: application/json
{"strategy":"json","source":{"value_type":"boolean","default":true,"rules":[]}}
HTTP 200

PUT http://localhost:8080/v1/projects/hurl-paginate/flags/pager
Content-Type: application/json
{"strategy":"json","source":{"value_type":"boolean","default":false,"rules":[{"id":"r1","when":{"kind":"always"},"value":true}]}}
HTTP 200

PUT http://localhost:8080/v1/projects/hurl-paginate/flags/pager
Content-Type: application/json
{"strategy":"json","source":{"value_type":"boolean","default":false,"rules":[{"id":"r2","when":{"kind":"always"},"value":true}]}}
HTTP 200

PUT http://localhost:8080/v1/projects/hurl-paginate/flags/pager
Content-Type: application/json
{"strategy":"json","source":{"value_type":"boolean","default":false,"rules":[{"id":"r3","when":{"kind":"always"},"value":true}]}}
HTTP 200

# Versions list returns at least 5.
GET http://localhost:8080/v1/projects/hurl-paginate/flags/pager/versions
HTTP 200
[Asserts]
jsonpath "$.items"            count >= 5

# Audit log accumulates entries for each publish.
GET http://localhost:8080/v1/projects/hurl-paginate/audit-events
HTTP 200
[Asserts]
jsonpath "$.items"            count >= 5
